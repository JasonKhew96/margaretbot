package entityhelper

import (
	"unicode/utf16"

	"github.com/PaulSonOfLars/gotgbot/v2"
)

func getUtf16Len(s string) int64 {
	return int64(len(utf16.Encode([]rune(s))))
}

type Message struct {
	text     string
	entities []gotgbot.MessageEntity
}

func NewMessage() *Message {
	return &Message{
		text:     "",
		entities: []gotgbot.MessageEntity{},
	}
}

func (m *Message) AddText(text string) {
	m.text += text
}

func (m *Message) AddEntity(text string, messageEntity gotgbot.MessageEntity) {
	messageEntity.Offset = getUtf16Len(m.text)
	messageEntity.Length = getUtf16Len(text)
	m.text += text
	m.entities = append(m.entities, messageEntity)
}

func (m *Message) AddNestedEntity(msg *Message, messageEntity gotgbot.MessageEntity) {
	messageEntity.Offset = getUtf16Len(m.text)
	messageEntity.Length = getUtf16Len(msg.text)
	for i := range msg.entities {
		msg.entities[i].Offset += getUtf16Len(m.text)
	}
	m.text += msg.text
	m.entities = append(m.entities, messageEntity)
	m.entities = append(m.entities, msg.entities...)
}

func (m *Message) GetText() string {
	return m.text
}

func (m *Message) GetEntities() []gotgbot.MessageEntity {
	return m.entities
}
