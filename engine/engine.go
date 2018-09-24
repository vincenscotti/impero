package engine

import (
	"github.com/jinzhu/gorm"
	"time"
)

type Engine struct {
	db *gorm.DB
	et *eventThread
}

type EngineSession struct {
	timestamp time.Time
	tx        *gorm.DB
	toCommit  bool
}

func NewEngine(db *gorm.DB) *Engine {
	e := &Engine{db: db}

	e.et = NewEventThread(e)

	e.processEvents()

	return e
}

func (e *Engine) OpenSession() *EngineSession {
	es := &EngineSession{}

	es.timestamp = time.Now()
	e.et.RequestToken(es.timestamp)

	es.tx = e.db.Begin()

	return es
}

func (es *EngineSession) Commit() {
	es.toCommit = true
}

func (es *EngineSession) Rollback() {
	es.toCommit = false
}

func (es *EngineSession) Close() {
	if es.toCommit {
		es.tx.Commit()
	} else {
		es.tx.Rollback()
	}
}

func (e *Engine) processEvents() (bool, time.Time) {
	// query db
	// for each pending event
	//   handle it

	return true, time.Now() //nextOne.timestamp
}
