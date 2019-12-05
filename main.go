package main

import (
	"github.com/galdor/go-cmdline"
	"github.com/jasonlvhit/gocron"
	log "github.com/sirupsen/logrus"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
)

var workingDirectory string
var notifiers []Notifier

func main() {
	cmdline := cliOptions()
	log.Info("/////////////////////////////////////////////////////////////")
	log.Info("****************** *Process Started* ************************")
	log.Info("/////////////////////////////////////////////////////////////")
	workingDirectory = cmdline.OptionValue("workingdir")
	pushoverToken := cmdline.OptionValue("pushover")
	notifiers = make([]Notifier, 0, 2)
	if pushoverToken != "" {
		tokenArr := strings.Split(pushoverToken, ":")
		po := NewPushover(tokenArr[0], tokenArr[1])
		notifiers = append(notifiers, po)
		log.Debug("added pushover notifier")
	}
	telegramToken := cmdline.OptionValue("telegram")
	if telegramToken != "" {
		tokenArr := strings.Split(telegramToken, "#")
		tele := NewTelegramNotifier(tokenArr[0], tokenArr[1])
		notifiers = append(notifiers, tele)
		log.Debug("added telegram notifier")
	}
	interval, _ := strconv.ParseUint(cmdline.OptionValue("interval"), 10, 64)
	log.Infof("Feeds will be checked at intervals of %d minutes", interval)
	files := cmdline.TrailingArgumentsValues("watchfile")
	log.Debug("watching files: ", files)
	for _, file := range files {
		watcher := NewMonitoredFile(file, interval)
		watcher.Start()
	}
	<-gocron.Start()
	log.Info("Completed process")
}

func cliOptions() *cmdline.CmdLine {

	cmdline := cmdline.New()
	cmdline.AddOption("w", "workingdir", "dir", "defaults to .feednotifier")
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	usr, err := user.Current()
	if err == nil {
		dir = usr.HomeDir
	}
	path := filepath.Join(dir, ".feednotifier")
	cmdline.SetOptionDefault("workingdir", path)
	cmdline.AddOption("l", "loglevel", "level", "debug, info, warn, error, fatal or panic")
	cmdline.SetOptionDefault("loglevel", "warn")
	cmdline.AddOption("f", "log-file", "file", "log file; logs to console if not specified")
	cmdline.AddOption("", "pushover", "pushover token", "pushover token app:user")
	cmdline.AddOption("", "telegram", "telegram bot and chat token", "telegram token - botid#chatid")
	cmdline.AddOption("i", "interval", "in minutes", "feeds will be checked at every X minutes")
	cmdline.SetOptionDefault("interval", "30")
	cmdline.AddTrailingArguments("watchfile", "files to watch and read rss feed urls from")
	cmdline.Parse(os.Args)
	levelname := cmdline.OptionValue("loglevel")
	logfilename := ""
	if cmdline.IsOptionSet("log-file") {
		logfilename = cmdline.OptionValue("log-file")
	}
	initLog(levelname, logfilename)
	return cmdline
}

func initLog(levelname, logfilename string) {
	formatter := &log.TextFormatter{
		FullTimestamp: true,
	}
	log.SetFormatter(formatter)
	level, e := log.ParseLevel(levelname)
	if e != nil {
		log.Panicf("Could not parse log level, exiting %v", e)
	}
	log.SetLevel(level)
	if logfilename != "" {
		os.MkdirAll(filepath.Dir(logfilename), os.ModePerm)
		logfile, e := os.OpenFile(logfilename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if e != nil {
			log.Panicf("Unable to open log file, bailing %v", e)
		}
		log.SetOutput(logfile)
	} else {
		log.SetOutput(os.Stdout)
	}
	log.Info("Log level set to: ", level)
}
