package main

import (
	"bufio"
	"io"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
)

func ReadLines(fn string, trimset string, action func(l string) error) error {
	log.Debug("Loading file: ", fn)
	file, err := os.Open(fn)
	if err != nil {
		log.Errorf("error opening file %v\n", err)
		return err
	}
	defer file.Close()
	reader := bufio.NewReader(file)
	for {
		line, err := reader.ReadString('\n')
		if err == io.EOF {
			log.Infof("Completed reading file %s", fn)
			return nil
		}
		if err != nil {
			log.Error("Error reading line from file ", fn)
			continue
		}
		line = strings.Trim(line, trimset)

		err = action(line)
		if err != nil {
			return err
		}
	}
}

func copyFile(src, dst string) {
	from, err := os.Open(src)
	log.Infof("Copying from src:%s to dest: %s", src, dst)
	if err != nil {
		log.Errorf("Unable to open source file %s, %v", src, err)
	}
	defer from.Close()

	to, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		log.Errorf("Unable to open destination file %s, %v", dst, err)
	}
	defer to.Close()

	_, err = io.Copy(to, from)
	if err != nil {
		log.Errorf("Error while copying file %s -> %s, %v", src, dst, err)
	}
}