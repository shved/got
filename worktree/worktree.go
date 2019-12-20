package worktree

import (
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/shved/got/got"
	"github.com/shved/got/object"
)

type Worktree struct {
	root  *object.Object
	index []*object.Object
}

func NewFromWorktree() *Worktree {
	commit := &object.Object{ObjType: object.Commit}
	objIndex := buildObjIndex()
	return &Worktree{root: commit, index: objIndex}
}

// func NewFromCommit() *Worktree {
// 	//
// }

func MakeCommit() {
	wt := NewFromWorktree()
	wt.buildWorktreeGraph()
	wt.buildHashSums()
	wt.persistObjects()

	// for _, obj := range objIndex {
	// 	if obj.ObjType != Blob {
	// 		fmt.Println(hashString(obj.sha), obj.path+":")
	// 		fmt.Println(strings.Join(obj.contentLines, "\n"))
	// 		fmt.Println()
	// 	}
	// }
}

func buildObjIndex() []*object.Object {
	var objIndex []*object.Object

	worktreeWalker := func(path string, f os.FileInfo, err error) error {
		if err != nil {
			log.Fatal(err)
		}

		if path == got.AbsRepoRoot {
			return nil
		}

		empty, _ := isEmpty(path)

		if f.IsDir() && empty {
			return filepath.SkipDir
		}

		for _, entry := range got.DefaultIgnoreEntries {
			if f.IsDir() && f.Name() == entry {
				return filepath.SkipDir
			}

			if !f.IsDir() && f.Name() == entry {
				return nil
			}
		}

		// build object
		var obj object.Object

		relPath, err := filepath.Rel(got.AbsRepoRoot, path)
		parentPath := filepath.Dir(path)
		relParentPath, err := filepath.Rel(got.AbsRepoRoot, parentPath)
		if err != nil {
			log.Fatal(err)
		}

		if f.IsDir() {
			obj = object.Object{ObjType: object.Tree, ParentPath: relParentPath, Name: f.Name(), Path: relPath}
		} else {
			obj = object.Object{ObjType: object.Blob, ParentPath: relParentPath, Name: f.Name(), Path: relPath}
		}

		objIndex = append(objIndex, &obj)

		return nil
	}

	err := filepath.Walk(got.AbsRepoRoot, worktreeWalker)
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

func (wt *Worktree) persistObjects() {
	wt.root.RecWriteObjects()
}

func (wt *Worktree) buildHashSums() {
	wt.root.RecBuildHashSums()
}

func (wt *Worktree) buildWorktreeGraph() {
	if wt.root.ObjType != object.Commit {
		log.Fatal(got.ErrWrongRootType)
	}

	parentCommitSha, err := ioutil.ReadFile(got.HeadAbsPath())
	if err != nil {
		log.Fatal(err)
	}
	wt.root.ParentCommitSha = parentCommitSha

	for _, obj := range wt.index {
		if obj.ParentPath == "." {
			obj.Parent = wt.root
			wt.root.Children = append(wt.root.Children, obj)
		}

		for _, oObj := range wt.index {
			if oObj.ParentPath == obj.Path {
				obj.Children = append(obj.Children, oObj)
			}

			if obj.ParentPath == oObj.Path {
				obj.Parent = oObj
			}
		}
	}
}
