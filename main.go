package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/shved/got/got"
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
		// worktree.ToCommit()
	default:
		// TODO: print usage info
		fmt.Println("No commands provided")
		os.Exit(0)
	}
}

// gzip header
// type Header struct {
// 	Comment string    // comment
// 	Extra   []byte    // "extra data"
// 	ModTime time.Time // modification time
// 	Name    string    // file name
// 	OS      byte      // operating system type
// }

// func readWriteGzip() {
//     path := "test/file.rb"
//     data, _ := ioutil.ReadFile(path)
//     fd, _ := os.Create("test/file.gz")
//     archiver := gzip.NewWriter(fd)
//     archiver.Comment = "hello"
//     archiver.Write(data)
//     archiver.Close()
//
//     fd2, _ := os.Open("test/file.gz")
//     unarchiver, _ := gzip.NewReader(fd2)
//     res := make([]byte, len(data))
//     fmt.Println(unarchiver.Comment)
//     size, _ := unarchiver.Read(res)
//     fmt.Println(size)
//     fmt.Println(res)
// }
