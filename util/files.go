package util

import (
	"log"
	"os"
	"os/exec"
	"regexp"
	"strconv"
)

// CanLoadFile returns true if a file can be opened or false if otherwise.
func CanLoadFile(path string) bool {
	if _, statErr := os.Stat(path); os.IsNotExist(statErr) {
		return false
	}
	return true
}

func FileSize(path string) (int64, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return fi.Size(), nil
}

func IsDirectory(path string) (bool, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return fi.IsDir(), nil
}

// Cwd returns the current working directory or panics.
func Cwd() string {
	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return pwd
}

var pdfPageCount = regexp.MustCompile(`Pages:\s+(\d+)`)

// pdfinfo ~/Desktop/ChefConf2014schedule.pdf
func GetPdfPageCount(file string) (int, error) {
	_, err := exec.LookPath("pdfinfo")
	if err != nil {
		log.Println("pdfinfo command not found")
		return 0, err
	}
	out, err := exec.Command("pdfinfo", file).Output()
	if err != nil {
		log.Println(string(out))
		log.Fatal(err)
		return 0, err
	}
	matches := pdfPageCount.FindStringSubmatch(string(out))
	if len(matches) == 2 {
		return strconv.Atoi(matches[1])
	}
	return 0, nil
}
