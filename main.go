package main

import (
	"bytes"
	"github.com/go-diff/diffmatchpatch"
	"github.com/spf13/cast"
	"io/ioutil"
	"os"
)

func ReadFileAsRunes(filename string) []rune{
	byteArray,_ := ioutil.ReadFile(filename)
	runes := bytes.Runes(byteArray)
	return runes
}

func main(){
	oldFile := ReadFileAsRunes(os.Args[1])
	newFile := ReadFileAsRunes(os.Args[2])
	myDiff := diffmatchpatch.New()
	diff := myDiff.DiffMainRunes(oldFile, newFile, false)
	myDiff.DiffXIndex(diff, cast.ToInt(os.Args[3]))
}
