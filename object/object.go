// Package object includes all the functions and types related to the Got objects and operations on them.
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

// Object is a struct representation of a repo object.
type Object struct {
	ObjType          ObjectType
	Parent           *Object
	Children         []*Object
	Name             string
	ParentPath       string
	Path             string
	ParentCommitHash string
	CommitMessage    string
	HashString       string
	Timestamp        time.Time

	sha          []byte
	contentLines []string
	gzipContent  string
}

// Show returns a string with object content.
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

// objContent reads object archive and returns only its contentw without gzip headers.
func objContent(p string) string {
	res, _ := readArchive(p)
	return string(res)
}

// RecRestoreFromObject recursively writes objects into files/folders making an object graph
// persisted in a worktree.
func (o *Object) RecRestoreFromObject(p string) {
	switch o.ObjType {
	case Commit:
		for _, ch := range o.Children {
			ch.RecRestoreFromObject(p)
		}
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

// RecReadObject recursively reads objects archives and links them into an object graph.
func RecReadObject(t ObjectType, hashString string, parentObj *Object) *Object {
	switch t {
	case Commit:
		oPath := path.Join(t.storePath(), hashString)
		res, header := readArchive(oPath)
		commit := &Object{
			ObjType:       Commit,
			Name:          header.Name,
			sha:           []byte(hashString),
			HashString:    hashString,
			Timestamp:     header.ModTime,
			CommitMessage: header.Comment,
		}
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
		tree := &Object{
			ObjType:    Tree,
			Name:       header.Name,
			sha:        []byte(hashString),
			HashString: hashString,
			Parent:     parentObj,
			Timestamp:  header.ModTime,
		}
		children := parseObjContent(string(res))
		for _, child := range children {
			tree.Children = append(tree.Children, RecReadObject(child.t, child.hashString, tree))
		}
		return tree
	case Blob:
		oPath := path.Join(t.storePath(), hashString)
		res, header := readArchive(oPath)
		blob := &Object{
			ObjType:     Blob,
			Name:        header.Name,
			sha:         []byte(hashString),
			HashString:  hashString,
			Parent:      parentObj,
			gzipContent: string(res),
			Timestamp:   header.ModTime,
		}
		return blob
	default:
		log.Fatalf("RecReadObject(): %v", got.ErrInvalidObjType)
	}
	panic("never reach")
}

// objRepr is a local type to proceed object string representation for the further transformation intro and object.
type objRepr struct {
	t          ObjectType
	hashString string
	name       string
}

// parseObjContent takes object (commit or tree) contents and returns a slice of containing objects
// in a special representation form.
func parseObjContent(s string) []objRepr {
	var objects []objRepr
	lines := strings.Split(s, "\n")
	for _, line := range lines {
		objects = append(objects, parseObjString(line))
	}
	return objects
}

// parseObjString parses object string representation.
func parseObjString(s string) objRepr {
	entries := strings.Split(s, "\t")
	var name string
	if len(entries) > 3 {
		name = entries[2]
	}
	return objRepr{t: strToObjType(entries[0]), hashString: entries[1], name: name}
}

// storePath returns objects path to write into depending on its type.
func (t ObjectType) storePath() string {
	switch t {
	case Commit:
		return got.CommitDirAbsPath()
	case Tree:
		return got.TreeDirAbsPath()
	case Blob:
		return got.BlobDirAbsPath()
	default:
		log.Fatalf("storePath(): %v", got.ErrInvalidObjType)
	}
	panic("never reach")
}

// LogEntry function returns a string representation of a commit for repo commit log.
func (o *Object) LogEntry() string {
	if o.ObjType != Commit {
		log.Fatal(got.ErrWrongLogEntryType)
	}

	logEntry := strings.Join(
		[]string{
			o.Timestamp.UTC().Format(time.RFC3339),
			o.HashString,
			o.ParentCommitHash,
			o.CommitMessage,
		},
		"\t",
	)
	return logEntry + "\n"
}

// RecCalcHashSum recursively calculates all objects sha1 in an object graph started from very far children
// and puts it into the object struct fields sha and HashString.
func (o *Object) RecCalcHashSum() {
	switch o.ObjType {
	case Commit:
		for _, ch := range o.Children {
			ch.RecCalcHashSum()
			o.contentLines = append(o.contentLines, ch.buildContentLineForParent())
		}
		if o.ParentCommitHash != string(got.EmptyCommitRef) {
			parentCommitLine := parentCommitShaContentLine(o.ParentCommitHash)
			o.contentLines = append(o.contentLines, parentCommitLine)
		}
		sort.Strings(o.contentLines)
		o.gzipContent = strings.Join(o.contentLines, "\n")
		data := []byte(o.gzipContent)
		o.writeShaSum(data)
	case Tree:
		for _, ch := range o.Children {
			ch.RecCalcHashSum()
			o.contentLines = append(o.contentLines, ch.buildContentLineForParent())
		}
		sort.Strings(o.contentLines)
		o.gzipContent = strings.Join(o.contentLines, "\n")
		data := []byte(o.gzipContent)
		o.writeShaSum(data)
	case Blob:
		data, err := ioutil.ReadFile(o.Path)
		if err != nil {
			log.Fatal(err)
		}
		o.writeShaSum(data)
	default:
		log.Fatalf("RecCalcHashSum(): %v", got.ErrInvalidObjType)
	}
}

// writeShaSum takes bytes data, calculates sha sum for it and writes sum and hash string into the object struct.
func (o *Object) writeShaSum(data []byte) {
	h := sha1.New()
	h.Write(data)
	o.sha = h.Sum(nil)
	o.HashString = hashString(o.sha)
}

// buildContentLineForParent builds a string to put into parents (commit or tree) content to be archived.
func (o *Object) buildContentLineForParent() string {
	entries := []string{o.ObjType.toString(), o.HashString, o.Name}
	return strings.Join(entries, "\t")
}

// parentCommitShaContentLine reads commit archive and builds content line for commit
// pointing to parent commit.
func parentCommitShaContentLine(parentHash string) string {
	parentCommitPath := path.Join(got.CommitDirAbsPath(), parentHash)
	fd, err := os.Open(parentCommitPath)
	if err != nil {
		log.Fatal(err)
	}
	unarchiver, _ := gzip.NewReader(fd)
	defer fd.Close()
	defer unarchiver.Close()
	entries := []string{Commit.toString(), parentHash, unarchiver.Comment}
	return strings.Join(entries, "\t")
}

// RecWriteObjects recursively writes archive for objects in a graph.
func (o *Object) RecWriteObjects() {
	if o.ObjType == Commit || o.ObjType == Tree {
		for _, ch := range o.Children {
			ch.RecWriteObjects()
		}
	}

	o.write()
}

// write function writes archives for objects.
func (o *Object) write() {
	switch o.ObjType {
	case Commit:
		path := path.Join(got.CommitDirAbsPath(), o.HashString)
		writeArchive(path, o.Name, []byte(o.gzipContent), time.Now(), o.CommitMessage)
		got.UpdateHead(o.HashString)
	case Tree:
		path := path.Join(got.TreeDirAbsPath(), o.HashString)
		if exists(path) {
			break
		}
		writeArchive(path, o.Name, []byte(o.gzipContent), time.Now(), "")
	case Blob:
		path := path.Join(got.BlobDirAbsPath(), o.HashString)
		if exists(path) {
			break
		}
		data, err := ioutil.ReadFile(o.Path)
		if err != nil {
			log.Fatal(err)
		}
		writeArchive(path, o.Name, data, time.Now(), "")
	default:
		log.Fatalf("write(): %v", got.ErrInvalidObjType)
	}
}

// writeArchive implements archive writing for object data.
func writeArchive(p string, name string, data []byte, t time.Time, commitMessage string) {
	fd, _ := os.Create(p)
	archiver := gzip.NewWriter(fd)
	defer fd.Close()
	defer archiver.Close()
	archiver.Name = name
	archiver.ModTime = t
	if commitMessage != "" {
		archiver.Comment = commitMessage
	}
	archiver.Write(data)
}

// readArchive reads a gzip archive and returns its content and header struct.
func readArchive(p string) ([]byte, gzip.Header) {
	fd, err := os.Open(p)
	unarchiver, err := gzip.NewReader(fd)
	if err != nil {
		log.Fatalf("reading archive %s: %v", p, err)
	}
	defer fd.Close()
	defer unarchiver.Close()
	res, err := ioutil.ReadAll(unarchiver)
	if err != nil {
		log.Fatalf("reading archive %s: %v", p, err)
	}
	return res, unarchiver.Header
}

// hashString converts hashSum into string representation.
func hashString(hashSum []byte) string {
	return fmt.Sprintf("%x", hashSum)
}

// toString converts object type into its string representation.
func (t ObjectType) toString() string {
	switch t {
	case Commit:
		return "commit"
	case Tree:
		return "tree"
	case Blob:
		return "blob"
	default:
		log.Fatalf("toString(): %v", got.ErrInvalidObjType)
		panic("never reach")
	}
}

// strToObjType converts string into respective object type.
func strToObjType(s string) ObjectType {
	switch s {
	case "commit":
		return Commit
	case "tree":
		return Tree
	case "blob":
		return Blob
	default:
		log.Fatalf("strToObjType(): %v (%v)", got.ErrInvalidObjType, s)
		panic("never reach")
	}
}

// exists tests wheather a file exists.
func exists(path string) bool {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}
