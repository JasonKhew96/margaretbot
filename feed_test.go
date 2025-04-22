package main

import (
	"encoding/xml"
	"os"
	"testing"

	"github.com/JasonKhew96/margaretbot/entity"
	"github.com/davecgh/go-spew/spew"
)

func Test_feed(t *testing.T) {
	data, err := os.ReadFile("./data/feed.xml")
	if err != nil {
		t.Error(err)
	}
	feed := &entity.Feed{}
	if err := xml.Unmarshal(data, feed); err != nil {
		t.Error(err)
	}

	spew.Dump(feed)

}
