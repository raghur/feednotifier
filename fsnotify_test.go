package main

import (
	"fmt"
	"time"
	"github.com/fsnotify/fsnotify"
	"os"
	"testing"
	log "github.com/sirupsen/logrus"
)

func TestParseDuration(t *testing.T) {
	durationStr := "1m13.619259473s"
	d,_ := time.ParseDuration(durationStr)
	fmt.Printf("duration parsed as %v\n", d)
}
func TestFsNotifyDir(t *testing.T) {
	os.Mkdir("fsnotify", os.ModeDir)
	defer os.Remove("fsnotify")
	watcher, err := fsnotify.NewWatcher()
	defer watcher.Close()
	done := make(chan bool)

	go func() {
			for {
				select {
				case event := <-watcher.Events:
					log.Println("event:", event)
					if event.Op&fsnotify.Write == fsnotify.Write {
						log.Println("modified file:", event.Name)
					}
				case err := <-watcher.Errors:
					log.Println("error:", err)
				}
			}
	}()
	err = watcher.Add("fsnotify")
	if err != nil {
		log.Fatal(err)
	}
	<-done
}
