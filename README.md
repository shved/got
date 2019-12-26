# Q&A

Q: What is Got?<br/>
A: It is a fully not distributed, purely local CVS

Q: Do I need Got?<br/>
A: Absolutely not

Q: Why should I use Got when I have Git?<br/>
A: You shouldn't

Q: What was the reason to introduce Got?<br/>
A: Absolutely no reason, just pure fun

Q: Why Got is missing feature X, Y, Z?<br/>
A: Because it is oversimplified Git

# Installation

* clone the repo
* build it with `make build`
* put the output file to your executables path

# Usage

```
got init                                        // to init a repo in current dir
got commit 'initial commit'                     // to commit the state
got log                                         // to see commits list
got to d143528ac209d5d927e485e0f923758a21d0901e // to restore a commit
got current                                     // to see current head commit hash
```

# TODO
- [x] git log
- [x] add commands success messages
- [x] documentation comments
- [ ] support branches
- [ ] support .gotignore file among with default ingore entries
- [ ] ignore nested empty folders
- [ ] reduce system calls (especially io)
- [ ] server and client over ssh
- [ ] keep files permissions when checkout to commit
- [ ] command to delete hanging commits
- [ ] experiment with object compression level
- [ ] atomic commit writing
