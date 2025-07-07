package censore

import (
	"encoding/json"
	"log/slog"

	"github.com/lovoo/goka"
)

var (
	WordFilterGroup   goka.Group  = "filtered-messages"
	FilterWordsStream goka.Stream = "filter-words-stream"
)

type AddFilterWordReq struct {
	BadWord     string `json:"bad_word"`
	ReplaceWord string `json:"replace_word"`
}

type FilterWordCodec struct{}

func (mc FilterWordCodec) Encode(value interface{}) ([]byte, error) {
	return json.Marshal(value)
}

func (mc FilterWordCodec) Decode(data []byte) (interface{}, error) {
	var req AddFilterWordReq
	err := json.Unmarshal(data, &req)
	if err != nil {
		return nil, err
	}
	return &req, nil
}

func NewFilterWord(brokers []string, topic goka.Stream, word *AddFilterWordReq) {
	emitter, err := goka.NewEmitter(brokers, topic, new(FilterWordCodec))
	if err != nil {
		slog.Error("ошибка создания эмиттера для добавления слова в фильтр", "err", err.Error())
		return
	}

	defer emitter.Finish()

	if err := emitter.EmitSync(word.BadWord, word); err != nil {
		slog.Error("ошибка синка при добавлении слова в фильтр", "badword", word.BadWord, "replaceword", word.ReplaceWord, "err", err.Error())
		return
	}
}
