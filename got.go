package main

import (
	_ "crypto/sha1"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	_ "time"
)

type objectType int

const (
	Commit objectType = iota + 1
	Tree
	Blob
)

var (
	ErrInvalidObjType = errors.New("invalid object type")
	ErrNotGotRepo     = errors.New("not a got repo")
	ErrWrongRootType  = errors.New("only commit could be an object graph root")
)

var _emptyCommitRef = []byte("0000000000000000000000000000000000000000")

var _defaultIgnoreEntries = []string{
	".git",
	".got",
	".DS_Store",
}

var _absRepoRoot string

type object struct {
	objType      objectType
	parent       *object
	children     []*object
	name         string
	parentPath   string
	path         string
	sha          []byte
	contentLines []string
	gzipContent  []byte
}

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
		fmt.Println("hi")
	default:
		// TODO: print usage info
		fmt.Println("No commands provided")
		os.Exit(0)
	}
}

func initRepo() {
	if _, err := os.Stat(".got"); os.IsNotExist(err) {
		os.Mkdir(".got", 0755)
	} else {
		log.Fatal("repo already initialized")
	}

	// TODO save paths to consts
	os.Mkdir(".got/objects", 0755)
	os.Mkdir(".got/objects/commits", 0755)
	os.Mkdir(".got/objects/tree", 0755)
	os.Mkdir(".got/objects/files", 0755)

	if err := ioutil.WriteFile(".got/HEAD", _emptyCommitRef, 0644); err != nil {
		log.Fatal(err)
	}
}

func makeCommit() {
	objIndex := buildObjIndex()
	rootObj := &object{objType: Commit}
	buildWorktreeGraph(objIndex, rootObj)

	for _, obj := range objIndex {
		fmt.Println(obj.name, obj.parent)
		for _, ch := range obj.children {
			fmt.Printf("%s", &ch.name)
		}
		fmt.Println()
	}

	if comment := flag.Arg(1); comment != "" {
		// if len(additions) > 0 || len(deletions) > 0 || len(modifications) > 0 {
		// 	writeCommit(comment)
		// 	fmt.Println(comment)
		// } else {
		// 	fmt.Println("No changes to commit")
		// 	os.Exit(0)
		// }

		// writeCommit(comment)
		fmt.Println(comment)
	}
}

// recursive calculate all objects sha1 started from very far children
// func buildHashSums(o *obj) {
// 	switch o.objType {
// 	case Commit:
// 		//
// 	case Tree:
// 		for _, obj := range o.children {
// 			buildHashSums(obj)
// 		}
// 	case Blob:
// 		data, err := ioutil.ReadFile(o.path)
// 		if err != nil {
// 			log.Fatal(err)
// 		}
//
// 		h := sha1.New()
// 		h.Write(data)
// 		o.sha = h.Sum(nil)
// 	default:
// 		log.Fatal(ErrInvalidObjType)
// 	}
// }

func buildWorktreeGraph(objIndex []*object, commit *object) {
	if commit.objType != Commit {
		log.Fatal(ErrWrongRootType)
	}

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

func commitsObjDirAbsPath() string {
	return path.Join(_absRepoRoot, ".got/objects/commits")
}

func treeObjDirAbsPath() string {
	return path.Join(_absRepoRoot, ".got/objects/tree")
}

func filesObjDirAbsPath() string {
	return path.Join(_absRepoRoot, ".got/objects/files")
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

func truncatePathPrefix(fPath string) string {
	return strings.Replace(fPath, _absRepoRoot+string(os.PathSeparator), "", 1)
}

func hashString(hashSum []byte) string {
	return fmt.Sprintf("%x\n", hashSum)
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
