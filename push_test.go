
package main

import (
	"strings"
	"os"
	"testing"
	"net/url"
	"net/http"
	"github.com/mmcdole/gofeed"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"bufio"
)
func TestSendPush(t *testing.T) {
	pushoverToken:= "abc:def"
	file, _ := os.Open("test/third.xml")
	defer file.Close()
	fp := gofeed.NewParser()
	feed, _ := fp.Parse(file)
	item := feed.Items[0]
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