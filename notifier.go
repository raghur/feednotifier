package main

import (
	"bufio"
	"bytes"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"text/template"

	"github.com/mmcdole/gofeed"
	log "github.com/sirupsen/logrus"
)

var mdTmpl *template.Template

func init() {
	mdTmpl, _ = template.New("message").Parse(`
	Reddit Message from *{{.Author.Name}}* to [{{.Title}}]({{.Link}})
`)
}

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
	log.Debugf("Pushed %s - response: %s", msg, responseContent)
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

type TelegramNotifier struct {
	botId  string
	chatId string
}

func NewTelegramNotifier(botid, chatid string) *TelegramNotifier {
	var p TelegramNotifier
	p.botId = botid
	p.chatId = chatid
	return &p
}

func (p *TelegramNotifier) Notify(msg string) {
	data := make(url.Values)
	data["chat_id"] = []string{p.chatId}
	data["text"] = []string{msg}
	data["parse_mode"] = []string{"markdown"}

	url := strings.Replace("https://api.telegram.org/bot{}/sendMessage", "{}", p.botId, -1)
	resp, err := http.PostForm(url, data)
	if err != nil {
		log.Errorf("Error sending push notification %v", err)
	}
	defer resp.Body.Close()
	responseContent, _ := ioutil.ReadAll(bufio.NewReader(resp.Body))
	log.Debugf("Pushed %s - response: %s", msg, responseContent)
}

func (p *TelegramNotifier) NotifyItem(item *gofeed.Item) {

	data := make(url.Values)
	data["chat_id"] = []string{p.chatId}
	buf := bytes.NewBufferString("")
	mdTmpl.Execute(buf, item)
	data["text"] = []string{buf.String()}
	data["parse_mode"] = []string{"markdown"}
	//log.Debugf("Item:  %v", item)
	url := strings.Replace("https://api.telegram.org/bot{}/sendMessage", "{}", p.botId, -1)
	resp, err := http.PostForm(url, data)
	if err != nil {
		log.Errorf("Error sending push notification %v", err)
	}
	defer resp.Body.Close()
	responseContent, _ := ioutil.ReadAll(bufio.NewReader(resp.Body))
	log.Debugf("Pushed feed item- response: %s", responseContent)

}
