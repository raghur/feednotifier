package main

import (
	"fmt"
	"github.com/mmcdole/gofeed"
	"os"
	"testing"
)

func TestFeedParseBasic(t *testing.T) {
	file, _ := os.Open("test/first.xml")
	defer file.Close()
	fp := gofeed.NewParser()
	feed, e := fp.Parse(file)
	if e != nil {
		fmt.Println(e)
		os.Exit(1)
	}
	fmt.Println(feed.Title)
	for _, e := range feed.Items {
		fmt.Println(e.Title)
		fmt.Println(e.Description)
		fmt.Println(e.GUID)
		fmt.Println(e.Link)
		fmt.Println(e.Published)
		fmt.Println(e.PublishedParsed)
		fmt.Println(e.Content)
		for k, v := range e.Custom {
			fmt.Println(k, v)
		}
	}
}
