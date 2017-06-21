package main

import (
	"errors"
	. "impero/model"
	"impero/templates"
	"math"
	"math/rand"
	"net/http"
	"time"
)

func randYield() int {
	r := rand.Intn(100) + 1

	switch {
	case r <= 20:
		return 1
	case r > 20 && r <= 85:
		return 2
	case r > 85 && r <= 96:
		return 4
	case r > 96 && r <= 99:
		return 6
	default:
		return 8
	}
}

func Admin(w http.ResponseWriter, r *http.Request) {
	opt := GetOptions(r)
	session := GetSession(r)

	p := &AdminData{Options: opt}
	if msg := session.Flashes("message_"); len(msg) > 0 {
		p.Message = msg[0].(string)
	}

	session.Save(r, w)

	w.WriteHeader(200)
	w.Write([]byte(templates.AdminPage(p)))
}

type formTime time.Time

func (this *formTime) UnmarshalText(text []byte) error {
	t, err := time.Parse("2006-01-02 15:04:05-07:00", string(text))
	*this = formTime(t)

	return err
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
		goto out
	}

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

out:
	session.Save(r, w)

	url, err := router.Get("admin").URL()
	if err != nil {
		panic(err)
	}

	http.Redirect(w, r, url.Path, http.StatusFound)
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

		yields := map[int]float64{
			1:  0.2,
			3:  0.6,
			6:  0.13,
			10: 0.05,
			20: 0.02,
		}

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

		for yield, prob := range yields {
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

	session.Save(r, w)

	url, err := router.Get("admin").URL()
	if err != nil {
		panic(err)
	}

	http.Redirect(w, r, url.Path, http.StatusFound)
}

func BroadcastMessage(w http.ResponseWriter, r *http.Request) {
	tx := GetTx(r)
	session := GetSession(r)

	msg := &Message{}
	players := make([]*Player, 0)

	if err := validateAdmin(r); err != nil {
		session.AddFlash(err.Error(), "message_")
	} else {
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
	}

out:
	session.Save(r, w)

	url, err := router.Get("admin").URL()
	if err != nil {
		panic(err)
	}

	http.Redirect(w, r, url.Path, http.StatusFound)
}
