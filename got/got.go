package got

import (
	"errors"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
)

var (
	ErrRepoAlreadyInited = errors.New("repo already initialized")
	ErrInvalidObjType    = errors.New("invalid object type")
	ErrNotGotRepo        = errors.New("not a got repo")
	ErrWrongRootType     = errors.New("only commit could be an object graph root")
)

var DefaultIgnoreEntries = []string{
	".gitignore",
	".git",
	".got",
	".DS_Store",
}

var AbsRepoRoot string

var EmptyCommitRef = []byte("0000000000000000000000000000000000000000")

var (
	gotPath     string = ".got"
	objectsPath string = strings.Join([]string{gotPath, "objects"}, string(filepath.Separator))
	commitPath  string = strings.Join([]string{objectsPath, "commit"}, string(filepath.Separator))
	treePath    string = strings.Join([]string{objectsPath, "tree"}, string(filepath.Separator))
	blobPath    string = strings.Join([]string{objectsPath, "blob"}, string(filepath.Separator))
	headPath    string = strings.Join([]string{gotPath, "HEAD"}, string(filepath.Separator))
)

func InitRepo() {
	if _, err := os.Stat(gotPath); os.IsNotExist(err) {
		os.Mkdir(gotPath, 0755)
	} else {
		log.Fatal(ErrRepoAlreadyInited)
	}

	os.Mkdir(objectsPath, 0755)
	os.Mkdir(commitPath, 0755)
	os.Mkdir(treePath, 0755)
	os.Mkdir(blobPath, 0755)

	if err := ioutil.WriteFile(headPath, EmptyCommitRef, 0644); err != nil {
		log.Fatal(err)
	}
}

func SetRepoRoot() {
	AbsRepoRoot = getRepoRoot()
}

func HeadAbsPath() string {
	return path.Join(AbsRepoRoot, headPath)
}

func UpdateHead(sha string) {
	if err := ioutil.WriteFile(headPath, []byte(sha), 0644); err != nil {
		log.Fatal(err)
	}
}

func CommitDirAbsPath() string {
	return path.Join(AbsRepoRoot, commitPath)
}

func TreeDirAbsPath() string {
	return path.Join(AbsRepoRoot, treePath)
}

func BlobDirAbsPath() string {
	return path.Join(AbsRepoRoot, blobPath)
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
		if file.Name() == gotPath {
			return true
		}
	}

	return false
}
