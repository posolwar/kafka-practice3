package user

import "github.com/lovoo/goka"

var BlockListGroup goka.Group = "blocked-list"

type BlockUserReq struct {
	UserID      string `json:"user_id"`
	BlockUserID string `json:"block_user_id"`
}

type BlockList struct {
	BlockedUsers map[string][]string
}
