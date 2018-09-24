package engine

import (
	"errors"
	"github.com/jinzhu/gorm"
	. "github.com/vincenscotti/impero/model"
	"golang.org/x/crypto/bcrypt"
)

func (es *EngineSession) SignupPlayer(p *Player) (error, *Player) {
	if p.Name != "" && p.Password != "" {
		cnt := 0
		if err := es.tx.Model(p).Where(&Player{Name: p.Name}).Count(&cnt).Error; err != nil {
			panic(err)
		}

		if cnt != 0 {
			return errors.New("Username gia' in uso!"), nil
		} else {
			//p.Budget = opt.PlayerBudget
			//p.ActionPoints = opt.PlayerActionPoints
			p.Budget, p.ActionPoints = 10, 10 // FIXME: this should depend on the game options

			pwdhash, err := bcrypt.GenerateFromPassword([]byte(p.Password), 10)

			if err != nil {
				panic(err)
			}

			p.Password = string(pwdhash)

			if err := es.tx.Create(&p).Error; err != nil {
				panic(err)
			}

			return nil, p
		}
	} else {
		return errors.New("Username e password devono essere non vuoti!"), nil
	}
}

func (es *EngineSession) LoginPlayer(p *Player) (error, *Player) {
	if p.Name != "" && p.Password != "" {
		hashedp := Player{}
		hashedp.Name = p.Name

		if err := es.tx.Where(&hashedp).First(&hashedp).Error; err != nil && err != gorm.ErrRecordNotFound {
			panic(err)
		}

		if bcrypt.CompareHashAndPassword([]byte(hashedp.Password), []byte(p.Password)) == nil {
			p.ID = hashedp.ID

			return nil, p
		} else {
			return errors.New("Login fallito!"), nil
		}
	} else {
		return errors.New("Username e password devono essere non vuoti!"), nil
	}
}
