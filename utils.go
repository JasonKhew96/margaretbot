package main

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf16"

	"github.com/JasonKhew96/margaretbot/entityhelper"
	"github.com/PaulSonOfLars/gotgbot/v2"
)

var allMdV2 = []string{"_", "*", "[", "]", "(", ")", "~", "`", ">", "#", "+", "-", "=", "|", "{", "}", ".", "!"}
var mdV2Repl = strings.NewReplacer(func() (out []string) {
	for _, x := range allMdV2 {
		out = append(out, x, "\\"+x)
	}
	return out
}()...)

func EscapeMarkdownV2(s string) string {
	return mdV2Repl.Replace(s)
}

func Text2ExpandableQuote(text string) string {
	result := ""
	splits := strings.Split(text, "\n")
	for i, s := range splits {
		if i == 0 {
			result += "**>" + EscapeMarkdownV2(s) + "\n"
		} else if i == len(splits)-1 {
			result += ">" + EscapeMarkdownV2(s) + "||"
		} else {
			result += ">" + EscapeMarkdownV2(s) + "\n"
		}
	}
	return result
}

type Caption struct {
	VideoTitle string
	VideoUrl   string
	// ThumbnailUrl       string
	VideoDescription   string
	ChannelName        string
	ChannelUrl         string
	AllowedRegion      string
	BlockedRegion      string
	ScheduledStartTime string
	PublishedTime      string
	TimeZone           string
}

func getUtf16Len(s string) int64 {
	return int64(len(utf16.Encode([]rune(s))))
}

func BuildCaption(caption *Caption) (string, []gotgbot.MessageEntity) {

	msg := entityhelper.NewMessage()
	msg.AddEntity(fmt.Sprintf("%s\n", caption.VideoTitle), gotgbot.MessageEntity{
		Type: "text_link",
		Url:  caption.VideoUrl,
	})
	quotedText := entityhelper.NewMessage()
	quotedText.AddText("频道: ")
	quotedText.AddEntity(caption.ChannelName, gotgbot.MessageEntity{
		Type: "text_link",
		Url:  caption.ChannelUrl,
	})
	quotedText.AddText("\n")
	if len(caption.ScheduledStartTime) > 0 {
		quotedText.AddText("首播时间: ")
		parsedTime, err := time.Parse("2006-01-02T15:04:05Z", caption.ScheduledStartTime)
		if err != nil {
			quotedText.AddEntity(caption.ScheduledStartTime, gotgbot.MessageEntity{
				Type: "code",
			})
		} else {
			loc, _ := time.LoadLocation(caption.TimeZone)
			quotedText.AddEntity(parsedTime.In(loc).Format("02/01/2006 15:04:05 MST-07"), gotgbot.MessageEntity{
				Type: "code",
			})
		}
		quotedText.AddText("\n")
	} else {
		quotedText.AddText("发布时间: ")
		parsedTime, err := time.Parse("2006-01-02T15:04:05Z", caption.PublishedTime)
		if err != nil {
			quotedText.AddEntity(caption.PublishedTime, gotgbot.MessageEntity{
				Type: "code",
			})
		} else {
			loc, _ := time.LoadLocation(caption.TimeZone)
			quotedText.AddEntity(parsedTime.In(loc).Format("02/01/2006 15:04:05 MST-07"), gotgbot.MessageEntity{
				Type: "code",
			})
		}
		quotedText.AddText("\n")
	}
	if len(caption.AllowedRegion) > 0 {
		quotedText.AddText("允许地区: ")
		quotedText.AddEntity(caption.AllowedRegion, gotgbot.MessageEntity{
			Type: "code",
		})
		quotedText.AddText("\n")
	}
	if len(caption.BlockedRegion) > 0 {
		quotedText.AddText("屏蔽地区: ")
		quotedText.AddEntity(caption.BlockedRegion, gotgbot.MessageEntity{
			Type: "code",
		})
		quotedText.AddText("\n")
	}
	msg.AddNestedEntity(quotedText, gotgbot.MessageEntity{
		Type: "expandable_blockquote",
	})
	if len(caption.VideoDescription) > 0 {
		msg.AddEntity(caption.VideoDescription, gotgbot.MessageEntity{
			Type: "expandable_blockquote",
		})
	}

	return msg.GetText(), msg.GetEntities()
}
