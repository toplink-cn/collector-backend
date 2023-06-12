package util

import "log"

func FailOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

func LogIfErr(err error) {
	if err != nil {
		log.Printf("%s: %s", err.Error())
	}
}
