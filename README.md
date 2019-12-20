# Q&A

Q: What is Got?
A: It is a fully not distributed, purely local CVS

Q: Do I need Got?
A: Absolutely not

Q: Why should I use Got when I have Git?
A: You shouldn't

Q: What was the reason to introduce Got?
A: Absolutely no reason, just pure fun

# Installation

* clone the repo
* build it with `make build`
* put the output file to your executables path

# Usage

```
got init
got commit // to see what will be commited
got commit 'initial commit'
got log
got to d143528ac209d5d927e485e0f923758a21d0901e
```

# TODO
* support branches
* support .gotignore file among with default ingore entries
* ignore nested empty folders
* atomic commit writing
* reduce system calls and hard drive usage
* server and client
* documentation comments
