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
got init
got commit 'initial commit'
got log
got to d143528ac209d5d927e485e0f923758a21d0901e
```

# TODO
- [x] git log
- [ ] fix timestamps in log
- [ ] support branches
- [ ] support .gotignore file among with default ingore entries
- [ ] ignore nested empty folders
- [ ] atomic commit writing
- [ ] reduce system calls (especially io)
- [ ] server and client over ssh
- [ ] documentation comments
- [ ] keep files permissions when checkout to commit
- [ ] command to delete hanging commits
