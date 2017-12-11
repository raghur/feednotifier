package main

import (
	"bufio"
	"crypto/md5"
	"fmt"
	"github.com/galdor/go-cmdline"
	log "github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
)

func initLog(cmdline *cmdline.CmdLine) {
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
	md5hash := md5.Sum([]byte(line))
	filename := fmt.Sprintf("%x", md5hash)
	base = filepath.Join(workingDirectory, url.Hostname(), filename)
	log.Info("Path for url:", line, ":", base)
	// file not exists
	isTemp = false
	if _, err = os.Stat(base); os.IsNotExist(err) {
		os.MkdirAll(filepath.Dir(base), os.ModePerm)
		fw, err := os.Create(base)
		if err != nil {
			log.Errorf("Unable to create file %v\n", err)
		}
		defer fw.Close()
		log.Info("Base file does not exist for url: ", line, "; creating")
		io.Copy(fw, r.Body)
	} else {
		// file exists
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
			return nil
		}
		if err != nil {
			log.Error("Error reading line from file ", file)
		}
		basefile, tmpfile, isTemp, err := downloadFile(line)
		if err != nil {
			log.Error("Error downloading line from file ", line)
		}
		// process the delta here
		log.Info("File downloaded", basefile, tmpfile, isTemp)

	}
}
func initFileWatchers(files []string) {
	log.Debug("watching files: ", files)
	for _, file := range files {
		processFile(file)
	}
}

var workingDirectory string

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
	cmdline.AddTrailingArguments("watchfile", "files to watch and read rss feed urls from")
	cmdline.Parse(os.Args)
	initLog(cmdline)
	log.Info("Starting process")
	workingDirectory = cmdline.OptionValue("workingdir")
	initFileWatchers(cmdline.TrailingArgumentsValues("watchfile"))
	log.Info("Completed process")
	// fp := gofeed.NewParser()
	// feed, e := fp.ParseURL("https://www.skytorrents.in/rss/all/ad/1/blue%20planet%20II%20%20hevc%201080p%20psa")
	// if e != nil {
	// 	fmt.Println(e)
	// 	os.Exit(1)
	// }
	// fmt.Println(feed.Title)
	// log.Debug("Feed items: ", len(feed.Items))
	// for _, e := range feed.Items {
	// 	fmt.Println(e.Title)
	// 	fmt.Println(e.GUID)
	// 	fmt.Println(e.Link)
	// 	// fmt.Println(e.Content)
	// 	// for k, v := range e.Custom {
	// 	// 	fmt.Println(k, v)
	// 	// }
	// }
}
