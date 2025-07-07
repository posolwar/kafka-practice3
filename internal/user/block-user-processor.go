package user

import (
	"context"
	"encoding/json"
	"log"
	"log/slog"
	"net/http"
	"slices"

	"github.com/gorilla/mux"
	"github.com/lovoo/goka"
)

var (
	BlockUserGroup       goka.Group = "blocked-userss"
	ConsumerUserBlocking goka.Group = "user-blocking"
)

type BlockUserReq struct {
	UserID      string `json:"user_id"`
	IsBlock     bool   `json:"is_block"`
	BlockUserID string `json:"block_user_id"`
}

type BlockUserReqCodec struct{}

func (c BlockUserReqCodec) Encode(value interface{}) ([]byte, error) {
	return json.Marshal(value)
}

func (c BlockUserReqCodec) Decode(data []byte) (interface{}, error) {
	var req BlockUserReq
	err := json.Unmarshal(data, &req)
	if err != nil {
		return nil, err
	}
	return &req, nil
}

type BlockList struct{ BlockedUsers []string }

type BlockUserListCodec struct{}

func (c BlockUserListCodec) Encode(value interface{}) ([]byte, error) {
	return json.Marshal(value)
}

func (c BlockUserListCodec) Decode(data []byte) (interface{}, error) {
	var req BlockList
	err := json.Unmarshal(data, &req)
	if err != nil {
		return nil, err
	}
	return &req, nil
}

func process(ctx goka.Context, msg any) {
	var blockUsers *BlockList

	blockUserReq, ok := msg.(*BlockUserReq)
	if !ok || blockUserReq == nil {
		return
	}

	if blockUserReq.UserID == "" || blockUserReq.BlockUserID == "" {
		slog.Error("Пустой ID пользователя")
		return
	}

	if val := ctx.Value(); val != nil {
		blockUsers = val.(*BlockList)
	} else {
		blockUsers = &BlockList{BlockedUsers: []string{}}
	}

	// Проверяем, есть ли blockUserReq.BlockUserID в массиве blockUsers.BlockedUsers
	for i, id := range blockUsers.BlockedUsers {
		if id != blockUserReq.BlockUserID {
			continue
		}

		// Если нашли и пользователь не заблокирован - разблокируем
		if !blockUserReq.IsBlock {
			// удаляем из списка
			blockUsers.BlockedUsers = slices.Delete(blockUsers.BlockedUsers, i, i+1)
			ctx.SetValue(blockUsers)

			slog.Debug("Пользователь разблокирован",
				slog.String("Кто разблокировал", blockUserReq.UserID),
				slog.String("Кого разблокировали", blockUserReq.BlockUserID))

			return
		}

		// Если нашли юзера в списке блокировке и был запрос на блокировку - ничего не делаем
		if blockUserReq.IsBlock {
			slog.Warn("Попытка заблокировать уже заблокированного юзера",
				slog.String("Кто блокирует", blockUserReq.UserID),
				slog.String("Кого блокируют", blockUserReq.BlockUserID))

			return
		}
	}

	// Если при переборе ранее в списках юзер не был найден, и он должен быть заблокированны, то блокируем
	if blockUserReq.IsBlock {
		blockUsers.BlockedUsers = append(blockUsers.BlockedUsers, blockUserReq.BlockUserID)
		ctx.SetValue(blockUsers)

		slog.Debug("Добавление в список блокировки",
			slog.String("Кто добавил", blockUserReq.UserID),
			slog.String("Кого добавил", blockUserReq.BlockUserID))

	}

	// Пользователь не найден, а запрос на разблокировку — логируем предупреждение
	if !blockUserReq.IsBlock {
		slog.Warn("Попытка разблокировать отсутствующего в списке пользователя",
			slog.String("user_id", blockUserReq.UserID),
			slog.String("block_user_id", blockUserReq.BlockUserID),
		)
	}
}

func RunBlockProcess(brokers []string, inputStream goka.Stream) {
	g := goka.DefineGroup(
		BlockUserGroup,
		goka.Input(inputStream, new(BlockUserReqCodec), process),
		goka.Persist(new(BlockUserListCodec)),
	)

	p, err := goka.NewProcessor(
		brokers,
		g,
		// TODO: debug kafka bug
		// goka.WithStorageBuilder(storage.DefaultBuilder("/tmp/goka")),
	)
	if err != nil {
		log.Fatal(err)
	}

	if err := p.Run(context.Background()); err != nil {
		log.Fatal(err)
	}
}

func RunBlockUserView(brokers []string) {
	view, err := goka.NewView(brokers, goka.GroupTable(BlockUserGroup), new(BlockUserListCodec))
	if err != nil {
		slog.Error("Ошибка создания view для блокировки пользователей", slog.String("err", err.Error()))

		return
	}

	root := mux.NewRouter()
	root.HandleFunc("/block-list/{user_id}", func(w http.ResponseWriter, r *http.Request) {
		value, err := view.Get(mux.Vars(r)["user_id"])
		if err != nil {
			http.Error(w, "Ошибка получения данных: "+err.Error(), http.StatusInternalServerError)
			return
		}

		data, err := json.Marshal(value)
		if err != nil {
			http.Error(w, "Ошибка получения данных: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Write(data)
	})

	slog.Info("check list of blocked users on: http://localhost:9091/block-list/client_id")

	go func() {
		err = http.ListenAndServe(":9091", root)
		if err != nil {
			log.Fatal(err)
		}
	}()

	err = view.Run(context.Background())
	if err != nil {
		log.Fatal(err)
	}
}
