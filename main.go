package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/shved/got/got"
	"github.com/shved/got/object"
	"github.com/shved/got/worktree"
)

func init() {
	flag.Parse()
}

func main() {
	command := flag.Arg(0)

	if command != "init" {
		got.SetRepoRoot()
	}

	switch command {
	case "init":
		got.InitRepo()
	case "commit":
		worktree.MakeCommit()
	case "to":
		shaString := flag.Arg(1)
		worktree.ToCommit(shaString)
	case "show":
		shaString := flag.Arg(1)
		fmt.Println(object.Show(shaString))
	case "log":
		fmt.Println(got.ReadLog())
	default:
		// TODO: print usage info
		fmt.Println("No commands provided")
		os.Exit(0)
	}
}
