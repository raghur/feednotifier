package feednotifier

import (
	"os"
	"testing"

	"github.com/mmcdole/gofeed"
)

func TestSendPush(t *testing.T) {
	pushoverToken := "pushover:abc:def"
	file, _ := os.Open("test/third.xml")
	defer file.Close()
	fp := gofeed.NewParser()
	feed, _ := fp.Parse(file)
	item := feed.Items[0]

	po, e := CreateNotifier(pushoverToken)
	if e != nil {
		t.Errorf("Unexpected error parsing token - %s", pushoverToken)
		t.Fail()
	}
	po.NotifyItem("www.somewhere.com/invalid/url", item)
}
