package main

import (
	"crypto/sha1"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/shved/got/got"
	"github.com/shved/got/object"
	"github.com/shved/got/worktree"
)

var expectedHashSums map[string]string = map[string]string{
	"initial state":                  "e3980c53eecf817099d9eed5202e33d50a84a903",
	"repo initiated":                 "ff03475a2b21e13c2ad33881a5171f2aeb8f84a2",
	"after initial commit":           "8b8c226541835284c85241139074e9f5374017ca",
	"after first change":             "ca4d39f953416ab4a2afba301413da863331e11a",
	"after second change":            "fe48688c7d98149b75cea26937ea78caec56a023",
	"after checkout to first change": "100b270116cf66a49ce4cc312817bc4f70f90a9f",
}

var commitToCheckout = "4c69dc662e8873513ef9d1e1e1375579f6fd3725"

var expectedShowLen = 167
var expectedLogLen = 405

var dummyAppPath string

func TestMain(m *testing.M) {
	curDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("filed to get current working directory: %v", err)
	}
	dummyAppPath = path.Join(curDir, "test/dummy_app")
	os.Chdir(dummyAppPath)

	createDummyApp()
	exitCode := m.Run()
	removeDummyApp()
	os.Exit(exitCode)
}

func TestMainWorkflow(t *testing.T) {
    checkRepoSum(t, "initial state")

	got.InitRepo()
	got.SetRepoRoot()

	checkRepoSum(t, "repo initiated")

	worktree.MakeCommit("initial commit", time.Now())

	checkRepoSum(t, "after initial commit")

	makeFirstChange()
	worktree.MakeCommit("first change", time.Now().AddDate(0, 0, 1))

	checkRepoSum(t, "after first change")

	makeSecondChange()
	worktree.MakeCommit("second change", time.Now().AddDate(0, 0, 2))

	checkRepoSum(t, "after second change")

	worktree.ToCommit(commitToCheckout)

	checkRepoSum(t, "after checkout to first change")

	head := got.ReadHead()
	if head != commitToCheckout {
		t.Fatalf("expected head be on 4c69dc662e8873513ef9d1e1e1375579f6fd3725, got %v", head)
	}

	commitInfo := object.Show(commitToCheckout)
	if len(commitInfo) != expectedShowLen {
		t.Fatalf("expected to have %v bytes of commit contents, got %v", expectedShowLen, len(commitInfo))
	}

	logs := got.ReadLog()
	if len(logs) != expectedLogLen {
		t.Fatalf("expected to have %v bytes of logs, got %v", expectedLogLen, len(logs))
	}
}

func checkRepoSum(t *testing.T, step string) {
	sum := repoStateHashSum()
	if sum != expectedHashSums[step] {
		t.Fatalf("%v: expected to have %v sum, got %v", step, expectedHashSums[step], sum)
	}

}

func createDummyApp() {
	err := os.Mkdir("app", 0755)
	err = os.Mkdir("app/lib", 0755)
	err = os.Mkdir("app/views", 0755)

	err = ioutil.WriteFile("app/sample.file", []byte("some data here"), 0644)
	err = ioutil.WriteFile("app/lib/sample.php", []byte("some data here"), 0644)
	err = ioutil.WriteFile("app/views/sample.html", []byte("<body>some data here</body>"), 0644)

	if err != nil {
		log.Fatalf("failed to create dummy app: %v", err)
	}
}

func removeDummyApp() {
	err := os.RemoveAll("app")
	err = os.RemoveAll(".got")
	if err != nil {
		log.Fatalf("filed to remove dummy app: %v", err)
	}
}

func makeFirstChange() {
	err := ioutil.WriteFile("app/views/index.html", []byte("<body>hi there!</body>"), 0644)
	if err != nil {
		log.Fatalf("failed to make first change")
	}
}

func makeSecondChange() {
	err := ioutil.WriteFile("app/views/index.html", []byte("new content"), 0644)
	err = os.Remove("app/sample.file")
	if err != nil {
		log.Fatalf("failed to make second change")
	}
}

func repoStateHashSum() string {
	paths := readRepo()
	sort.Sort(sort.StringSlice(paths))
	str := strings.Join(paths, "")
	h := sha1.New()
	h.Write([]byte(str))
	sum := h.Sum(nil)
	return fmt.Sprintf("%x", sum)
}

func readRepo() []string {
	var paths []string

	repoWalker := func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			log.Fatal(err)
		}

		if path == dummyAppPath {
			return nil
		}

		relPath, err := filepath.Rel(dummyAppPath, path)
		if err != nil {
			return err
		}
		paths = append(paths, relPath)

		return nil
	}

	err := filepath.Walk(dummyAppPath, repoWalker)
	if err != nil {
		log.Fatal(err)
	}

	return paths
}
