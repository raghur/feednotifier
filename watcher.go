package feednotifier

import (
	"bufio"
	"crypto/md5"
	"fmt"
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

	"github.com/fsnotify/fsnotify"
	"github.com/jasonlvhit/gocron"
	"github.com/mmcdole/gofeed"
	"github.com/raghur/feednotifier/static"
	log "github.com/sirupsen/logrus"
)

var didInitXslts bool

func initXslt() {
	if didInitXslts {
		return
	}
	xslts, _ := static.WalkDirs("assets/xslt", false)
	log.Debugf("In built xslt transforms: %v", xslts)
	didInitXslts = true
}

type ratelimitError struct {
	retryDuration time.Duration
}

func (e *ratelimitError) Error() string {
	return fmt.Sprintf("Rate limited - retry after %v", e.retryDuration)
}

type FeedUrl struct {
	url      string
	savePath string
	added    time.Time
}

func (f FeedUrl) String() string {
	return fmt.Sprintf("%s", f.savePath)
}

type MonitoredFile struct {
	filename  string
	urls      map[string]FeedUrl
	interval  uint64
	watcher   *fsnotify.Watcher
	notifiers *[]Notifier
	basedir   string
}

func NewMonitoredFile(filename string, interval uint64, notifiers *[]Notifier, basedir string) *MonitoredFile {
	var mf MonitoredFile
	mf.filename = filename
	mf.interval = interval
	mf.urls = make(map[string]FeedUrl)
	mf.notifiers = notifiers
	mf.watcher, _ = fsnotify.NewWatcher()
	mf.basedir = basedir
	mf.initFile()
	initXslt()
	return &mf
}

func (mf *MonitoredFile) initFile() error {
	time := time.Now()
	err := ReadLines(mf.filename, " \r\n", func(line string) error {
		if line != "" {
			url, _ := url.Parse(line)
			md5hash := md5.Sum([]byte(line))
			filename := fmt.Sprintf("%x", md5hash)
			base := filepath.Join(mf.basedir, url.Hostname(), filename)
			_, exists := mf.urls[line]
			mf.urls[line] = FeedUrl{url: line, savePath: base, added: time}
			if !exists {
				processLine(line, mf.urls[line], *mf.notifiers)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	log.Debugf("Checking to see if there are any old urls to be cleaned")
	urlsRemovedNotification := ""
	for k, v := range mf.urls {
		if v.added.Before(time) {
			log.Debugf("Url %s not added now - will be deleted", k)
			os.Remove(v.savePath)
			log.Debugf("Removed file: %s", v.savePath)
			delete(mf.urls, k)
			urlsRemovedNotification = fmt.Sprintf("%s\nRemoved URL: %s", urlsRemovedNotification, k)
		}
	}
	if urlsRemovedNotification != "" {
		for _, notifier := range *mf.notifiers {
			notifier.Notify(urlsRemovedNotification)
		}
	}
	log.Debugf("Final list of %d urls to be monitored: %v", len(mf.urls), mf.urls)
	return nil
}

func (mf *MonitoredFile) Start() {
	job := func(f *MonitoredFile) {
		nextRun := time.Now().Add(time.Duration(f.interval) * time.Minute)
		log.Debug("Starting scheduled run: ")
		for line, value := range f.urls {
			processLine(line, value, *mf.notifiers)
		}
		log.Debugf("Completed scheduled run: Sleeping for %d minutes.", f.interval)
		log.Debugf("Next run at %v", nextRun)
		log.Info("*************************************")
	}
	cleanup := func() {
		log.Debugf("Removing scheduled task ")
		gocron.Remove(job)
		log.Debugf("Closing fs watcher")
		mf.watcher.Close()
	}

	mf.watcher.Add(mf.filename)
	debounceDuration := 1 * time.Second
	go func() {
		lastTriggered := time.Now()
		log.Debug("IN for loop waiting on channel event")
		for {
			select {
			case event := <-mf.watcher.Events:
				log.Println("event:", event)
				if time.Now().Sub(lastTriggered) > debounceDuration {
					log.Debug("modified file:", event.Name)
					lastTriggered = time.Now()

					// sometimes we get a remove event - though the file is being
					// edited - for ex. with Vim. In those cases:
					//	- initialize after a delay
					//  - readd the watch
					wasRemoveEvent := event.Op&fsnotify.Remove == fsnotify.Remove
					time.AfterFunc(500*time.Millisecond, func() {
						err := mf.initFile()
						if wasRemoveEvent {
							mf.watcher.Add(mf.filename)
						}
						if err != nil {
							log.Errorf("file %s could not be read. Error %v", mf.filename, err)
						}
					})
				}
			case err := <-mf.watcher.Errors:
				log.Debug("error while watching file:", err)
				break
			}
		}
		cleanup()
	}()
	gocron.Every(mf.interval).Minutes().Do(job, mf)
}

func downloadFile(line, base string) (tempfn string, err error) {
	url, err := url.Parse(line)
	if err != nil {
		log.Errorf("Unable to parse url %v\n", err)
		return
	}
	// "User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:73.0) Gecko/20100101 Firefox/73.0"
	client := &http.Client{}
	req, _ := http.NewRequest(http.MethodGet, url.String(), nil)
	req.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:73.0) Gecko/20100101 Firefox/73.0")
	r, err := client.Do(req)
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
	// file not exists
	tempfn = ""
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
	stderr, err := ioutil.ReadAll(cmdStdErrPipe)
	diff, err := ioutil.ReadAll(cmdStdoutPipe)
	err = cmd.Wait()

	if err != nil {
		log.Errorf("Error applying xslt: %v\n", err)
		if string(stderr) != "" {
			log.Warningf("xsltproc stderr: %s", stderr)
		}
		return nil, err
	}
	feedparser := gofeed.NewParser()

	feed, err := feedparser.ParseString(string(diff))

	if err != nil {
		log.Errorf("Could not parse feed, %v", err)
		return nil, err
	}

	return feed.Items, nil

}

func compareFeedsInProc(base, new string) ([]*gofeed.Item, error) {
	var idlist map[string]*gofeed.Item
	idlist = make(map[string]*gofeed.Item)
	fp := gofeed.NewParser()

	fh, err := os.Open(new)
	if err != nil {
		log.Errorf("Could not open new file - %s", new)
		return nil, err
	}
	defer fh.Close()

	newFeed, err := fp.Parse(fh)
	if err != nil {
		log.Errorf("Could not parse new file - %s, %v", new, err)
		return nil, err
	}
	for _, item := range newFeed.Items {
		idlist[item.GUID] = item
	}

	oldfh, err := os.Open(base)
	if err != nil {
		log.Errorf("Could not open base file - %s", base)
		return nil, err
	}
	defer oldfh.Close()
	oldfeed, err := fp.Parse(oldfh)
	if err != nil {
		log.Errorf("Could not parse base feed - %s, %v", base, err)
		return nil, err
	}
	for _, item := range oldfeed.Items {
		if _, found := idlist[item.GUID]; found {
			delete(idlist, item.GUID)
		}
	}

	var itemList []*gofeed.Item
	itemList = make([]*gofeed.Item, 0, len(idlist))
	for _, v := range idlist {
		itemList = append(itemList, v)
	}

	return itemList, nil
}
func getTransformFile(line string) (string, error) {
	url, err := url.Parse(line)
	if err != nil {
		log.Errorf("Unable to parse url %v\n", err)
		return "", err
	}
	fs := static.FS
	staticAsset := filepath.Join("assets", "xslt", url.Hostname()+".xslt")
	if _, err := fs.Stat(static.CTX, staticAsset); os.IsNotExist(err) {
		log.Debugf("transform file not available in embedded resource path: %s", staticAsset)
	}
	exePath, _ := os.Executable()
	exeFolder := filepath.Dir(exePath)
	xsltPath := filepath.Join(exeFolder, "assets", url.Hostname()+".xslt")
	if _, err := os.Stat(xsltPath); os.IsNotExist(err) {
		return "", fmt.Errorf("xslt %s does not exist", xsltPath)
	}
	return xsltPath, nil
}

func processLine(line string, value FeedUrl, notifiers []Notifier) error {
	success := false
	retries := 0
	var tmpfile string
	var err error
	for !success && retries < 3 {
		tmpfile, err = downloadFile(line, value.savePath)
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
		return nil
	}
	// process the delta here
	log.Infof("File downloaded %s, %s", value.savePath, tmpfile)
	if tmpfile == "" {
		log.Infof("Send push notification to acknowledge new feed url %s", line)
		for _, notifier := range notifiers {
			notifier.Notify(fmt.Sprintf("New url %s monitored. Base file %s", line, value.savePath))
		}
	} else {
		// compare temp with base
		// if new items found
		//		send pushes
		xslt, err := getTransformFile(line)
		var newItems []*gofeed.Item
		if err != nil {
			log.Warnf("Could not get transform file - %v", err)
			log.Info("Falling back to in proc comparison")
			newItems, err = compareFeedsInProc(value.savePath, tmpfile)
		} else {
			newItems, err = compareFeeds(xslt, value.savePath, tmpfile)
			if err != nil {
				log.Warnf("Error comparing feeds with xslt: %s,  %v", xslt, err)
				log.Info("Falling back to in proc comparison")
				newItems, err = compareFeedsInProc(value.savePath, tmpfile)
			}
		}
		if len(newItems) > 0 {
			defer os.Remove(tmpfile)
			log.Infof("Feed diff has %d new items", len(newItems))
			copyFile(tmpfile, value.savePath)

			log.Infof("Pushing %d new items found in feed %s", len(newItems), line)
			for _, item := range newItems {
				for _, notifier := range notifiers {
					notifier.NotifyItem(line, item)
				}
			}
		} else {
			log.Infof("No new items found in feed %s", line)
		}
	}
	return nil
}
