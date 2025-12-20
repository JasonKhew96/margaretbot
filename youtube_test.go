package main

import (
	"testing"

	"github.com/davecgh/go-spew/spew"
)

func Test_youtube(t *testing.T) {
	h, err := NewYoutubeHelper()
	if err != nil {
		t.Error(err)
	}

	videoList, err := h.service.Videos.List([]string{"snippet", "contentDetails", "liveStreamingDetails"}).Id("PWECV5Er1XE").Do()
	if err != nil {
		t.Error(err)
	}
	spew.Dump(videoList)
}

func Test_youtube_IsShort(t *testing.T) {
	h, err := NewYoutubeHelper()
	if err != nil {
		t.Error(err)
	}

	isShort, err := h.IsShort("5aqhv8qZAJA", "#shorts")
	if err != nil {
		t.Error(err)
	}
	t.Log(isShort)
}
