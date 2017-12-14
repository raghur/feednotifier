package main

import (
	"github.com/mmcdole/gofeed"
	"os"
	"strings"
	"testing"
)

func TestSendPush(t *testing.T) {
	pushoverToken := "abc:def"
	tokenParts := strings.Split(pushoverToken, ":")
	file, _ := os.Open("test/third.xml")
	defer file.Close()
	fp := gofeed.NewParser()
	feed, _ := fp.Parse(file)
	item := feed.Items[0]

	po := NewPushover(tokenParts[0], tokenParts[1])
	po.NotifyItem(item)
}
