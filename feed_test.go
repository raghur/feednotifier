package feednotifier

import (
	"os"
	"testing"

	"github.com/mmcdole/gofeed"
)

func TestFeedParseBasic(t *testing.T) {
	file, _ := os.Open("test/third.xml")
	defer file.Close()
	fp := gofeed.NewParser()
	feed, e := fp.Parse(file)
	if e != nil {
		t.Log(e)
		os.Exit(1)
	}
	t.Log(feed.Title)
	for _, e := range feed.Items {
		t.Log(e.Title)
		t.Log(e.Description)
		t.Log(e.GUID)
		t.Log(e.Link)
		t.Log(e.Published)
		// t.Log(e.PublishedParsed)
		t.Log(e.Content)
		for k, v := range e.Custom {
			t.Log(k, v)
		}
	}
}

func TestFeedParseZooqle(t *testing.T) {
	file, _ := os.Open("test/zooqle.first.xml")
	defer file.Close()
	fp := gofeed.NewParser()
	feed, e := fp.Parse(file)
	if e != nil {
		t.Log(e)
		os.Exit(1)
	}
	t.Log(feed.Title)
	for _, e := range feed.Items {
		t.Log(e.Title)
		t.Log(e.Description)
		t.Log(e.GUID)
		t.Log(e.Link)
		t.Log(e.Published)
		// t.Log(e.PublishedParsed)
		t.Log(e.Content)
		for k, v := range e.Custom {
			t.Log(k, v)
		}
		for k, v := range e.Extensions {
			t.Logf("%s => %v\n", k, v)
			for k2, v2 := range v {
				t.Logf("%s => %v\n", k2, v2[0].Value)
			}
		}
	}
}
