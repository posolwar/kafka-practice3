package message

import (
	"context"
	"encoding/json"
	"log"
	"log/slog"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/lovoo/goka"
)

var MessageReceiverGroup goka.Group = "messages-receiver"

type UserMessages struct {
	Messages []*Message `json:"messages"`
}

type UserMessagesCodec struct{}

func (c UserMessagesCodec) Encode(value interface{}) ([]byte, error) {
	return json.Marshal(value)
}

func (c UserMessagesCodec) Decode(data []byte) (interface{}, error) {
	var messages UserMessages
	err := json.Unmarshal(data, &messages)
	if err != nil {
		return nil, err
	}
	return &messages, nil
}

func processReceiver(ctx goka.Context, msg interface{}) {
	var userMessages *UserMessages

	message, ok := msg.(*Message)
	if !ok {
		log.Printf("expected *Message, got %T", msg)
		return
	}

	if val := ctx.Value(); val != nil {
		userMessages = val.(*UserMessages)
	} else {
		userMessages = &UserMessages{Messages: []*Message{}}
	}

	userMessages.Messages = append(userMessages.Messages, message)
	ctx.SetValue(userMessages)
}

func RunMessageReceiver(brokers []string, inputStream goka.Stream) {
	g := goka.DefineGroup(
		MessageReceiverGroup,
		goka.Input(inputStream, new(MessageCodec), processReceiver),
		goka.Persist(new(UserMessagesCodec)),
	)

	p, err := goka.NewProcessor(brokers, g)
	if err != nil {
		log.Fatal("Ошибка создания процессора: ", err)
	}

	if err := p.Run(context.Background()); err != nil {
		log.Fatal("Ошибка запуска процессора: ", err)
	}
}

func RunMessageView(brokers []string) {
	view, err := goka.NewView(brokers, goka.GroupTable(MessageReceiverGroup), new(UserMessagesCodec))
	if err != nil {
		slog.Error("Ошибка создания view для сообщений", slog.String("err", err.Error()))
		return
	}

	root := mux.NewRouter()
	root.HandleFunc("/messages/{user_id}", func(w http.ResponseWriter, r *http.Request) {
		userID := mux.Vars(r)["user_id"]
		value, err := view.Get(userID)
		if err != nil {
			http.Error(w, "Ошибка получения сообщений: "+err.Error(), http.StatusInternalServerError)
			return
		}

		var response []byte
		if value == nil {
			response = []byte(`{"messages": []}`)
		} else {
			response, err = json.Marshal(value)
			if err != nil {
				http.Error(w, "Ошибка сериализации данных: "+err.Error(), http.StatusInternalServerError)
				return
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(response)
	})

	slog.Info("Получение сообщений доступно на: http://localhost:9092/messages/{user_id}")

	go func() {
		err = http.ListenAndServe(":9092", root)
		if err != nil {
			log.Fatal("Ошибка запуска HTTP сервера: ", err)
		}
	}()

	err = view.Run(context.Background())
	if err != nil {
		log.Fatal("Ошибка запуска view: ", err)
	}
}
