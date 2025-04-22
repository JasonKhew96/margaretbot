package main

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/JasonKhew96/margaretbot/entityhelper"
	"github.com/PaulSonOfLars/gotgbot/v2"
)

func Test_Entities(t *testing.T) {
	expected := []gotgbot.MessageEntity{
		{
			Type:   "blockquote",
			Offset: 11,
			Length: 21,
		},
		{
			Type:   "strikethrough",
			Offset: 14,
			Length: 4,
		},
		{
			Type:   "text_link",
			Offset: 22,
			Length: 2,
			Url:    "https://google.com/",
		},
		{
			Type:   "spoiler",
			Offset: 34,
			Length: 9,
		},
	}
	/*
	   测试 TEST 🤔
	   测试 TEST 🤔
	   测试 TEST 🤔
	   测试 TEST 🤔
	*/
	entities := []gotgbot.MessageEntity{}

	tmpLen := getUtf16Len("测试 TEST 🤔\n")

	entities = append(entities, gotgbot.MessageEntity{
		Type:   "blockquote",
		Offset: tmpLen,
		Length: getUtf16Len("测试 TEST 🤔\n测试 TEST 🤔"),
	})

	tmpLen += getUtf16Len("测试 ")
	entities = append(entities, gotgbot.MessageEntity{
		Type:   "strikethrough",
		Offset: tmpLen,
		Length: getUtf16Len("TEST"),
	})

	tmpLen += getUtf16Len("TEST 🤔\n")
	entities = append(entities, gotgbot.MessageEntity{
		Type:   "text_link",
		Offset: tmpLen,
		Length: getUtf16Len("测试"),
		Url:    "https://google.com/",
	})

	tmpLen += getUtf16Len("测试 TEST 🤔\n测")
	entities = append(entities, gotgbot.MessageEntity{
		Type:   "spoiler",
		Offset: tmpLen,
		Length: getUtf16Len("试 TEST 🤔"),
	})

	if !reflect.DeepEqual(entities, expected) {
		t.Errorf("Expected %v, got %v", expected, entities)
	}
}

func Test_Entities2(t *testing.T) {
	expected := []gotgbot.MessageEntity{
		{
			Type:   "blockquote",
			Offset: 11,
			Length: 21,
		},
		{
			Type:   "strikethrough",
			Offset: 14,
			Length: 4,
		},
		{
			Type:   "text_link",
			Offset: 22,
			Length: 2,
			Url:    "https://google.com/",
		},
		{
			Type:   "spoiler",
			Offset: 34,
			Length: 9,
		},
	}
	/*
	   测试 TEST 🤔
	   测试 TEST 🤔
	   测试 TEST 🤔
	   测试 TEST 🤔
	*/
	msg := entityhelper.NewMessage()
	msg.AddText("测试 TEST 🤔\n")

	msg2 := entityhelper.NewMessage()
	msg2.AddText("测试 ")
	msg2.AddEntity("TEST", gotgbot.MessageEntity{
		Type: "strikethrough",
	})
	msg2.AddText(" 🤔\n")
	msg2.AddEntity("测试", gotgbot.MessageEntity{
		Type: "text_link",
		Url:  "https://google.com/",
	})
	msg2.AddText(" TEST 🤔")

	fmt.Println("--------")
	fmt.Println(msg2.GetText())
	fmt.Println(msg2.GetEntities())
	fmt.Println("--------")

	msg.AddNestedEntity(msg2, gotgbot.MessageEntity{
		Type: "blockquote",
	})
	msg.AddText("\n测")
	msg.AddEntity("试 TEST 🤔", gotgbot.MessageEntity{
		Type: "spoiler",
	})
	if !reflect.DeepEqual(msg.GetEntities(), expected) {
		t.Errorf("Expected %v, got %v", expected, msg.GetEntities())
	}
}
