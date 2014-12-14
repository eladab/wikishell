# README #

Wikipedia for Terminal

## Installation ##

### OS X ###

* Install [golang] (https://golang.org/dl/)
* Create a directory that would be your GOPATH, for instance /Users/<username>/go 
* Create 3 directories under this directory: bin, pkg & src 
* In your .bashrc or .bash_profile - add the GOPATH variable:
  export GOPATH=/Users/<username>/go 
  Add the $GOPATH/bin to your $PATH, so that build products would be accessible globally:
  export PATH=$PATH:$GOPATH/bin
* source .bashrc (or .bash_profile) or restart the shell.
* Install git, mercurial and gcc (if you don’t have them already).
* go get github.com/eladab/wikishell

### Linux (Ubuntu) ###

* apt-get install golang
* Create a directory that would be your GOPATH, for instance /home/<username>/go
* Create 3 directories under this directory: bin, pkg & src
* In your .bashrc or .bash_profile - add the GOPATH variable:
  export GOPATH=/home/<username>/go 
  Add the $GOPATH/bin to your $PATH, so that build products would be accessible globally:
  export PATH=$PATH:$GOPATH/bin
* source .bashrc or restart the shell.
* Install git, mercurial and gcc (if you don’t have them already).

## Usage: ##

* wikishell (will show welcome page)
* wikishell [term]
