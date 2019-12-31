package main

import (
	"crypto/sha1"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	p "path"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"testing/quick"
	"time"

	"github.com/shved/got/got"
	"github.com/shved/got/worktree"
)

const pathSymbols = "abcdefghijklmnopqrstuvwxyz-_0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"

var dummyAppPath string

type Project struct {
	Files []File
	Paths []string
}

type File struct {
	Path    string
	Content []byte
}

func (prj *Project) Generate(r *rand.Rand, _ int) reflect.Value {
	layers := r.Intn(5)

	var files []File
	var fullPaths []string
	var paths []string

	addLayer := func(layer int, prefixes []string) []string {
		c := len(prefixes) + r.Intn(5)
		var res []string
		for i := 0; i <= c; i++ {
			randomStr := randString(r, r.Intn(10))
			randomPrefix := randStringFromRange(r, prefixes)
			randomPath := p.Join(randomPrefix, randomStr)
			res = append(res, randomPath)
		}
		return res
	}

	for i := 0; i <= layers; i++ {
		paths = addLayer(i, paths)
	}

	for _, path := range paths {
		filesCount := r.Intn(5)
		for i := 0; i <= filesCount; i++ {
			randomContent, err := generateBytes(r)
			if err != nil {
				log.Fatalf("generating file: %v", err)
			}
			randomName := randString(r, r.Intn(10))

			fullPath := p.Join(path, randomName)
			fullPaths = append(fullPaths, fullPath)
			files = append(files, File{Path: fullPath, Content: randomContent})
		}
	}

	return reflect.ValueOf(&Project{Files: files, Paths: fullPaths})
}

func generateBytes(r *rand.Rand) ([]byte, error) {
	b := make([]byte, r.Intn(50))
	_, err := r.Read(b)
	return b, err
}

func randString(r *rand.Rand, n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = pathSymbols[r.Intn(len(pathSymbols))]
	}
	return string(b)
}

func randStringFromRange(r *rand.Rand, rng []string) string {
	if len(rng) > 0 {
		return rng[r.Intn(len(rng))]
	}

	return ""
}

func TestCommit(t *testing.T) {
	curDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("filed to get current working directory: %v", err)
	}
	dummyAppPath = p.Join(curDir, "test/dummy_app")
	if curDir != dummyAppPath {
		os.Chdir(dummyAppPath)
	}

	assertion := func(prj *Project) bool {
		prj.Persist()
		got.InitRepo()
		got.SetRepoRoot()
		worktree.MakeCommit("message", time.Now())
		str1, firstSum := repoStateHashSum()
		commitHash := got.ReadHead()
		removeDummyApp(false)
		worktree.ToCommit(commitHash)
		str2, secondSum := repoStateHashSum()
		fmt.Println(str1, str2)
		removeDummyApp(true)

		return firstSum == secondSum
	}

	if err := quick.Check(assertion, nil); err != nil {
		t.Fatal(err)
	}
}

func (prj *Project) Persist() {
	for _, f := range prj.Files {
		pathWithApp := p.Join("app", f.Path)
		basePath := p.Dir(pathWithApp)
		os.MkdirAll(basePath, 0755)
		_ = ioutil.WriteFile(pathWithApp, f.Content, 0644)
	}
}

func repoStateHashSum() (string, string) {
	paths := readRepo()
	sort.Sort(sort.StringSlice(paths))
	str := strings.Join(paths, "")
	h := sha1.New()
	h.Write([]byte(str))
	sum := h.Sum(nil)
	return str, fmt.Sprintf("%x", sum)
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

		if relPath == ".gitkeep" {
			return nil
		}

		if relPath == ".got" && fi.IsDir() {
			return filepath.SkipDir
		}

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

func removeDummyApp(withRepo bool) {
	err := os.RemoveAll("app")
	if withRepo {
		err = os.RemoveAll(".got")
	}
	if err != nil {
		log.Fatalf("filed to remove dummy app: %v", err)
	}
}
