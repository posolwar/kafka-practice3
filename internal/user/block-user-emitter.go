package user

import (
	"log/slog"

	"github.com/lovoo/goka"
)

var BlockUserReqGroup goka.Stream = "blocked-users-stream"

func SendBlockRequest(brokers []string, topic goka.Stream, req *BlockUserReq) error {
	slog.Info("Отправка запроса на (раз)блокировку", "req", req)

	// Создаем producer
	emitter, err := goka.NewEmitter(brokers, topic, new(BlockUserReqCodec))
	if err != nil {
		return err
	}
	defer emitter.Finish()

	// Отправляем сообщение с ключом = UserID (чтобы все запросы для одного пользователя попадали в одну партицию)
	err = emitter.EmitSync(req.UserID, req)
	if err != nil {
		return err
	}

	return nil
}
