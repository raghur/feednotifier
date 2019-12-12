package feednotifier

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"
	"os"
	"testing"
	"time"
)

func TestParseDuration(t *testing.T) {
	durationStr := "1m13.619259473s"
	d, _ := time.ParseDuration(durationStr)
	fmt.Printf("duration parsed as %v\n", d)
}
func TestFsNotifyDir(t *testing.T) {
	t.Skipf("Skipping fsnotify test")
	os.Mkdir("fsnotify", os.ModeDir)
	defer os.Remove("fsnotify")
	watcher, err := fsnotify.NewWatcher()
	defer watcher.Close()
	done := make(chan bool)

	go func() {
		for {
			select {
			case event := <-watcher.Events:
				t.Log("event:", event)
				if event.Op&fsnotify.Write == fsnotify.Write {
					t.Log("modified file:", event.Name)
				}
			case err := <-watcher.Errors:
				t.Log("error:", err)
			}
		}
	}()
	err = watcher.Add("fsnotify")
	if err != nil {
		log.Fatal(err)
	}
	<-done
}
