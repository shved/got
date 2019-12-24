package worktree

import (
	"io"
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

func NewFromWorktree(commitMessage string) *Worktree {
	commit := &object.Object{ObjType: object.Commit, CommitMessage: commitMessage}
	objIndex := buildObjIndex()
	return &Worktree{root: commit, index: objIndex}
}

func NewFromCommit(commitHash string) *Worktree {
	commit := object.RecReadObject(object.Commit, commitHash, &object.Object{})
	return &Worktree{root: commit}
}

func MakeCommit(message string) {
	wt := NewFromWorktree(message)
	wt.buildWorktreeGraph()
	wt.buildHashSums()
	wt.persistObjects()
	got.UpdateLog(wt.root.Timestamp, wt.root.HashString, wt.root.ParentCommitHash, wt.root.CommitMessage)
}

func ToCommit(commitHash string) {
	wt := NewFromCommit(commitHash)
	// TODO insert prompt before rewrite worktree
	wt.restoreFromObjects()
}

func (wt *Worktree) restoreFromObjects() {
	eraseCurrentWorktree()
	wt.root.RecRestoreFromObject(got.AbsRepoRoot)
}

func eraseCurrentWorktree() {
	var paths []string

	worktreeWalker := func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			log.Fatal(err)
		}

		if path == got.AbsRepoRoot {
			return nil
		}

		for _, entry := range got.DefaultIgnoreEntries {
			if fi.IsDir() && fi.Name() == entry {
				return filepath.SkipDir
			}

			if !fi.IsDir() && fi.Name() == entry {
				return nil
			}
		}

		paths = append(paths, path)

		return nil
	}

	err := filepath.Walk(got.AbsRepoRoot, worktreeWalker)
	if err != nil {
		log.Fatal(err)
	}

	for _, p := range paths {
		_ = os.RemoveAll(p)
	}
}

func buildObjIndex() []*object.Object {
	var objIndex []*object.Object

	worktreeWalker := func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			log.Fatal(err)
		}

		if path == got.AbsRepoRoot {
			return nil
		}

		empty, _ := isEmpty(path)

		if fi.IsDir() && empty {
			return filepath.SkipDir
		}

		for _, entry := range got.DefaultIgnoreEntries {
			if fi.IsDir() && fi.Name() == entry {
				return filepath.SkipDir
			}

			if !fi.IsDir() && fi.Name() == entry {
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

		if fi.IsDir() {
			obj = object.Object{ObjType: object.Tree, ParentPath: relParentPath, Name: fi.Name(), Path: relPath}
		} else {
			obj = object.Object{ObjType: object.Blob, ParentPath: relParentPath, Name: fi.Name(), Path: relPath}
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
	wt.root.RecCalcHashSum()
}

func (wt *Worktree) buildWorktreeGraph() {
	if wt.root.ObjType != object.Commit {
		log.Fatal(got.ErrWrongRootType)
	}

	wt.root.ParentCommitHash = got.ReadHead()

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
