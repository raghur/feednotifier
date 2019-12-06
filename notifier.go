package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"text/template"

	"github.com/mmcdole/gofeed"
	log "github.com/sirupsen/logrus"
)

var mdTmpl *template.Template

const defaultTemplate = "__message"

func init() {
	mdTmpl, _ = template.New(defaultTemplate).Parse(`
		Message: [{{.Title}}]({{.Link}})
		`)
}

func parseCustomTemplates(templates []string) {
	log.Debugf("Parsing custom templates, %v", templates)
	if len(templates) > 0 {
		custom, err := template.ParseFiles(templates...)
		if err != nil {
			log.Warnf("Error loading templates from files - %v", err)
			log.Warnf("Will use default template for all notifications")
			return
		}

		mdTmpl, _ = custom.AddParseTree(defaultTemplate, mdTmpl.Tree)
		log.Info(mdTmpl.DefinedTemplates())
		return
	}
}

type Notifier interface {
	NotifyItem(furl string, item *gofeed.Item)
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
func (p Pushover) String() string {
	return fmt.Sprintf("[PUSHOVER: %s]", p.user)
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
func (p *Pushover) NotifyItem(furl string, item *gofeed.Item) {

	data := make(url.Values)
	data["token"] = []string{p.token}
	data["user"] = []string{p.user}
	data["title"] = []string{item.Title}
	data["url"] = []string{item.Link}
	data["url_title"] = []string{"Add this torrent"}
	data["message"] = []string{renderItem(furl, item)}

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

func (p *TelegramNotifier) String() string {
	return fmt.Sprintf("[TELEGRAM:%s]", p.botId)
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

func (p *TelegramNotifier) NotifyItem(furl string, item *gofeed.Item) {

	data := make(url.Values)
	data["chat_id"] = []string{p.chatId}
	renderItem(furl, item)
	data["text"] = []string{renderItem(furl, item)}
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

func renderItem(furl string, item *gofeed.Item) string {
	u, _ := url.Parse(furl)
	templateName := u.Hostname()
	if t := mdTmpl.Lookup(templateName); t == nil {
		templateName = defaultTemplate
	}
	buf := bytes.NewBufferString("")
	err := mdTmpl.ExecuteTemplate(buf, templateName, item)
	if err != nil {
		mdTmpl.ExecuteTemplate(buf, defaultTemplate, item)
		return fmt.Sprintf("There was an error rendering message content - %v. Message is rendered with default template below: \n%s", err, buf.String())
	}
	return buf.String()
}
