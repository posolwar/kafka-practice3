package message

import (
	"log/slog"

	"github.com/lovoo/goka"
)

func SendMessage(brokers []string, topic goka.Stream, msg *Message) error {
	slog.Info("Отправка сообщения", "from_user_id", msg.FromUserID, "to_user_id", msg.ToUserID, "content", msg.Content)

	// Создаем эмиттер
	emitter, err := goka.NewEmitter(brokers, topic, new(MessageCodec))
	if err != nil {
		slog.Error("Ошибка создания эмиттера для отправки сообщения", "err", err.Error())
		return err
	}
	defer emitter.Finish()

	err = emitter.EmitSync(msg.ToUserID, msg)
	if err != nil {
		slog.Error("Ошибка отправки сообщения", "err", err.Error())
		return err
	}

	return nil
}
