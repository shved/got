package main

import (
	"compress/gzip"
	"crypto/sha1"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type objectType int

const (
	Commit objectType = iota + 1
	Tree
	Blob
)

var (
	_gotPath     string = ".got"
	_objectsPath string = strings.Join([]string{_gotPath, "objects"}, string(filepath.Separator))
	_commitPath  string = strings.Join([]string{_objectsPath, "commit"}, string(filepath.Separator))
	_treePath    string = strings.Join([]string{_objectsPath, "tree"}, string(filepath.Separator))
	_blobPath    string = strings.Join([]string{_objectsPath, "blob"}, string(filepath.Separator))
	_headPath    string = strings.Join([]string{_gotPath, "HEAD"}, string(filepath.Separator))
)

var (
	ErrInvalidObjType = errors.New("invalid object type")
	ErrNotGotRepo     = errors.New("not a got repo")
	ErrWrongRootType  = errors.New("only commit could be an object graph root")
)

var _emptyCommitRef = []byte("0000000000000000000000000000000000000000")

var _defaultIgnoreEntries = []string{
	".gitignore",
	".git",
	".got",
	".DS_Store",
}

var _absRepoRoot string

type object struct {
	objType          objectType
	parent           *object
	children         []*object
	name             string
	parentPath       string
	path             string
	sha              []byte
	contentLines     []string
	gzipContent      string
	parentCommitHash string
}

////////////////////////////////////////
// package got
// import "github.com/shved/got/worktree"
// package worktree
// got.makeCommit()
// got.toCommit()
// got.initRepo()
// wt := &worktree.New()
// wt.calcHashes()

// https: //fabianlindfors.se/blog/decorators-in-go-using-embedded-structs/
// type worktree struct {
//     root *commit
//     objIndex []*object
// }
//
// type commit struct {
//     objType objectType
//	   parentCommitHash string
//     object
// }
//
// type tree struct {
//     objType objectType
//     object
// }
//
// type blob struct {
//     objType objectType
//     object
// }
//
// type objectReaderWriter interface {
//     WriteObject()
//     ReadObject()
// }
//
// type objectBuilder interface {
//     BuildObject()
// }
//
// func (w *worktree) calcObjSha() {
// }
//
// func (c *commit) WriteObject() {
// }
//
// func (c *commit) WriteObject() {
// }
////////////////////////////////////////

func init() {
	flag.Parse()
}

func main() {
	command := flag.Arg(0)

	if command != "init" {
		_absRepoRoot = getRepoRoot()
	}

	switch command {
	case "init":
		initRepo()
	case "commit":
		makeCommit()
	case "to":
		toCommit()
	default:
		// TODO: print usage info
		fmt.Println("No commands provided")
		os.Exit(0)
	}
}

func initRepo() {
	if _, err := os.Stat(_gotPath); os.IsNotExist(err) {
		os.Mkdir(_gotPath, 0755)
	} else {
		log.Fatal("repo already initialized")
	}

	// TODO save paths to consts
	os.Mkdir(_objectsPath, 0755)
	os.Mkdir(_commitPath, 0755)
	os.Mkdir(_treePath, 0755)
	os.Mkdir(_blobPath, 0755)

	if err := ioutil.WriteFile(_headPath, _emptyCommitRef, 0644); err != nil {
		log.Fatal(err)
	}
}

func makeCommit() {
	objIndex := buildObjIndex()
	commit := &object{objType: Commit}
	commit.buildWorktreeGraph(objIndex)
	commit.recBuildHashSums()
	commit.recWriteObjects()

	// for _, obj := range objIndex {
	// 	if obj.objType != Blob {
	// 		fmt.Println(hashString(obj.sha), obj.path+":")
	// 		fmt.Println(strings.Join(obj.contentLines, "\n"))
	// 		fmt.Println()
	// 	}
	// }

	// if comment := flag.Arg(1); comment != "" {
	// 	if len(additions) > 0 || len(deletions) > 0 || len(modifications) > 0 {
	// 		writeCommit(comment)
	// 		fmt.Println(comment)
	// 	} else {
	// 		fmt.Println("No changes to commit")
	// 		os.Exit(0)
	// 	}
	//
	// 	writeCommit(comment)
	// 	fmt.Println(comment)
	// }
}

func toCommit() {

}

// recursive calculate all objects sha1 started from very far children
func (o *object) recBuildHashSums() {
	switch o.objType {
	case Commit:
		for _, ch := range o.children {
			ch.recBuildHashSums()
			o.contentLines = append(o.contentLines, ch.buildContentLineForParent())
		}
		parentCommitLine := parentCommitShaContentLine(o.parentCommitHash)
		o.contentLines = append(o.contentLines, parentCommitLine)
		sort.Strings(o.contentLines)
		o.gzipContent = strings.Join(o.contentLines, "\n")
		data := []byte(o.gzipContent)
		h := sha1.New()
		h.Write(data)
		o.sha = h.Sum(nil)
	case Tree:
		for _, ch := range o.children {
			ch.recBuildHashSums()
			o.contentLines = append(o.contentLines, ch.buildContentLineForParent())
		}
		sort.Strings(o.contentLines)
		o.gzipContent = strings.Join(o.contentLines, "\n")
		data := []byte(o.gzipContent)
		h := sha1.New()
		h.Write([]byte(o.path))
		h.Write(data)
		o.sha = h.Sum(nil)
	case Blob:
		data, err := ioutil.ReadFile(o.path)
		if err != nil {
			log.Fatal(err)
		}

		h := sha1.New()
		h.Write([]byte(o.path))
		h.Write(data)
		o.sha = h.Sum(nil)
	default:
		log.Fatal(ErrInvalidObjType)
	}
}

func (o *object) recWriteObjects() {
	if o.objType == Commit || o.objType == Tree {
		for _, ch := range o.children {
			ch.recWriteObjects()
		}
	}

	o.write()
}

func updateHead(sha []byte) {
	if err := ioutil.WriteFile(_headPath, sha, 0644); err != nil {
		log.Fatal(err)
	}
}

func (o *object) write() {
	switch o.objType {
	case Commit:
		path := path.Join(commitDirAbsPath(), hashString(o.sha))
		fd, _ := os.Create(path)
		archiver := gzip.NewWriter(fd)
		archiver.Name = o.name
		archiver.ModTime = time.Now()
		archiver.Write([]byte(o.gzipContent))
		archiver.Close()
		updateHead(o.sha)
	case Tree:
		path := path.Join(treeDirAbsPath(), hashString(o.sha))
		if exists(path) {
			break
		}
		fd, _ := os.Create(path)
		archiver := gzip.NewWriter(fd)
		archiver.Name = o.name
		archiver.ModTime = time.Now()
		archiver.Write([]byte(o.gzipContent))
		archiver.Close()
	case Blob:
		path := path.Join(blobDirAbsPath(), hashString(o.sha))
		if exists(path) {
			break
		}
		data, err := ioutil.ReadFile(o.path)
		if err != nil {
			log.Fatal(err)
		}
		fd, _ := os.Create(path)
		archiver := gzip.NewWriter(fd)
		archiver.Name = o.name
		archiver.ModTime = time.Now()
		archiver.Write(data)
		archiver.Close()
	default:
		log.Fatal(ErrInvalidObjType)
	}
}

func commitDirAbsPath() string {
	return path.Join(_absRepoRoot, _commitPath)
}

func treeDirAbsPath() string {
	return path.Join(_absRepoRoot, _treePath)
}

func blobDirAbsPath() string {
	return path.Join(_absRepoRoot, _blobPath)
}

func headAbsPath() string {
	return path.Join(_absRepoRoot, _headPath)
}

func (o *object) buildContentLineForParent() string {
	entries := []string{o.objType.toString(), hashString(o.sha), o.name}
	return strings.Join(entries, "\t")
}

func parentCommitShaContentLine(hashString string) string {
	// TODO add real commit message when it will be unpacked as a worktree object
	entries := []string{Commit.toString(), hashString, "here will be commit message"}
	return strings.Join(entries, "\t")
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
		log.Fatal(ErrInvalidObjType)
	}
	return ""
}

func (commit *object) buildWorktreeGraph(objIndex []*object) {
	if commit.objType != Commit {
		log.Fatal(ErrWrongRootType)
	}

	parentCommitHash, err := ioutil.ReadFile(headAbsPath())
	if err != nil {
		log.Fatal(err)
	}
	commit.parentCommitHash = string(parentCommitHash)

	for _, obj := range objIndex {
		if obj.parentPath == "." {
			obj.parent = commit
			commit.children = append(commit.children, obj)
		}

		for _, oObj := range objIndex {
			if oObj.parentPath == obj.path {
				obj.children = append(obj.children, oObj)
			}

			if obj.parentPath == oObj.path {
				obj.parent = oObj
			}
		}
	}
}

func printEntry(file string) {
	fmt.Println(file)
}

func buildObjIndex() []*object {
	var objIndex []*object

	worktreeWalker := func(path string, f os.FileInfo, err error) error {
		if err != nil {
			log.Fatal(err)
		}

		if path == _absRepoRoot {
			return nil
		}

		empty, _ := isEmpty(path)

		if f.IsDir() && empty {
			return filepath.SkipDir
		}

		for _, entry := range _defaultIgnoreEntries {
			if f.IsDir() && f.Name() == entry {
				return filepath.SkipDir
			}

			if !f.IsDir() && f.Name() == entry {
				return nil
			}
		}

		// build object
		var obj object

		relPath, err := filepath.Rel(_absRepoRoot, path)
		parentPath := filepath.Dir(path)
		relParentPath, err := filepath.Rel(_absRepoRoot, parentPath)
		if err != nil {
			log.Fatal(err)
		}

		if f.IsDir() {
			obj = object{objType: Tree, parentPath: relParentPath, name: f.Name(), path: relPath}
		} else {
			obj = object{objType: Blob, parentPath: relParentPath, name: f.Name(), path: relPath}
		}

		objIndex = append(objIndex, &obj)

		return nil
	}

	err := filepath.Walk(_absRepoRoot, worktreeWalker)
	if err != nil {
		log.Fatal(err)
	}

	return objIndex
}

func isEmpty(path string) (bool, error) {
	fd, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer fd.Close()

	_, err = fd.Readdirnames(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err
}

func getRepoRoot() string {
	relPath := getRootRelPath()
	absPath, err := filepath.Abs(relPath)
	if err != nil {
		log.Fatal(err)
	}
	return absPath
}

func getRootRelPath() string {
	path := "."
	if isRepoRoot(path) {
		return path
	}

	path = ".."

	for {
		if abs, _ := filepath.Abs(path); abs == string(filepath.Separator) {
			log.Fatal(ErrNotGotRepo)
		}
		if isRepoRoot(path) {
			return path
		}
		path = path + string(filepath.Separator) + ".."
	}

	log.Fatal(ErrNotGotRepo)
	// TODO rewrite to return string, error
	return ""
}

func isRepoRoot(path string) bool {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		log.Fatal(err)
	}

	for _, file := range files {
		if file.Name() == ".got" {
			return true
		}
	}

	return false
}

// func readWriteGzip() {
//     path := "test/file.rb"
//     data, _ := ioutil.ReadFile(path)
//     fd, _ := os.Create("test/file.gz")
//     archiver := gzip.NewWriter(fd)
//     archiver.Comment = "hello"
//     archiver.Write(data)
//     archiver.Close()
//
//     fd2, _ := os.Open("test/file.gz")
//     unarchiver, _ := gzip.NewReader(fd2)
//     res := make([]byte, len(data))
//     fmt.Println(unarchiver.Comment)
//     size, _ := unarchiver.Read(res)
//     fmt.Println(size)
//     fmt.Println(res)
// }

// gzip header
// type Header struct {
// 	Comment string    // comment
// 	Extra   []byte    // "extra data"
// 	ModTime time.Time // modification time
// 	Name    string    // file name
// 	OS      byte      // operating system type
// }

// func writeCommit(message string) {
// retryLoop:
// 	path := path.Join(commitsObjDirAbsPath(), hashString(commitSha(message)))
// 	if exists(path) {
// 		goto retryLoop
// 	}
//
// 	fmt.Println()
// 	fmt.Println("commit obj path")
// 	fmt.Println(path)
// }

func truncatePathPrefix(fPath string) string {
	return strings.Replace(fPath, _absRepoRoot+string(os.PathSeparator), "", 1)
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

// func commitSha(message string) []byte {
// 	h := sha1.New()
// 	h.Write([]byte(time.Now().String()))
// 	h.Write([]byte(message))
// 	return h.Sum(nil)
// }

// func treeSha(dirPath string) []byte {
// 	fileInfos, err := ioutil.ReadDir(dirPath)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
//
// 	var filesFromRepoRoot []string
//
// 	for _, file := range fileInfos {
// 		// filesFromRepoRoot = append(filesFromRepoRoot, truncatePathPrefix(dirPath)
// 	}
//
// 	h := sha1.New()
// 	// h.Write([]byte(filePath))
// 	// h.Write(filePath)
//     return h.Sum(nil)
// }
//
// func fileSha(filePath string) []byte {
//     data, err := ioutil.ReadFile(filePath)
//     if err != nil {
//         log.Fatal(err)
//     }
//
//     h := sha1.New()
//     h.Write(data)
//     return h.Sum(nil)
// }
// func readWriteGzip() {
//     path := "test/file.rb"
//     data, _ := ioutil.ReadFile(path)
//     fd, _ := os.Create("test/file.gz")
//     archiver := gzip.NewWriter(fd)
//     archiver.Comment = "hello"
//     archiver.Write(data)
//     archiver.Close()
//
//     fd2, _ := os.Open("test/file.gz")
//     unarchiver, _ := gzip.NewReader(fd2)
//     res := make([]byte, len(data))
//     fmt.Println(unarchiver.Comment)
//     size, _ := unarchiver.Read(res)
//     fmt.Println(size)
//     fmt.Println(res)
// }

func debPrintGraphChildren(ind []*object) {
	for _, obj := range ind {
		fmt.Println(obj.name, obj.parent)
		for _, ch := range obj.children {
			fmt.Printf("%s", &ch.name)
		}
	}
	fmt.Println()
}

func debPrintGraphSums(ind []*object) {
	for _, obj := range ind {
		fmt.Println(obj.path, hashString(obj.sha))
	}
}
