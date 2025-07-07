package main

import (
	"flag"
	"log/slog"

	"practice3/internal/message"
)

var (
	FromUserID string
	ToUserID   string
	Content    string
)

var brokers []string = []string{"localhost:9094", "localhost:9095", "localhost:9096"}

func init() {
	flag.StringVar(&FromUserID, "from_user_id", "", "ID отправителя")
	flag.StringVar(&ToUserID, "to_user_id", "", "ID получателя")
	flag.StringVar(&Content, "content", "", "Текст сообщения")
}

func main() {
	flag.Parse()

	if FromUserID == "" || ToUserID == "" || Content == "" {
		slog.Error("Не указаны обязательные параметры: from_user_id, to_user_id, content")
		return
	}

	if err := message.SendMessage(
		brokers,
		message.MessageStream,
		&message.Message{
			FromUserID: FromUserID,
			ToUserID:   ToUserID,
			Content:    Content,
		},
	); err != nil {
		slog.Error("Ошибка отправки сообщения", "err", err.Error())
	}
}
