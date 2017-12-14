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
