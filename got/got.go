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
	"time"
)

var (
	ErrRepoAlreadyInited = errors.New("repo already initialized")
	ErrInvalidObjType    = errors.New("invalid object type")
	ErrNotGotRepo        = errors.New("not a got repo")
	ErrWrongRootType     = errors.New("only commit could be an object graph root")
	ErrObjDoesNotExist   = errors.New("object does not exist")
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
	objectsPath string = path.Join(gotPath, "objects")
	headPath    string = path.Join(gotPath, "HEAD")
	logPath     string = path.Join(gotPath, "LOG")

	CommitPath string = path.Join(objectsPath, "commit")
	TreePath   string = path.Join(objectsPath, "tree")
	BlobPath   string = path.Join(objectsPath, "blob")

	logsHeader string = "Time\tCommit hash\tParent hash\tCommit message\n"
)

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

func SetRepoRoot() {
	AbsRepoRoot = getRepoRoot()
}

func HeadAbsPath() string {
	return path.Join(AbsRepoRoot, headPath)
}

func LogAbsPath() string {
	return path.Join(AbsRepoRoot, logPath)
}

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

func UpdateLog(t time.Time, hashString string, parentHashString string, message string) {
	logEntry := strings.Join(
		[]string{
			t.UTC().Format(time.RFC3339),
			hashString,
			parentHashString,
			message,
		},
		"\t",
	)
	logEntry = logEntry + "\n"

	f, err := os.OpenFile(LogAbsPath(), os.O_APPEND|os.O_WRONLY, 0644)
	defer f.Close()
	_, err = f.WriteString(logEntry)

	if err != nil {
		log.Fatal(err)
	}
}

func UpdateHead(sha string) {
	if err := ioutil.WriteFile(HeadAbsPath(), []byte(sha), 0644); err != nil {
		log.Fatal(err)
	}
}

func ReadHead() string {
	commitSha, err := ioutil.ReadFile(HeadAbsPath())
	if err != nil {
		log.Fatal(err)
	}
	return string(commitSha)
}

func CommitDirAbsPath() string {
	return path.Join(AbsRepoRoot, CommitPath)
}

func TreeDirAbsPath() string {
	return path.Join(AbsRepoRoot, TreePath)
}

func BlobDirAbsPath() string {
	return path.Join(AbsRepoRoot, BlobPath)
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
