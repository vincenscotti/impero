package tgui

import (
	"context"
	"github.com/vincenscotti/impero/engine"
	. "github.com/vincenscotti/impero/model"
	"gopkg.in/telegram-bot-api.v4"
	"log"
	"os"
)

type TGUI struct {
	bot           *tgbotapi.BotAPI
	e             *engine.Engine
	token         string
	webURL        string
	logger        *log.Logger
	pendingChats  map[int64]*ChatState
	chatToPlayer  map[int64]*Player
	playerToChat  map[uint]int64
	notifications chan (gameNotification)
	quit          chan (bool)
}

type ChatState struct {
	State    int
	Username string
}

const (
	StateWaitUsername = iota
	StateWaitPassword = iota
)

func New(e *engine.Engine, token string, weburl string) (tg *TGUI) {
	tg = new(TGUI)

	tg.e = e
	tg.token = token
	tg.webURL = weburl
	tg.logger = log.New(os.Stdout, "tgui: ", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)
	tg.pendingChats = make(map[int64]*ChatState)
	tg.chatToPlayer = make(map[int64]*Player)
	tg.playerToChat = make(map[uint]int64)
	tg.notifications = make(chan gameNotification, 1000)
	tg.quit = make(chan bool, 1)

	return
}

func (tg *TGUI) Shutdown(ctx context.Context) error {
	tg.quit <- true

	select {
	case <-ctx.Done():
		return ctx.Err()

	case <-tg.quit:
		return nil
	}
}
