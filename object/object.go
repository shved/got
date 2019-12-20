package object

import (
	"compress/gzip"
	"crypto/sha1"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/shved/got/got"
)

type objectType int

const (
	Commit objectType = iota + 1
	Tree
	Blob
)

type Object struct {
	ObjType         objectType
	Parent          *Object
	Children        []*Object
	Name            string
	ParentPath      string
	Path            string
	ParentCommitSha []byte

	sha          []byte
	contentLines []string
	gzipContent  string
	parentCommit *Object
}

func (t objectType) toString() string {
	switch t {
	case Commit:
		return "commit"
	case Tree:
		return "tree"
	case Blob:
		return "blob"
	default:
		log.Fatal(got.ErrInvalidObjType)
		panic("not reached")
	}
}

// recursive calculate all objects sha1 started from very far children
func (o *Object) RecBuildHashSums() {
	switch o.ObjType {
	case Commit:
		for _, ch := range o.Children {
			ch.RecBuildHashSums()
			o.contentLines = append(o.contentLines, ch.buildContentLineForParent())
		}
		parentCommitLine := parentCommitShaContentLine(o.ParentCommitSha)
		o.contentLines = append(o.contentLines, parentCommitLine)
		sort.Strings(o.contentLines)
		o.gzipContent = strings.Join(o.contentLines, "\n")
		data := []byte(o.gzipContent)
		h := sha1.New()
		h.Write(data)
		o.sha = h.Sum(nil)
	case Tree:
		for _, ch := range o.Children {
			ch.RecBuildHashSums()
			o.contentLines = append(o.contentLines, ch.buildContentLineForParent())
		}
		sort.Strings(o.contentLines)
		o.gzipContent = strings.Join(o.contentLines, "\n")
		data := []byte(o.gzipContent)
		h := sha1.New()
		h.Write([]byte(o.Path))
		h.Write(data)
		o.sha = h.Sum(nil)
	case Blob:
		data, err := ioutil.ReadFile(o.Path)
		if err != nil {
			log.Fatal(err)
		}

		h := sha1.New()
		h.Write([]byte(o.Path))
		h.Write(data)
		o.sha = h.Sum(nil)
	default:
		log.Fatal(got.ErrInvalidObjType)
	}
}

func (o *Object) buildContentLineForParent() string {
	entries := []string{o.ObjType.toString(), hashString(o.sha), o.Name}
	return strings.Join(entries, "\t")
}

func parentCommitShaContentLine(parentSha []byte) string {
	// TODO add real commit message when it will be unpacked as a worktree object
	entries := []string{Commit.toString(), hashString(parentSha), "here will be commit message"}
	return strings.Join(entries, "\t")
}

func (o *Object) RecWriteObjects() {
	if o.ObjType == Commit || o.ObjType == Tree {
		for _, ch := range o.Children {
			ch.RecWriteObjects()
		}
	}

	o.write()
}

func (o *Object) write() {
	switch o.ObjType {
	case Commit:
		path := path.Join(got.CommitDirAbsPath(), hashString(o.sha))
		fd, _ := os.Create(path)
		archiver := gzip.NewWriter(fd)
		archiver.Name = o.Name
		archiver.ModTime = time.Now()
		archiver.Write([]byte(o.gzipContent))
		archiver.Close()
		got.UpdateHead(hashString(o.sha))
	case Tree:
		path := path.Join(got.TreeDirAbsPath(), hashString(o.sha))
		if exists(path) {
			break
		}
		fd, _ := os.Create(path)
		archiver := gzip.NewWriter(fd)
		archiver.Name = o.Name
		archiver.ModTime = time.Now()
		archiver.Write([]byte(o.gzipContent))
		archiver.Close()
	case Blob:
		path := path.Join(got.BlobDirAbsPath(), hashString(o.sha))
		if exists(path) {
			break
		}
		data, err := ioutil.ReadFile(o.Path)
		if err != nil {
			log.Fatal(err)
		}
		fd, _ := os.Create(path)
		archiver := gzip.NewWriter(fd)
		archiver.Name = o.Name
		archiver.ModTime = time.Now()
		archiver.Write(data)
		archiver.Close()
	default:
		log.Fatal(got.ErrInvalidObjType)
	}
}

func hashString(hashSum []byte) string {
	return fmt.Sprintf("%x", hashSum)
}

func exists(path string) bool {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}
