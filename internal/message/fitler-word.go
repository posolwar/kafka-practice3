package message

import (
	"context"
	"encoding/json"
	"log"
	"strings"

	"practice3/internal/censore"

	"github.com/lovoo/goka"
)

var (
	MessageStream            goka.Stream = "messages-stream"
	FilterMessageStream      goka.Stream = "filtered-messages-stream"
	PreFilteredMessageStream goka.Stream = "pre-filtered-messages-stream"
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
		return nil, err
	}
	return &req, nil
}

func process(outputTopic goka.Stream) func(ctx goka.Context, msg interface{}) {
	return func(ctx goka.Context, msg interface{}) {
		var filterWords *FilterWords

		if val := ctx.Value(); val != nil {
			filterWords = val.(*FilterWords)
		} else {
			filterWords = &FilterWords{WordFilter: map[string]string{}}
			ctx.SetValue(filterWords)
		}

		message, ok := msg.(*Message)
		if !ok {
			log.Printf("expected *Message, got %T", msg)
			return
		}

		filteredContent := message.Content
		for badWord, replacement := range filterWords.WordFilter {
			filteredContent = strings.ReplaceAll(filteredContent, badWord, replacement)
		}

		ctx.Emit(outputTopic, ctx.Key(), &Message{
			FromUserID: message.FromUserID,
			ToUserID:   message.ToUserID,
			Content:    filteredContent,
		})
	}
}

func processFilterWord(ctx goka.Context, msg interface{}) {
	var filterWords *FilterWords

	if val := ctx.Value(); val != nil {
		filterWords = val.(*FilterWords)
	} else {
		filterWords = &FilterWords{WordFilter: map[string]string{}}
		ctx.SetValue(filterWords)
	}

	wordReq, ok := msg.(*censore.AddFilterWordReq)
	if !ok {
		log.Printf("expected *AddFilterWordReq, got %T", msg)
		return
	}

	filterWords.WordFilter[wordReq.BadWord] = wordReq.ReplaceWord
	ctx.SetValue(filterWords)
}

func RunWordFilter(ctx context.Context, brokers []string, inputTopic goka.Stream, outputTopic goka.Stream) {
	g := goka.DefineGroup(
		censore.WordFilterGroup,
		goka.Input(inputTopic, new(MessageCodec), process(outputTopic)),
		goka.Input(censore.FilterWordsStream, new(censore.FilterWordCodec), processFilterWord),
		goka.Output(outputTopic, new(MessageCodec)),
		goka.Persist(new(FilterWordsCodec)),
	)

	p, err := goka.NewProcessor(brokers, g)
	if err != nil {
		log.Fatal(err)
	}

	err = p.Run(ctx)
	if err != nil {
		log.Fatal(err)
	}
}
