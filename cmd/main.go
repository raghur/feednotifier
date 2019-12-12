package main

//go:generate fileb0x b0x.toml
import (
	"os"
	"path/filepath"

	"github.com/jasonlvhit/gocron"
	"github.com/jessevdk/go-flags"
	"github.com/mitchellh/go-homedir"
	"github.com/raghur/feednotifier"
	log "github.com/sirupsen/logrus"
)

var opts struct {
	LogLevel     string   `short:"l" long:"loglevel" default:"info" description:"Set log level" choice:"debug" choice:"info" choice:"warn" choice:"error" choice:"fatal" choice:"panic"`
	Interval     uint64   `short:"i" long:"interval" default:"30" description:"interval between checks" value-name:"MINUTES"`
	Logfile      string   `short:"f" long:"log" description:"log file" value-name:"FILE"`
	Notifier     []string `short:"n" long:"notifier" required:"1" description:"Attach a notifier - format type:value, can be specified multiple times" value-name:"notifierspec"`
	WorkingDir   string   `short:"w" long:"workingdir" default:"~/.feednotifier" description:"Working directory" value-name:"FOLDER"`
	Templates    []string `short:"t" long:"template" description:"Go template file for message rendering; multiple; Use domain name as template name to override default template" value-name:"TEMPLATE"`
	WatchedFiles struct {
		Files []string `required:"yes" description:"Watched file(s) with RSS feeds - one feed per line" positional-arg-name:"FEED-FILE"`
	} `positional-args:"yes"`
	notifiers []feednotifier.Notifier
	// Make sure to keep this as the last option - ordering of fields in this struct matters.
	Config func(string) `short:"c" long:"config" description:"ini formatted config file" default:"~/.feednotifier/feednotifier.ini" value-name:"CONFIG"`
}

func main() {
	parseOptions(os.Args[1:])
	log.Info("/////////////////////////////////////////////////////////////")
	log.Info("****************** *Process Started* ************************")
	log.Info("/////////////////////////////////////////////////////////////")
	log.Infof("Feeds will be monitored every: %v mins", opts.Interval)
	log.Infof("New items will be published to: %v", opts.notifiers)
	log.Infof("watching files: %v", opts.WatchedFiles.Files)
	for _, file := range opts.WatchedFiles.Files {
		watcher := feednotifier.NewMonitoredFile(file, opts.Interval, &opts.notifiers, opts.WorkingDir)
		watcher.Start()
	}
	<-gocron.Start()
	log.Debugf("Completed process")
}

func parseIniIfFound(file string, parser *flags.Parser) {
	log.Debugf("Start parsing ini file %s", file)
	iniParser := flags.NewIniParser(parser)
	iniParser.ParseAsDefaults = true
	path, _ := homedir.Expand(file)
	if err := iniParser.ParseFile(path); err != nil {
		log.Errorf("Error reading ini file - %v", err)
		if !os.IsNotExist(err) {
			log.Fatalf("Error reading ini file - %s, %v", path, err)
		}
		return
	}
	log.Infof("Read options from ini file - %s", path)
}

func parseOptions(args []string) []string {
	parser := flags.NewParser(&opts, flags.Default)
	opts.Config = func(file string) {
		parseIniIfFound(file, parser)
	}
	args, err := parser.ParseArgs(args)
	if err != nil {
		if e, ok := err.(*flags.Error); ok {
			if e.Type != flags.ErrHelp {
				parser.WriteHelp(os.Stdout)
			}
		}
		os.Exit(1)
	}
	initLog(opts.LogLevel, opts.Logfile)
	feednotifier.ParseCustomTemplates(opts.Templates)
	opts.WorkingDir, _ = homedir.Expand(opts.WorkingDir)
	log.Debugf("Working directory: %s", opts.WorkingDir)
	// log.Debugf("Now parsing notifiers %v", len(opts.Notifier))
	opts.notifiers = make([]feednotifier.Notifier, 0, 5)
	for _, no := range opts.Notifier {
		notifier, err := feednotifier.CreateNotifier(no)
		if err != nil {
			log.Fatalf("Error parsing notifier - %v", err)
		}
		opts.notifiers = append(opts.notifiers, notifier)
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
