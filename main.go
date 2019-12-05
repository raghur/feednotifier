package main

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/jasonlvhit/gocron"
	"github.com/jessevdk/go-flags"
	log "github.com/sirupsen/logrus"
)

var opts struct {
	LogLevel     string   `short:"l" long:"loglevel" default:"info" description:"Set log level" choice:"debug" choice:"info" choice:"warn" choice:"error" choice:"fatal" choice:"panic"`
	Interval     uint64   `short:"i" long:"interval" default:"30" description:"interval between checks" value-name:"MINUTES"`
	Logfile      string   `short:"f" long:"log" description:"log file" value-name:"FILE"`
	Notifier     []string `short:"n" long:"notifier" required:"1" description:"Attach a notifier - format type:value, can be specified multiple times" value-name:"notifierspec"`
	WorkingDir   string   `short:"w" long:"workingdir" default:"~/.feednotifier" description:"Working directory" value-name:"FOLDER"`
	WatchedFiles struct {
		Files []string `required:"yes" description:"Watched file(s) with RSS feeds - one feed per line" positional-arg-name:"FEED-FILE"`
	} `positional-args:"yes"`
	notifiers []Notifier
}

func main() {
	parseOptions()
	log.Info("/////////////////////////////////////////////////////////////")
	log.Info("****************** *Process Started* ************************")
	log.Info("/////////////////////////////////////////////////////////////")
	log.Infof("Feeds will be monitored every: %v mins", opts.Interval)
	log.Debug("watching files: ", opts.WatchedFiles.Files)
	for _, file := range opts.WatchedFiles.Files {
		watcher := NewMonitoredFile(file, opts.Interval)
		watcher.Start()
	}
	<-gocron.Start()
	log.Info("Completed process")
}

func parseOptions() []string {
	parser := flags.NewParser(&opts, flags.Default)
	args, err := parser.Parse()
	if err != nil {
		if e, ok := err.(*flags.Error); ok {
			if e.Type != flags.ErrHelp {
				parser.WriteHelp(os.Stdout)
			}
		}
		os.Exit(1)
	}
	if opts.WorkingDir == "~/.feednotifier" {
		dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
		usr, err := user.Current()
		if err == nil {
			dir = usr.HomeDir
		}
		opts.WorkingDir = filepath.Join(dir, ".feednotifier")
	}
	initLog(opts.LogLevel, opts.Logfile)
	for _, no := range opts.Notifier {
		parts := strings.SplitN(no, ":", 1)
		if parts == nil {
			fmt.Printf("Error parsing notifier - %s", no)
		}
		opts.notifiers = make([]Notifier, 2)
		switch parts[0] {
		case "telegram":
			tokenArr := strings.Split(parts[1], "#")
			opts.notifiers = append(opts.notifiers, NewTelegramNotifier(tokenArr[0], tokenArr[1]))
			break
		case "pushover":
			tokenArr := strings.Split(parts[1], ":")
			po := NewPushover(tokenArr[0], tokenArr[1])
			opts.notifiers = append(opts.notifiers, po)
		}
	}
	return args
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
