package got

import (
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/shved/got/misc"
)

var dummyAppPath string

func TestMain(m *testing.M) {
	rootP, err := filepath.Abs("..")
	if err != nil {
		log.Fatalf("get dummy app abs path: %v", err)
	}
	dummyAppPath = path.Join(rootP, "test/dummy_app")
	os.Chdir(dummyAppPath)

	misc.CreateDummyApp()
	exitCode := m.Run()
	misc.RemoveDummyApp()
	os.Exit(exitCode)
}

func TestInitRepo(t *testing.T) {
	InitRepo()
	entries, err := ioutil.ReadDir(dummyAppPath)
	if err != nil {
		t.Fatalf("reading dummy app dir: %v", err)
	}
	var names []string
	for _, fi := range entries {
		names = append(names, fi.Name())
	}

	var repoInitiated = false
	for _, name := range names {
		if name == ".got" {
			repoInitiated = true
		}
	}

	if !repoInitiated {
		t.Fatalf("no repo dir found in dummy app path, got %v", names)
	}

	SetRepoRoot()
	if AbsRepoRoot != dummyAppPath {
		t.Fatalf("expected repo root to be %v, got %v", dummyAppPath, AbsRepoRoot)
	}
}
