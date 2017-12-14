package main

import (
	"bufio"
	"github.com/mmcdole/gofeed"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"net/url"
)

type Notifier interface {
	NotifyItem(item *gofeed.Item)
	Notify(msg string)
}

type Pushover struct {
	token string
	user  string
}

func NewPushover(token, user string) *Pushover {
	var p Pushover
	p.token = token
	p.user = user
	return &p
}
func (p *Pushover) Notify(msg string) {
	data := make(url.Values)
	data["token"] = []string{p.token}
	data["user"] = []string{p.user}
	data["title"] = []string{"Feednotifier - message"}
	data["message"] = []string{msg}

	resp, err := http.PostForm("https://api.pushover.net/1/messages.json", data)
	if err != nil {
		log.Errorf("Error sending push notification %v", err)
	}
	defer resp.Body.Close()
	responseContent, _ := ioutil.ReadAll(bufio.NewReader(resp.Body))
	log.Debugf("Pushed %s - response: %s", item.Title, responseContent)
}
func (p *Pushover) NotifyItem(item *gofeed.Item) {

	data := make(url.Values)
	data["token"] = []string{p.token}
	data["user"] = []string{p.user}
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
