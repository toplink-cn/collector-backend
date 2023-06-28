package util

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func FailOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

func LogIfErr(err error) {
	if err != nil {
		log.Printf("%s \n", err.Error())
	}
}

func GetRootDir() string {
	var dir string
	var err error
	if IsTesting() {
		dir, err = os.Getwd()
		if err != nil {
			fmt.Println(err)
		}
	} else {
		exePath, err := os.Executable()
		if err != nil {
			panic(err)
		}
		dir = filepath.Dir(exePath)
	}
	fmt.Println("root dir: ", dir)
	return dir
}

func IsTesting() bool {
	args := os.Args
	return len(args) > 0 && strings.Contains(strings.ToLower(os.Args[0]), "go-build")
}
