package main

import (
	"bufio"
	"crypto/md5"
	"fmt"
	"github.com/galdor/go-cmdline"
	"github.com/mmcdole/gofeed"
	"github.com/jasonlvhit/gocron"
	log "github.com/sirupsen/logrus"
	"io"
	"time"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
)

func initLog(cmdline *cmdline.CmdLine) {
	formatter := &log.TextFormatter{
		FullTimestamp: true,
	}
	log.SetFormatter(formatter)
	level, e := log.ParseLevel(cmdline.OptionValue("loglevel"))
	if e != nil {
		log.Panicf("Could not parse log level, exiting %v", e)
	}
	log.SetLevel(level)
	log.Info("Log level set to: ", level)
	logfilename := cmdline.OptionValue("log-file")
	os.MkdirAll(filepath.Dir(logfilename), os.ModePerm)
	logfile, e := os.OpenFile(logfilename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if e != nil {
		log.Panicf("Unable to open log file, bailing %v", e)
	}
	log.SetOutput(logfile)
}
type ratelimitError struct {
	retryDuration time.Duration
}
func (e *ratelimitError) Error() string {
	return fmt.Sprintf("Rate limited - retry after %v", e.retryDuration)
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
	log.Infof("Path for url: %s =  %s", line,  base)
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

func CopyFile(src, dst string) {
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
func compareFeeds(xslt, base, temp string) ([]*gofeed.Item, error) {
	defer os.Remove(temp)
	baseXSLTParam :=base
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
	stderr, err:= ioutil.ReadAll(cmdStdErrPipe)
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
		CopyFile(temp, base)
	} 
	return feed.Items, nil
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
					sendPush(item)
				}
			} else {
				log.Infof("No new items found in feed %s", line)
			}
		}

	}
}

func sendPush(item *gofeed.Item) {
	data := make(url.Values)
	token:= strings.Split(pushoverToken, ":")
	data["token"] = []string{token[0]}
	data["user"] = []string{token[1]}
	data["title"] = []string{item.Title}
	data["url"] = []string{item.Link}
	data["url_title"] = []string{"Add this torrent"}
	data["message"] = []string{item.Description}

	resp, err := http.PostForm("https://api.pushover.net/1/messages.json", data)
	if err != nil {
		log.Errorf("Error sending push notification %v", err)
	}
	defer resp.Body.Close()
	responseContent, _ := ioutil.ReadAll(bufio.NewReader(resp.Body))
	log.Debugf("Pushed %s - response: %s", item.Title, responseContent)
}
func initFileWatchers(files []string) {
	log.Debug("watching files: ", files)
	for _, file := range files {
		processFile(file)
		gocron.Every(10).Minutes().Do(processFile, file)
	}
}

var workingDirectory string
var pushoverToken string

func main() {

	cmdline := cmdline.New()
	cmdline.AddOption("w", "workingdir", "dir", "defaults to .feednotifier")
	usr, _ := user.Current()
	dir := usr.HomeDir
	path := filepath.Join(dir, ".feednotifier")
	cmdline.SetOptionDefault("workingdir", path)
	cmdline.AddOption("l", "loglevel", "level", "debug, info, warn, error, fatal or panic")
	cmdline.SetOptionDefault("loglevel", "warn")
	cmdline.AddOption("f", "log-file", "file", "log file")
	cmdline.SetOptionDefault("log-file", filepath.Join(path, "log.txt"))
	cmdline.AddOption("t", "token", "pushover token app:user", "pushover token")
	cmdline.AddTrailingArguments("watchfile", "files to watch and read rss feed urls from")
	cmdline.Parse(os.Args)
	initLog(cmdline)
	log.Info("Starting process")
	workingDirectory = cmdline.OptionValue("workingdir")
	pushoverToken = cmdline.OptionValue("token")
	initFileWatchers(cmdline.TrailingArgumentsValues("watchfile"))
	<-gocron.Start()
	log.Info("Completed process")
}
