module github.com/raghur/feednotifier

go 1.13

require (
	github.com/PuerkitoBio/goquery v1.5.0 // indirect
	github.com/andybalholm/cascadia v1.1.0 // indirect
	github.com/fsnotify/fsnotify v1.4.7
	github.com/jasonlvhit/gocron v0.0.0-20191125235832-30e323a962ed
	github.com/jessevdk/go-flags v1.4.0
	github.com/konsorten/go-windows-terminal-sequences v1.0.2 // indirect
	github.com/mitchellh/go-homedir v1.1.0
	github.com/mmcdole/gofeed v1.0.0-beta2
	github.com/mmcdole/goxpp v0.0.0-20181012175147-0068e33feabf // indirect
	github.com/sirupsen/logrus v1.4.2
	golang.org/x/net v0.0.0-20191209160850-c0dbc17a3553
	golang.org/x/sys v0.0.0-20191210023423-ac6580df4449 // indirect
	golang.org/x/text v0.3.2 // indirect
	gopkg.in/yaml.v2 v2.2.4 // indirect
)

replace github.com/jessevdk/go-flags v1.4.0 => github.com/raghur/go-flags v1.4.1-0.20191206051701-ed0e0cba599e
