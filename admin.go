package main

import (
	"errors"
	. "github.com/vincenscotti/impero/model"
	"github.com/vincenscotti/impero/templates"
	"math"
	"math/rand"
	"net/http"
	"time"
)

func Admin(w http.ResponseWriter, r *http.Request) {
	opt := GetOptions(r)
	session := GetSession(r)

	p := &AdminData{Options: opt}
	if msg := session.Flashes("message_"); len(msg) > 0 {
		p.Message = msg[0].(string)
	}

	RenderHTML(w, r, templates.AdminPage(p))
}

type PasswordForm struct {
	Password string
}

func validateAdmin(r *http.Request) (err error) {
	p := PasswordForm{}

	if err := binder.Bind(&p, r); err != nil {
		panic(err)
	}

	if p.Password != AdminPass {
		err = errors.New("Password errata!")
	}

	return
}

func UpdateOptions(w http.ResponseWriter, r *http.Request) {
	tx := GetTx(r)
	session := GetSession(r)

	newopt := &Options{}

	otheropts := struct {
		LastCheckpoint     formTime
		LastTurnCalculated formTime
		Action             string
	}{}

	if err := validateAdmin(r); err != nil {
		session.AddFlash(err.Error(), "message_")
	} else {
		if err := binder.Bind(newopt, r); err != nil {
			panic(err)
		}

		if err := binder.Bind(&otheropts, r); err != nil {
			panic(err)
		}

		newopt.ID = 1
		newopt.LastCheckpoint = time.Time(otheropts.LastCheckpoint)
		newopt.LastTurnCalculated = time.Time(otheropts.LastTurnCalculated)

		if err := tx.Save(newopt).Error; err != nil {
			panic(err)
		}

		session.AddFlash("Opzioni aggiornate", "message_")
	}

	Redirect(w, r, "admin")
}

var NodeYields = []struct {
	Yield       int
	Prob        float64
	UpgradeCost int
}{
	{1, 0.22, 1},
	{3, 0.5, 2},
	{6, 0.15, 5},
	{12, 0.08, 13},
	{25, 0.04, 30},
	{50, 0.01, 0},
}

func GenerateMap(w http.ResponseWriter, r *http.Request) {
	tx := GetTx(r)
	session := GetSession(r)

	params := struct {
		X0       int
		Y0       int
		X1       int
		Y1       int
		Generate bool
	}{}

	if err := binder.Bind(&params, r); err != nil {
		panic(err)
	}

	if err := validateAdmin(r); err != nil {
		session.AddFlash(err.Error(), "message_")
	} else {
		xsize := params.X1 - params.X0
		ysize := params.Y1 - params.Y0
		sortednodes := make([]*Node, 0, xsize*ysize)

		if params.Generate {
			for i := params.X0; i < params.X1; i++ {
				for j := params.Y0; j < params.Y1; j++ {
					sortednodes = append(sortednodes, &Node{X: i, Y: j})
				}
			}
		} else {
			if err := tx.Where("`x` > ? and `x` < ? and `y` > ? and `y` < ?",
				params.X0, params.X1, params.Y0, params.Y1).Find(&sortednodes).Error; err != nil {
				panic(err)
			}
		}

		perm := rand.Perm(len(sortednodes))

		shufflednodes := make([]*Node, len(sortednodes))

		for i, p := range perm {
			shufflednodes[i] = sortednodes[p]
		}

		totalnodes := float64(len(shufflednodes))
		remainingnodes := totalnodes
		maxyield := 0

		for _, y := range NodeYields {
			yield := y.Yield
			prob := y.Prob

			if yield > maxyield {
				maxyield = yield
			}

			nodesperyield := int(math.Floor(totalnodes * prob))

			logger.Println("inserisco", nodesperyield, "nodi con rendimento", yield)

			for nodesperyield > 0 {
				shufflednodes[0].Yield = yield
				if err := tx.Save(shufflednodes[0]).Error; err != nil {
					panic(err)
				}

				shufflednodes = shufflednodes[1:]
				nodesperyield -= 1
				remainingnodes -= 1
			}
		}

		logger.Println("inserisco restanti", remainingnodes, "nodi con rendimento", maxyield)

		for remainingnodes > 0 {
			shufflednodes[0].Yield = maxyield
			if err := tx.Save(shufflednodes[0]).Error; err != nil {
				panic(err)
			}

			shufflednodes = shufflednodes[1:]
			remainingnodes -= 1
		}

		logger.Println("al termine della procedura avanzano", len(shufflednodes), "nodi")

		session.AddFlash("Rendimenti aggiornati", "message_")
	}

	Redirect(w, r, "admin")
}

func BroadcastMessage(w http.ResponseWriter, r *http.Request) {
	tx := GetTx(r)
	session := GetSession(r)

	msg := &Message{}
	players := make([]*Player, 0)

	if err := validateAdmin(r); err != nil {
		session.AddFlash(err.Error(), "message_")

		goto out
	}

	if err := binder.Bind(msg, r); err != nil {
		panic(err)
	}

	if msg.Content == "" {
		session.AddFlash("Non puoi inviare un messaggio vuoto!", "message_")

		goto out
	}

	msg.Date = time.Now()
	msg.Read = false

	if err := tx.Find(&players).Error; err != nil {
		panic(err)
	}

	for _, p := range players {
		msg.ID = 0
		msg.ToID = p.ID

		if err := tx.Create(msg).Error; err != nil {
			panic(err)
		}
	}

	session.AddFlash("Messaggio inviato!", "message_")

out:
	Redirect(w, r, "admin")
}
