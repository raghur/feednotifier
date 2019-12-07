package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/jasonlvhit/gocron"
	"github.com/jessevdk/go-flags"
	"github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
)

var opts struct {
	Config       func(string) `short:"c" long:"config" description:"ini formatted config file" default:"~/.feednotifier/feednotifier.ini" value-name:"CONFIG"`
	LogLevel     string       `short:"l" long:"loglevel" default:"info" description:"Set log level" choice:"debug" choice:"info" choice:"warn" choice:"error" choice:"fatal" choice:"panic"`
	Interval     uint64       `short:"i" long:"interval" default:"30" description:"interval between checks" value-name:"MINUTES"`
	Logfile      string       `short:"f" long:"log" description:"log file" value-name:"FILE"`
	Notifier     []string     `short:"n" long:"notifier" required:"1" description:"Attach a notifier - format type:value, can be specified multiple times" value-name:"notifierspec"`
	WorkingDir   string       `short:"w" long:"workingdir" default:"~/.feednotifier" description:"Working directory" value-name:"FOLDER"`
	Templates    []string     `short:"t" long:"template" description:"Go template file for message rendering; multiple; Use domain name as template name to override default template" value-name:"TEMPLATE"`
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
	log.Debugf("Feeds will be monitored every: %v mins", opts.Interval)
	log.Debugf("New items will be published to: %v", opts.notifiers)
	log.Debugf("watching files: %v", opts.WatchedFiles.Files)
	for _, file := range opts.WatchedFiles.Files {
		watcher := NewMonitoredFile(file, opts.Interval)
		watcher.Start()
	}
	<-gocron.Start()
	log.Debugf("Completed process")
}

func parseIniIfFound(file string, parser *flags.Parser) {
	log.Infof("parsing ini file %s", file)
	iniParser := flags.NewIniParser(parser)
	iniParser.ParseAsDefaults = false
	path, _ := homedir.Expand(file)
	if err := iniParser.ParseFile(path); err != nil {
		log.Debugf("Error reading ini file - %v", err)
		if !os.IsNotExist(err) {
			log.Fatalf("Error reading ini file - %s, %v", path, err)
		}
		return
	}
	log.Infof("Read options from ini file - %s", path)
}
func parseOptions() []string {
	parser := flags.NewParser(&opts, flags.Default)
	opts.Config = func(config string) {
		parseIniIfFound(config, parser)
	}
	args, err := parser.Parse()
	if err != nil {
		if e, ok := err.(*flags.Error); ok {
			if e.Type != flags.ErrHelp {
				parser.WriteHelp(os.Stdout)
			}
		}
		os.Exit(1)
	}
	initLog(opts.LogLevel, opts.Logfile)
	parseCustomTemplates(opts.Templates)
	opts.WorkingDir, _ = homedir.Expand(opts.WorkingDir)
	log.Debugf("Working directory: %s", opts.WorkingDir)
	// log.Debugf("Now parsing notifiers %v", len(opts.Notifier))
	opts.notifiers = make([]Notifier, 0, 5)
	for _, no := range opts.Notifier {
		parts := strings.SplitN(no, ":", 2)
		if parts == nil {
			log.Fatalf("Error parsing notifier spec - %s", no)
			os.Exit(1)
		}
		switch parts[0] {
		case "telegram":
			tokenArr := strings.Split(parts[1], "#")
			tele := NewTelegramNotifier(tokenArr[0], tokenArr[1])
			opts.notifiers = append(opts.notifiers, tele)
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
