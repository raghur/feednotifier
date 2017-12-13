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
	Notify(item *gofeed.Item)
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

func (p *Pushover) Notify(item *gofeed.Item) {

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
