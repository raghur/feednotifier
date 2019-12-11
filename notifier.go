package feednotifier

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
	"github.com/raghur/feednotifier/static"
	log "github.com/sirupsen/logrus"
)

var mdTmpl *template.Template
var didInitTemplates bool

const defaultTemplate = "__message"

func initTemplates() {
	if didInitTemplates {
		return
	}
	defaultTmpl, _ := template.New(defaultTemplate).Parse(`
		Message: [{{.Title}}]({{.Link}})
		`)
	text, _ := static.ReadFile("assets/default.tmpl")
	s := string(text)
	mdTmpl, _ = template.New("embedded").Parse(s)
	mdTmpl.AddParseTree(defaultTemplate, defaultTmpl.Tree)
	log.Debugf("Default templates loaded are: %s", mdTmpl.DefinedTemplates())
	didInitTemplates = true
}

func ParseCustomTemplates(templates []string) {
	initTemplates()
	if len(templates) > 0 {
		log.Debugf("Parsing custom templates, %v", templates)
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

/* Notifier ...
 */
type Notifier interface {
	NotifyItem(furl string, item *gofeed.Item)
	Notify(msg string)
}

type pushover struct {
	token string
	user  string
}

func newPushover(token, user string) *pushover {
	var p pushover
	p.token = token
	p.user = user
	return &p
}
func (p pushover) String() string {
	return fmt.Sprintf("[PUSHOVER: %s]", p.user)
}

func (p *pushover) Notify(msg string) {
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
func (p *pushover) NotifyItem(furl string, item *gofeed.Item) {

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

type telegramNotifier struct {
	botId  string
	chatId string
}

func (p *telegramNotifier) String() string {
	return fmt.Sprintf("[TELEGRAM:%s]", p.botId)
}

func newTelegramNotifier(botid, chatid string) *telegramNotifier {
	var p telegramNotifier
	p.botId = botid
	p.chatId = chatid
	return &p
}

func (p *telegramNotifier) Notify(msg string) {
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

func (p *telegramNotifier) NotifyItem(furl string, item *gofeed.Item) {

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

func CreateNotifier(spec string) (Notifier, error) {
	initTemplates() // in case parse custom templates was never called? stinks.
	parts := strings.SplitN(spec, ":", 2)
	if parts == nil {
		return nil, fmt.Errorf("Error parsing notifier spec - %s", spec)
	}
	switch parts[0] {
	case "telegram":
		tokenArr := strings.Split(parts[1], "#")
		tele := newTelegramNotifier(tokenArr[0], tokenArr[1])
		return tele, nil
	case "pushover":
		tokenArr := strings.Split(parts[1], ":")
		po := newPushover(tokenArr[0], tokenArr[1])
		return po, nil
	}
	return nil, fmt.Errorf("Unknown spec format - %s", spec)
}
