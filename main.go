package main

import (
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	. "impero/model"
	"impero/templates"
	"math/rand"
	"strconv"
)

var db *gorm.DB

func authFunc(user, pass string, c echo.Context) (error, bool) {
	p := Player{}

	p.Name = user
	err := db.Where(&p).First(&p)

	if err.Error != nil {
		return err.Error, false
	}

	if p.Password == pass {
		msgs := 0
		db.Model(&Message{}).Where("read = ? and to_id = ?", false,
			p.ID).Count(&msgs)

		header := &HeaderData{CurrentPlayer: &p, NewMessages: msgs}
		c.Set("header", header)
		return nil, true
	}

	return nil, false
}

func signUp(c echo.Context) error {
	p := Player{}
	msg := ""

	if p.Name != "" || p.Password != "" {
		p.Budget = 50
		p.ActionPoints = 3

		cnt := 0
		db.Model(&Player{}).Where(&Player{Name: p.Name}).Count(&cnt)

		if cnt != 0 {
			msg = "Username gia' in uso!"
		} else {
			db.Create(&p)
			if db.Error != nil {
				return db.Error
			}

			msg = "Registrazione effettuata!"
		}
	}

	c.HTML(200, templates.SignupPage(msg))

	return nil
}

func logout(c echo.Context) error {
	c.HTML(401, templates.LogoutPage())

	return nil
}

func generateMap(c echo.Context) error {
	x0, err := strconv.Atoi(c.Param("x0"))
	if err != nil {
		return err
	}

	y0, err := strconv.Atoi(c.Param("y0"))
	if err != nil {
		return err
	}

	x1, err := strconv.Atoi(c.Param("x1"))
	if err != nil {
		return err
	}

	y1, err := strconv.Atoi(c.Param("y1"))
	if err != nil {
		return err
	}

	n := Node{}

	for i := x0; i < x1; i++ {
		for j := y0; j < y1; j++ {
			n.X = i
			n.Y = j
			n.ID = 0

			db.Where(&n).First(&n)
			if n.ID == 0 {
				n.Yield = rand.Int() % 10
				n.ConstructionCost = n.Yield * 10

				db.Create(&n)
			}
		}
	}

	c.String(200, "ok")

	return nil
}

func gameHome(c echo.Context) error {
	header := c.Get("header").(*HeaderData)

	page := GameHomeData{HeaderData: header}

	c.HTML(200, templates.GameHomePage(&page))

	return nil
}

func messages(c echo.Context) error {
	header := c.Get("header").(*HeaderData)

	msgs := make([]*Message, 0)
	db.Where("read = ? and to_id = ?", false, header.CurrentPlayer.ID).Find(&msgs)
	page := MessagesData{HeaderData: header, Messages: msgs}

	c.HTML(200, templates.MessagesPage(&page))

	return nil
}

func message(c echo.Context) error {
	header := c.Get("header").(*HeaderData)

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return err
	}

	msg := &Message{}
	db.Where(id).First(msg)

	page := MessageData{HeaderData: header, Message: msg}

	c.HTML(200, templates.MessagePage(&page))

	return nil
}

func main() {
	var err error

	db, err = gorm.Open("sqlite3", "impero.db")
	if err != nil {
		panic(err)
	}

	defer db.Close()

	db.AutoMigrate(&Options{}, &Node{}, &Player{}, &Message{}, &Company{},
		&Share{}, &Rental{}, &ShareAuction{}, &RentAuction{})

	e := echo.New()

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.GET("/", signUp)
	e.POST("/", signUp)
	e.GET("/game/logout/", logout)

	game := e.Group("/game")
	game.Use(middleware.BasicAuth(authFunc))
	game.GET("/mapgen/:x0/:y0/:x1/:y1", generateMap)
	game.GET("/", gameHome)
	game.GET("/messages/", messages)
	game.GET("/messages/:id", message)

	e.Logger.Fatal(e.Start(":8080"))
}
