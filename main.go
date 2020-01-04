package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/shved/got/got"
	"github.com/shved/got/object"
	"github.com/shved/got/worktree"
)

var blankRepoCommands = []string{
	"",
	"init",
	"help",
}

func init() {
	flag.Parse()
}

func main() {
	command := flag.Arg(0)

	if !blankRepoCommand(command) {
		got.SetRepoRoot()
	}

	switch command {
	case "init":
		got.InitRepo()
		fmt.Println("Repo created in a current working directory")
	case "commit":
		message := flag.Arg(1)
		if message == "" {
			fmt.Println("No commit message provided")
			os.Exit(0)
		}
		worktree.MakeCommit(message, time.Now())
		fmt.Println("Worktree commited:", got.ReadHead())
	case "to":
		shaString := flag.Arg(1)
		if shaString == "" {
			fmt.Println("No commit hash provided")
			os.Exit(0)
		}
		worktree.ToCommit(shaString)
		fmt.Println("Worktree restored from commit:", shaString)
	case "show":
		shaString := flag.Arg(1)
		if shaString == "" {
			fmt.Println("No commit hash provided")
			os.Exit(0)
		}
		fmt.Println(object.Show(shaString))
	case "log":
		fmt.Println(got.ReadLog())
	case "current":
		fmt.Println("Current commit hash:", got.ReadHead())
	case "help":
		printHelpMessage()
		os.Exit(0)
	default:
		printHelpMessage()
		os.Exit(0)
	}
}

func printHelpMessage() {
	fmt.Println(`got init                                        // to init a repo in current dir
got commit 'initial commit'                     // to commit the state
got log                                         // to see commits list
got to d143528ac209d5d927e485e0f923758a21d0901e // to restore a commit
got current                                     // to see current head commit hash`)
}

func blankRepoCommand(command string) bool {
	for _, com := range blankRepoCommands {
		if com == command {
			return true
		}
	}

	return false
}
