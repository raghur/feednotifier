package main

import (
	"bufio"
	"crypto/md5"
	"fmt"
	"github.com/jasonlvhit/gocron"
	"github.com/mmcdole/gofeed"
	log "github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type ratelimitError struct {
	retryDuration time.Duration
}

func (e *ratelimitError) Error() string {
	return fmt.Sprintf("Rate limited - retry after %v", e.retryDuration)
}

type MonitoredFile struct {
	filename string
	urls     []string
	interval uint64
}

func New(filename string, interval uint64) *MonitoredFile {
	var mf MonitoredFile
	mf.filename = filename
	mf.interval = interval
	return &mf
}

func (mf *MonitoredFile) Start() {
	processFile(mf.filename)
	gocron.Every(mf.interval).Minutes().Do(processFile, mf.filename)
}

func (mf *MonitoredFile) Stop() {

}

func (mf *MonitoredFile) Delete() {

}

func downloadFile(line string) (base string, tempfn string, isTemp bool, err error) {
	url, err := url.Parse(line)
	if err != nil {
		log.Errorf("Unable to parse url %v\n", err)
		return
	}
	r, err := http.Get(url.String())
	if err != nil {
		log.Errorf("Error downloading from url: %s, %v\n", url, err)
		return
	}
	defer r.Body.Close()
	if r.StatusCode != 200 {
		if r.StatusCode == 429 {
			retry := r.Header.Get("X-Ratelimit-Retryafter")
			duration, _ := time.ParseDuration(retry)
			err = &ratelimitError{duration}
			return
		}
		log.Errorf("Error downloading from url %s, status code: %d", url, r.StatusCode)
		resp, _ := ioutil.ReadAll(bufio.NewReader(r.Body))
		err = fmt.Errorf("Got non 200 response for feed %s: %s", r.Status, resp)
		return
	}
	md5hash := md5.Sum([]byte(line))
	filename := fmt.Sprintf("%x", md5hash)
	base = filepath.Join(workingDirectory, url.Hostname(), filename)
	log.Infof("Path for url: %s =  %s", line, base)
	// file not exists
	isTemp = false
	if _, err = os.Stat(base); os.IsNotExist(err) {
		os.MkdirAll(filepath.Dir(base), os.ModePerm)
		var fw *os.File
		fw, err = os.Create(base)
		if err != nil {
			log.Errorf("Unable to create file %v\n", err)
			return
		}
		defer fw.Close()
		log.Info("Base file does not exist for url: ", line, "; creating", base)
		io.Copy(fw, r.Body)
	} else {
		// base file exists; write to temp
		isTemp = true
		var tmp *os.File
		tmp, err = ioutil.TempFile("", url.Hostname())
		if err != nil {
			log.Errorf("Unable to create temp file to download url: %s, %v", line, err)
			return
		}
		defer tmp.Close()
		tempfn = tmp.Name()
		log.Info("Base file exists; creating temp file: ", tempfn)
		io.Copy(tmp, r.Body)
	}
	return
}

func compareFeeds(xslt, base, temp string) ([]*gofeed.Item, error) {
	defer os.Remove(temp)
	baseXSLTParam := base
	if runtime.GOOS == "windows" {
		// xsltproc idiosyncracy on windows
		baseXSLTParam = strings.Replace(base, "\\", "/", -1)
	}
	log.Debugf("applying xslt %s to new file %s with base %s", xslt, temp, baseXSLTParam)
	cmd := exec.Command("xsltproc", "--stringparam", "originalfile", baseXSLTParam, xslt, temp)
	cmdStdoutPipe, _ := cmd.StdoutPipe()
	cmdStdErrPipe, _ := cmd.StderrPipe()
	cmd.Start()
	diff, err := ioutil.ReadAll(cmdStdoutPipe)
	stderr, err := ioutil.ReadAll(cmdStdErrPipe)
	cmd.Wait()
	if err != nil {
		log.Errorf("Error applying xslt: %v\n", err)
		return nil, err
	}
	if string(stderr) != "" {
		log.Warningf("xsltproc stderr: %s", stderr)
	}
	feedparser := gofeed.NewParser()
	feed, err := feedparser.ParseString(string(diff))
	if err != nil {
		log.Errorf("Could not parse feed, %v", err)
		return nil, err
	}
	if len(feed.Items) > 0 {
		log.Infof("Feed diff has %d new items", len(feed.Items))
		copyFile(temp, base)
	}
	return feed.Items, nil
}

func getTransformFile(line string) (string, error) {
	url, err := url.Parse(line)
	if err != nil {
		log.Errorf("Unable to parse url %v\n", err)
		return "", err
	}
	exePath, _ := os.Executable()
	exeFolder := filepath.Dir(exePath)
	xsltPath := filepath.Join(exeFolder, "assets", url.Hostname()+".xslt")
	if _, err := os.Stat(xsltPath); os.IsNotExist(err) {
		return "", fmt.Errorf("xslt %s does not exist", xsltPath)
	}
	return xsltPath, nil
}

func processFile(fn string) error {
	log.Debug("Processing file: ", fn)
	file, err := os.Open(fn)
	defer file.Close()

	if err != nil {
		log.Errorf("error opening file %v\n", err)
		return err
	}

	// Start reading from the file with a reader.
	reader := bufio.NewReader(file)

	var line string
	for {
		line, err = reader.ReadString('\n')
		if err == io.EOF {
			log.Infof("Completed processing file %s", fn)
			return nil
		}
		if err != nil {
			log.Error("Error reading line from file ", fn)
			continue
		}
		success := false
		retries := 0
		var basefile, tmpfile string
		var isTemp bool
		var err error
		for !success && retries < 3 {
			basefile, tmpfile, isTemp, err = downloadFile(line)
			if err == nil {
				success = true
			}
			if re, ok := err.(*ratelimitError); ok {
				log.Infof("Rate limited for %s - retrying after: %v at %v", line, re.retryDuration, time.Now().Add(re.retryDuration))
				retries++
				time.Sleep(re.retryDuration)
			}
		}
		if err != nil {
			log.Errorf("Error downloading: %s, %v", line, err)
			continue
		}
		// process the delta here
		log.Infof("File downloaded %s, %s, %b", basefile, tmpfile, isTemp)
		if !isTemp {
			log.Infof("Send push notification to acknowledge new feed url %s", line)

		} else {
			// compare temp with base
			// if new items found
			//		send pushes
			xslt, err := getTransformFile(line)
			if err != nil {

			}
			newItems, err := compareFeeds(xslt, basefile, tmpfile)
			if err != nil {
				log.Errorf("Error comparing feeds with xslt: %v", err)
				continue
			}
			if len(newItems) > 0 {
				for _, item := range newItems {
					for _, notifier := range notifiers {
						notifier.Notify(item)
					}
				}
			} else {
				log.Infof("No new items found in feed %s", line)
			}
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

	to, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		log.Errorf("Unable to open destination file %s, %v", dst, err)
	}
	defer to.Close()

	_, err = io.Copy(to, from)
	if err != nil {
		log.Errorf("Error while copying file %s -> %s, %v", src, dst, err)
	}
}
