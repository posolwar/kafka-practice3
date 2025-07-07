package main

import (
	"context"

	"practice3/internal/message"
	"practice3/internal/user"
)

var brokers = []string{"localhost:9094", "localhost:9095", "localhost:9096"}

func main() {
	// Запуск процессора фильтрации блокировок
	go message.RunBlockFilter(context.Background(), brokers, message.MessageStream, message.PreFilteredMessageStream)

	// Запуск процессора фильтрации слов
	go message.RunWordFilter(context.Background(), brokers, message.PreFilteredMessageStream, message.FilterMessageStream)

	// Запуск процессора получения сообщений
	go message.RunMessageReceiver(brokers, message.FilterMessageStream)

	// Запуск процессора блокировок
	go user.RunBlockProcess(brokers, user.BlockUserReqGroup)

	// Запуск view для блокировок
	go user.RunBlockUserView(brokers)

	// Запуск view для сообщений
	message.RunMessageView(brokers)
}
