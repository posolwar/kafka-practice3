package main

import (
	"flag"
	"log/slog"

	"practice3/internal/censore"
)

var (
	badWord     string
	replaceWord string
)

var brokers []string = []string{"localhost:9094", "localhost:9095", "localhost:9096"}

func init() {
	flag.StringVar(&badWord, "bad_word", "", "Плохое слово для фильтрации")
	flag.StringVar(&replaceWord, "replace_word", "", "Слово-замена для плохого слова")
}

func main() {
	flag.Parse()

	if badWord == "" || replaceWord == "" {
		slog.Error("Оба флага -bad_word и -replace_word обязательны")
		return
	}

	censore.NewFilterWord(
		brokers,
		censore.FilterWordsStream, // Используем новый топик
		&censore.AddFilterWordReq{
			BadWord:     badWord,
			ReplaceWord: replaceWord,
		},
	)
}
