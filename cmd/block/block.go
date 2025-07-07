package main

import (
	"flag"
	"log/slog"

	"practice3/internal/user"

	"github.com/lovoo/goka"
)

var (
	UserID        string
	BlockedUserID string
	IsBlock       bool
)

var brockers []string = []string{"localhost:9094", "localhost:9095", "localhost:9096"}

func init() {
	flag.StringVar(&UserID, "user_id", "", "user id, пользователя, который инициирует (раз)блокировку")
	flag.StringVar(&BlockedUserID, "blocked_user_id", "", "user_id пользователя, который (раз)блокируется")
	flag.BoolVar(&IsBlock, "is_block", true, "true - блокировать, false - разблокировать")
}

func main() {
	flag.Parse()

	if err := user.SendBlockRequest(
		brockers,
		goka.Stream(user.BlockUserReqGroup),
		&user.BlockUserReq{
			UserID:      UserID,
			BlockUserID: BlockedUserID,
			IsBlock:     IsBlock,
		},
	); err != nil {
		slog.Error("ошибка отправки события на (раз)блокировку", "err", err.Error())
	}
}
