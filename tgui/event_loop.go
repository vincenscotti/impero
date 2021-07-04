package tgui

import (
	"fmt"

	. "github.com/vincenscotti/impero/model"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

func (tg *TGUI) Run(debug bool) (err error) {
	tg.bot, err = tgbotapi.NewBotAPI(tg.token)
	if err != nil {
		return
	}

	tg.bot.Debug = debug

	tg.logger.Printf("Authorized on account %s", tg.bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := tg.bot.GetUpdatesChan(u)

	for {
		select {
		case update := <-updates:
			if update.Message == nil {
				continue
			}

			tg.logger.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")

			if state, ok := tg.pendingChats[update.Message.Chat.ID]; !ok {
				if update.Message.Command() != "start" && update.Message.Command() != "register" {
					msg.Text = "Ciao! Sono il bot telegram di Impero! Visto che sono " +
						"giovane e frusulone, non so fare ancora molte cose. Posso " +
						"aggiornarti sulle aste a cui hai partecipato e sulla fine dei " +
						"turni.\n\n"

					if p, ok := tg.chatToPlayer[update.Message.Chat.ID]; ok {
						msg.Text += "Ricevi le notifiche dell'utente " + p.Name
					} else {
						msg.Text += "Per iscriverti alle notifiche, scrivi /register"
					}
				} else {
					// show usernames keyboard
					tx := tg.e.OpenSession()
					_, players := tx.GetPlayers()
					tx.Close()

					rows := make([][]tgbotapi.KeyboardButton, 0, len(players))
					for _, p := range players {
						rows = append(rows, tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton(p.Name)))
					}

					kbd := tgbotapi.NewReplyKeyboard(rows...)
					kbd.OneTimeKeyboard = true
					msg.ReplyMarkup = kbd
					msg.Text = "Seleziona il tuo nome nel gioco"

					// update chat state
					tg.pendingChats[update.Message.Chat.ID] = &ChatState{State: StateWaitUsername}

					// remove old subscription
					if oldp, ok := tg.chatToPlayer[update.Message.Chat.ID]; ok {
						delete(tg.chatToPlayer, update.Message.Chat.ID)
						delete(tg.playerToChat, oldp.ID)
					}
				}
			} else if state.State == StateWaitUsername {
				state.Username = update.Message.Text
				msg.Text = "Ora scrivi la password dell'utente selezionato"
				state.State = StateWaitPassword
			} else if state.State == StateWaitPassword {
				tx := tg.e.OpenSession()
				p := &Player{Name: state.Username, Password: update.Message.Text}
				err, _, _ := tx.LoginPlayer(p)
				tx.Close()

				if err == nil {
					msg.Text = "Registrato come " + state.Username

					tg.chatToPlayer[update.Message.Chat.ID] = p
					tg.playerToChat[p.ID] = update.Message.Chat.ID
				} else {
					msg.Text = err.Error() + "\n\nPer riprovare scrivi /register"
				}

				delete(tg.pendingChats, update.Message.Chat.ID)
			}

			tg.bot.Send(msg)

		case n := <-tg.notifications:
			switch value := n.(type) {
			case endTurnNotification:
				for chatid, _ := range tg.chatToPlayer {
					msg := tgbotapi.NewMessage(chatid, "E' finito il turno!\n\nClicca <a href=\""+tg.webURL+"/game/report/all/\">qui</a> per leggere i report")
					msg.ParseMode = tgbotapi.ModeHTML
					tg.bot.Send(msg)
				}
			case auctionRaiseNotification:
				for _, p := range value.players {
					if chatid, ok := tg.playerToChat[p.ID]; ok && value.auction.HighestOfferPlayerID != p.ID {
						msg := tgbotapi.NewMessage(chatid, fmt.Sprint("L'asta per l'azione della societa' ", value.auction.Company.Name, " e' stata rialzata a ", value.auction.HighestOffer/100, " $!\n\nClicca <a href=\""+tg.webURL+"/game/market/\">qui</a> per controllare il mercato!"))
						msg.ParseMode = tgbotapi.ModeHTML
						tg.bot.Send(msg)
					}
				}
			case auctionEndNotification:
				for _, p := range value.players {
					if chatid, ok := tg.playerToChat[p.ID]; ok {
						name := "te"
						if value.auction.HighestOfferPlayerID != p.ID {
							name = value.auction.HighestOfferPlayer.Name
						}

						msg := tgbotapi.NewMessage(chatid, fmt.Sprint("L'asta per l'azione della societa' ", value.auction.Company.Name, " e' stata vinta da ", name, " per ", value.auction.HighestOffer/100, " $!\n\nClicca <a href=\""+tg.webURL+"/game/market/\">qui</a> per controllare il mercato!"))
						msg.ParseMode = tgbotapi.ModeHTML
						tg.bot.Send(msg)
					} else {
					}
				}
			}

		case <-tg.quit:
			tg.quit <- true
			return
		}
	}
}
