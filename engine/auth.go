package engine

import (
	"errors"

	"github.com/golang-jwt/jwt"
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
			_, opt := es.GetOptions()
			p.Budget = opt.PlayerBudget
			p.ActionPoints = opt.PlayerActionPoints

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

func (es *EngineSession) LoginPlayer(p *Player) (error, *jwt.Token, string) {
	claims := &TokenClaims{
		StandardClaims: jwt.StandardClaims{
			Issuer: "impero",
		},
	}

	if p.Name != "" && p.Password != "" {
		hashedp := Player{}
		hashedp.Name = p.Name

		if err := es.tx.Where(&hashedp).First(&hashedp).Error; err != nil && err != gorm.ErrRecordNotFound {
			panic(err)
		}

		if bcrypt.CompareHashAndPassword([]byte(hashedp.Password), []byte(p.Password)) == nil {
			p.ID = hashedp.ID

			token := Token{PlayerID: p.ID}
			if err := es.tx.Create(&token).Error; err != nil {
				panic(err)
			}
			claims.TokenID = token.ID

			jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
			tokenString, err := jwtToken.SignedString([]byte(es.e.jwtPass))
			if err != nil {
				panic(err)
			}
			return nil, jwtToken, tokenString
		} else {
			return errors.New("Login fallito!"), nil, ""
		}
	} else {
		return errors.New("Username e password devono essere non vuoti!"), nil, ""
	}
}

func (es *EngineSession) ValidateTokenString(tokenString string) (*Player, *Token) {
	token, err := jwt.ParseWithClaims(tokenString, &TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(es.e.jwtPass), nil
	})

	if err != nil {
		return nil, nil
	}

	if claims, ok := token.Claims.(*TokenClaims); ok && token.Valid {
		tokenID := claims.TokenID

		token := &Token{}
		token.ID = tokenID

		if err := es.tx.Where(token).Find(token).Error; err != nil && err != gorm.ErrRecordNotFound {
			panic(err)
		}

		if token.PlayerID != 0 {
			p := &Player{}
			p.ID = token.PlayerID
			return p, token
		}
	}

	return nil, nil
}

func (es *EngineSession) DeleteToken(token *Token) {
	es.tx.Delete(token)
}
