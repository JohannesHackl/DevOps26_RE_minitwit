# DevOps26_RE_minitwit

## Makefile:

### commands:
* build: Build flag_tool.c
* clean: deletes flag_tool

For the following commands considers using control.sh which is safer.
* init: Initializes the database in "./tmp/" Note: Only if minitwit.db does not exist.
* init-fresh-db: [USE WITH CAUTION!] Initializes the database in "./tmp/" overwriting the existing minitwit.db - 


## Go development docker container './docker/dev/Dockerfile'

The container run 'ubuntu24.04' and can install any version of go by using the flag --build-args GO_VERSION=<version>. However I would advise to use the development.sh script to build and run the container.

Right now there are only two different arguments to give the script:
- ./develop.sh build -> builds the Dockerfile
- ./develop.sh run -> runs the image after being built

Feel free to extend its functionality as you discover what you would like to have it doing :)

# Inside the container
You'll meet an ubuntu terminal with go installed and ready to use.
There are two folders: 
- '/go/src' => Contains the sourcefile and consists of file copied from ./src/ in out minitwit project
- '/go/bin' => Contains the binary which is compiled to simply typing 'gobuild' in the terminal

The docker container has a builtin script to compile the go module located at /go/src/ you simply need to run 'gobuild' and it will build the project to /go/bin/

When exiting the container the binary located at /go/bin/ will disappear. This probably needs to be changed and you're welcome to do that.

# Adding functionality to the container
You can add functionality (for example more scripts to do nice things for you) by just going to the Dockerfile located at ./docker/dev/ and extended to your liking. Feel free to expriment adding functionalities to the container :)
