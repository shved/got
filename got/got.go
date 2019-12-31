// Package got holds all the global repo vars and functions.
package got

import (
	"errors"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

var (
	ErrRepoAlreadyInited = errors.New("repo already initialized")
	ErrInvalidObjType    = errors.New("invalid object type")
	ErrNotGotRepo        = errors.New("not a got repo")
	ErrWrongRootType     = errors.New("only commit could be an object graph root")
	ErrWrongLogEntryType = errors.New("only commit could be saved in repo logs")
	ErrObjDoesNotExist   = errors.New("object does not exist")
)

var DefaultIgnoreEntries = []string{
	".gitignore",
	".gitkeep",
	".git",
	".got",
	".DS_Store",
}

var AbsRepoRoot string

var EmptyCommitRef = []byte("0000000000000000000000000000000000000000")

var (
	gotPath     string = ".got"
	objectsPath string = path.Join(gotPath, "objects")
	headPath    string = path.Join(gotPath, "HEAD")
	logPath     string = path.Join(gotPath, "LOG")

	CommitPath string = path.Join(objectsPath, "commit")
	TreePath   string = path.Join(objectsPath, "tree")
	BlobPath   string = path.Join(objectsPath, "blob")

	logsHeader string = "Time\t\t\tCommit hash\t\t\t\t\tParent hash\t\t\t\t\tCommit message\n"
)

// InitRepo initializes a repo in a current working directory by creating a .got dir with all the needing content.
func InitRepo() {
	if _, err := os.Stat(gotPath); os.IsNotExist(err) {
		os.Mkdir(gotPath, 0755)
	} else {
		log.Fatal(ErrRepoAlreadyInited)
	}

	err := os.Mkdir(objectsPath, 0755)
	err = os.Mkdir(CommitPath, 0755)
	err = os.Mkdir(TreePath, 0755)
	err = os.Mkdir(BlobPath, 0755)

	err = ioutil.WriteFile(headPath, EmptyCommitRef, 0644)

	_, err = os.Create(logPath)

	if err != nil {
		log.Fatal(err)
	}
}

// SetRepoRoot finds closer .got dir in parent paths and sets its absolute path into an exported variable.
func SetRepoRoot() {
	AbsRepoRoot = getRepoRoot()
}

// CommitDirAbsPath returns absolute path holding commit objects.
func CommitDirAbsPath() string {
	return path.Join(AbsRepoRoot, CommitPath)
}

// TreeDirAbsPath returns absolute path holding tree objects.
func TreeDirAbsPath() string {
	return path.Join(AbsRepoRoot, TreePath)
}

// BlobDirAbsPath returns absolute path holding blob objects.
func BlobDirAbsPath() string {
	return path.Join(AbsRepoRoot, BlobPath)
}

// HeadsAbsPath returns absolute HEAD file path.
func HeadAbsPath() string {
	return path.Join(AbsRepoRoot, headPath)
}

// HeadsAbsPath returns absolute LOG file path.
func LogAbsPath() string {
	return path.Join(AbsRepoRoot, logPath)
}

// ReadLog reads all LOG file contents, reverses it for the right historical order, ads headers
// and returns the result as a string.
func ReadLog() string {
	contents, err := ioutil.ReadFile(LogAbsPath())
	if err != nil {
		log.Fatal(err)
	}
	withHeaders := string(contents) + logsHeader
	entries := strings.Split(withHeaders, "\n")
	sort.Sort(sort.Reverse(sort.StringSlice(entries)))
	logs := strings.Join(entries, "\n")
	return logs
}

// UpdateLog adds a log entry into a LOG file.
func UpdateLog(entry string) {
	f, err := os.OpenFile(LogAbsPath(), os.O_APPEND|os.O_WRONLY, 0644)
	defer f.Close()
	_, err = f.WriteString(entry)

	if err != nil {
		log.Fatal(err)
	}
}

// UpdateHead refresh last commit hash string in a HEAD file.
func UpdateHead(sha string) {
	if err := ioutil.WriteFile(HeadAbsPath(), []byte(sha), 0644); err != nil {
		log.Fatal(err)
	}
}

// ReadHead reads commit hash string from a HEAD file.
func ReadHead() string {
	commitSha, err := ioutil.ReadFile(HeadAbsPath())
	if err != nil {
		log.Fatal(err)
	}
	return string(commitSha)
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
	panic("never reach")
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
