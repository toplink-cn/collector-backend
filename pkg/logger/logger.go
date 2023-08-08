package logger

import (
	"log"
	"os"
)

func Println(v ...any) {
	env := os.Getenv("enviroment")
	if env != "production" {
		log.Println(v...)
	}
}

func Printf(format string, v ...any) {
	env := os.Getenv("enviroment")
	if env != "production" {
		log.Printf(format, v...)
	}
}
