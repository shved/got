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

type ObjectType int

const (
	Commit ObjectType = iota + 1
	Tree
	Blob
)

type Object struct {
	ObjType          ObjectType
	Parent           *Object
	Children         []*Object
	Name             string
	ParentPath       string
	Path             string
	ParentCommitHash string

	sha          []byte
	contentLines []string
	gzipContent  string
	parentCommit *Object // TODO redundant???
}

func Show(shaString string) string {
	if exists(path.Join(got.CommitDirAbsPath(), shaString)) {
		return objContent(path.Join(got.CommitDirAbsPath(), shaString))
	}

	if exists(path.Join(got.TreeDirAbsPath(), shaString)) {
		return objContent(path.Join(got.TreeDirAbsPath(), shaString))
	}

	if exists(path.Join(got.TreeDirAbsPath(), shaString)) {
		return objContent(path.Join(got.TreeDirAbsPath(), shaString))
	}

	log.Fatal(got.ErrObjDoesNotExist)
	panic("never reach")
}

func (o *Object) RecRestoreFromObject(p string) {
	switch o.ObjType {
	case Commit:
		for _, ch := range o.Children {
			ch.RecRestoreFromObject(p)
		}
		got.UpdateHead(string(o.sha))
	case Tree:
		treePath := path.Join(p, o.Name)
		err := os.Mkdir(treePath, 0755)
		if err != nil {
			log.Fatal(err)
		}
		for _, ch := range o.Children {
			ch.RecRestoreFromObject(treePath)
		}
	case Blob:
		blobPath := path.Join(p, o.Name)
		if err := ioutil.WriteFile(blobPath, []byte(o.gzipContent), 0644); err != nil {
			log.Fatal(err)
		}
	}
}

func objContent(p string) string {
	res, _ := readArchive(p)
	return string(res)
}

func (t ObjectType) toString() string {
	switch t {
	case Commit:
		return "commit"
	case Tree:
		return "tree"
	case Blob:
		return "blob"
	default:
		log.Fatal(got.ErrInvalidObjType)
		panic("never reach")
	}
}

func strToObjType(s string) ObjectType {
	switch s {
	case "commit":
		return Commit
	case "tree":
		return Tree
	case "blob":
		return Blob
	default:
		log.Fatal(got.ErrInvalidObjType)
		panic("never reach")
	}
}

func RecReadObject(t ObjectType, hashString string, parentObj *Object) *Object {
	switch t {
	case Commit:
		oPath := path.Join(t.storePath(), hashString)
		res, header := readArchive(oPath)
		commit := &Object{ObjType: Commit, Name: header.Name, sha: []byte(hashString)}
		log.Println(commit)
		children := parseObjContent(string(res))
		for _, child := range children {
			if child.t != Commit {
				commit.Children = append(commit.Children, RecReadObject(child.t, child.hashString, commit))
			} else {
				continue // skip parent commit entry in commit content
			}
		}
		return commit
	case Tree:
		oPath := path.Join(t.storePath(), hashString)
		res, header := readArchive(oPath)
		tree := &Object{ObjType: Tree, Name: header.Name, sha: []byte(hashString), Parent: parentObj}
		children := parseObjContent(string(res))
		log.Println(tree)
		for _, child := range children {
			tree.Children = append(tree.Children, RecReadObject(child.t, child.hashString, tree))
		}
		return tree
	case Blob:
		oPath := path.Join(t.storePath(), hashString)
		res, header := readArchive(oPath)
		blob := &Object{ObjType: Blob, Name: header.Name, sha: []byte(hashString), Parent: parentObj, gzipContent: string(res)}
		log.Println(blob)
		return blob
	default:
		log.Fatal(got.ErrInvalidObjType)
	}
	panic("never reach")
}

func readArchive(p string) ([]byte, gzip.Header) {
	fd, err := os.Open(p)
	unarchiver, _ := gzip.NewReader(fd)
	defer fd.Close()
	defer unarchiver.Close()
	res, err := ioutil.ReadAll(unarchiver)
	if err != nil {
		log.Fatal(err)
	}
	return res, unarchiver.Header
}

func parseObjContent(s string) []objRepr {
	var objects []objRepr
	lines := strings.Split(s, "\n")
	for _, line := range lines {
		objects = append(objects, parseObjString(line))
	}
	return objects
}

func parseObjString(s string) objRepr {
	entries := strings.Split(s, "\t")
	var name string
	if len(entries) > 3 {
		name = entries[2]
	}
	return objRepr{t: strToObjType(entries[0]), hashString: entries[1], name: name}
}

type objRepr struct {
	t          ObjectType
	hashString string
	name       string
}

func (t ObjectType) storePath() string {
	switch t {
	case Commit:
		return got.CommitDirAbsPath()
	case Tree:
		return got.TreeDirAbsPath()
	case Blob:
		return got.BlobDirAbsPath()
	default:
		log.Fatal(got.ErrInvalidObjType)
	}
	panic("never reach")
}

// recursive calculate all objects sha1 started from very far children
func (o *Object) RecCalcHashSum() {
	switch o.ObjType {
	case Commit:
		for _, ch := range o.Children {
			ch.RecCalcHashSum()
			o.contentLines = append(o.contentLines, ch.buildContentLineForParent())
		}
		parentCommitLine := parentCommitShaContentLine(o.ParentCommitHash)
		o.contentLines = append(o.contentLines, parentCommitLine)
		sort.Strings(o.contentLines)
		o.gzipContent = strings.Join(o.contentLines, "\n")
		data := []byte(o.gzipContent)
		h := sha1.New()
		h.Write(data)
		o.sha = h.Sum(nil)
	case Tree:
		for _, ch := range o.Children {
			ch.RecCalcHashSum()
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

func parentCommitShaContentLine(parentSha string) string {
	// TODO add real commit message when it will be unpacked as a worktree object
	entries := []string{Commit.toString(), parentSha, "here will be commit message"}
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
		writeArchive(path, o.Name, []byte(o.gzipContent))
		got.UpdateHead(hashString(o.sha))
	case Tree:
		path := path.Join(got.TreeDirAbsPath(), hashString(o.sha))
		if exists(path) {
			break
		}
		writeArchive(path, o.Name, []byte(o.gzipContent))
	case Blob:
		path := path.Join(got.BlobDirAbsPath(), hashString(o.sha))
		if exists(path) {
			break
		}
		data, err := ioutil.ReadFile(o.Path)
		if err != nil {
			log.Fatal(err)
		}
		writeArchive(path, o.Name, data)
	default:
		log.Fatal(got.ErrInvalidObjType)
	}
}

func writeArchive(p string, name string, data []byte) {
	fd, _ := os.Create(p)
	archiver := gzip.NewWriter(fd)
	defer fd.Close()
	defer archiver.Close()
	archiver.Name = name
	archiver.ModTime = time.Now()
	archiver.Write(data)
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
