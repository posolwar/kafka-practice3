package message

import (
	"context"
	"log"
	"log/slog"

	"practice3/internal/user"

	"github.com/lovoo/goka"
)

func RunBlockFilter(ctx context.Context, brokers []string, inputTopic goka.Stream, outputTopic goka.Stream) {
	view, err := goka.NewView(brokers, goka.GroupTable(user.BlockUserGroup), new(user.BlockUserListCodec))
	if err != nil {
		log.Fatal("Ошибка создания view для блокировок: ", err)
	}

	go func() {
		if err := view.Run(context.Background()); err != nil {
			log.Fatalf("Ошибка запуска View: %v", err)
		}
	}()

	g := goka.DefineGroup(
		goka.Group("block-filter"),
		goka.Input(inputTopic, new(MessageCodec), func(ctx goka.Context, msg interface{}) {
			message, ok := msg.(*Message)
			if !ok {
				log.Printf("expected *Message, got %T", msg)
				return
			}

			blockList, err := view.Get(message.ToUserID)
			if err != nil {
				log.Printf("Ошибка получения списка блокировок для %s: %v", message.ToUserID, err)
				return
			}

			if blockList != nil {
				blockedUsers := blockList.(*user.BlockList).BlockedUsers
				for _, blockedID := range blockedUsers {
					if blockedID == message.FromUserID {
						slog.Debug("Заблокировано сообщение. Пользователь в блокировке.", slog.String("from", message.FromUserID), slog.String("to", message.ToUserID))
						return // Пользователь заблокирован, пропускаем сообщение
					}
				}
			}

			ctx.Emit(outputTopic, ctx.Key(), message)
		}),
		goka.Output(outputTopic, new(MessageCodec)),
	)

	p, err := goka.NewProcessor(brokers, g)
	if err != nil {
		log.Fatal("Ошибка создания процессора: ", err)
	}

	if err := p.Run(ctx); err != nil {
		log.Fatal("Ошибка запуска процессора: ", err)
	}
}
