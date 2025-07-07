package message

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"

	"practice3/internal/censore"

	"github.com/lovoo/goka"
)

var (
	MessageStream            goka.Stream = "messages-stream"
	FilterMessageStream      goka.Stream = "filtered-words-stream"
	PreFilteredMessageStream goka.Stream = "pre-filtered-words-stream"
)

type FilterWords struct {
	WordFilter map[string]string // плохое слово: замена
}

type FilterWordsCodec struct{}

func (fwc FilterWordsCodec) Encode(value interface{}) ([]byte, error) {
	return json.Marshal(value)
}

func (fwc FilterWordsCodec) Decode(data []byte) (interface{}, error) {
	var req FilterWords
	err := json.Unmarshal(data, &req)
	if err != nil {
		slog.Debug("Ошибка декодирования FilterWords", "error", err, "data", string(data))
		return nil, err
	}
	return &req, nil
}

type Message struct {
	FromUserID string `json:"from_user_id"`
	ToUserID   string `json:"to_user_id"`
	Content    string `json:"content"`
}

type MessageCodec struct{}

func (mc MessageCodec) Encode(value interface{}) ([]byte, error) {
	return json.Marshal(value)
}

func (mc MessageCodec) Decode(data []byte) (interface{}, error) {
	var req Message
	err := json.Unmarshal(data, &req)
	if err != nil {
		slog.Debug("Ошибка декодирования Message", "error", err, "data", string(data))
		return nil, err
	}
	return &req, nil
}

func process(view *goka.View, outputTopic goka.Stream) func(ctx goka.Context, msg interface{}) {
	return func(ctx goka.Context, msg interface{}) {
		slog.Debug("Обработка сообщения", "key", ctx.Key(), "msg", msg)
		message, ok := msg.(*Message)
		if !ok {
			slog.Debug("Ожидаю *Message", "got", slog.Any("type", msg))
			return
		}

		filteredContent := message.Content
		// Разбиваем содержимое на слова
		words := strings.Fields(message.Content)
		for _, word := range words {
			// Запрашиваем фильтр для каждого слова через View
			val, err := view.Get(word)
			if err != nil {
				slog.Debug("Ошибка получения фильтра для слова", "word", word, "error", err)
				continue
			}
			if val != nil {
				filterWords := val.(*FilterWords)
				if replacement, exists := filterWords.WordFilter[word]; exists {
					filteredContent = strings.ReplaceAll(filteredContent, word, replacement)
					slog.Debug("Замена слова", "word", word, "replacement", replacement)
				}
			}
		}

		ctx.Emit(outputTopic, ctx.Key(), &Message{
			FromUserID: message.FromUserID,
			ToUserID:   message.ToUserID,
			Content:    filteredContent,
		})
	}
}

func processAddFilterWord(ctx goka.Context, msg interface{}) {
	slog.Debug("Обработка сообщения", "key", ctx.Key())
	var filterWords *FilterWords

	if val := ctx.Value(); val != nil {
		filterWords = val.(*FilterWords)
		slog.Debug("Текущие данные для ключа", "key", ctx.Key(), "data", filterWords)
	} else {
		filterWords = &FilterWords{WordFilter: map[string]string{}}
		slog.Debug("Создан новый FilterWords для ключа", "key", ctx.Key())
	}

	wordReq, ok := msg.(*censore.AddFilterWordReq)
	if !ok {
		slog.Debug("Ожидаю *AddFilterWordReq", "got", slog.Any("type", msg))
		return
	}

	filterWords.WordFilter[wordReq.BadWord] = wordReq.ReplaceWord
	ctx.SetValue(filterWords)
	slog.Debug("Обновлены данные для ключа", "key", ctx.Key(), "data", filterWords)
}

func RunWordFilter(ctx context.Context, brokers []string, inputTopic goka.Stream, outputTopic goka.Stream) {
	// Создаем View для чтения таблицы группы
	view, err := goka.NewView(brokers, goka.GroupTable(censore.WordFilterGroup), new(FilterWordsCodec))
	if err != nil {
		slog.Error("Ошибка создания View", "error", err)
		return
	}
	go func() {
		if err := view.Run(ctx); err != nil {
			slog.Error("Ошибка запуска View", "error", err)
			return
		}
	}()

	g := goka.DefineGroup(
		censore.WordFilterGroup,
		goka.Input(inputTopic, new(MessageCodec), process(view, outputTopic)),
		goka.Input(censore.FilterWordsStream, new(censore.FilterWordCodec), processAddFilterWord),
		goka.Output(outputTopic, new(MessageCodec)),
		goka.Persist(new(FilterWordsCodec)),
	)

	p, err := goka.NewProcessor(brokers, g)
	if err != nil {
		slog.Error("Ошибка создания процессора", "error", err)
		return
	}

	err = p.Run(ctx)
	if err != nil {
		slog.Error("Ошибка запуска процессора", "error", err)
	}
}
