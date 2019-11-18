package main

import (
	"bytes"
	"fmt"
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
	oldLoc := cast.ToInt(os.Args[3])
	myDiff := diffmatchpatch.New()
	diff := myDiff.DiffMainRunes(oldFile, newFile, false)
	newLoc := myDiff.DiffXRuneIndex(diff, oldLoc)
	fmt.Println(string(oldFile[oldLoc:oldLoc+10]))
	fmt.Println(string(newFile[newLoc:newLoc+10]))

	fmt.Println(fmt.Sprintf("loc_change: %d -> %d", oldLoc, newLoc))
}
