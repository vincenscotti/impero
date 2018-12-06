package engine

import (
	"github.com/jinzhu/gorm"
	. "github.com/vincenscotti/impero/model"
	"log"
	"time"
)

type Engine struct {
	db          *gorm.DB
	et          *eventThread
	logger      *log.Logger
	notificator Notificator
}

type EngineSession struct {
	e         *Engine
	timestamp time.Time
	tx        *gorm.DB
	opt       Options
	toCommit  bool
}

func NewEngine(db *gorm.DB, logger *log.Logger) *Engine {
	e := &Engine{db: db, logger: logger}

	e.et = NewEventThread(e)

	return e
}

func (e *Engine) Boot() {
	tx := e.openSessionUnsafe()
	defer tx.Close()
	if nextEventValid, nextEventTs := tx.processEvents(); nextEventValid {
		e.et.RegisterEvent(nextEventTs)
	}
	tx.Commit()
}

func (e *Engine) OpenSession() *EngineSession {
	es := &EngineSession{e: e}

	es.timestamp = time.Now()
	e.et.RequestToken(es.timestamp)

	es.tx = e.db.Begin()
	_, es.opt = es.GetOptions()

	return es
}

func (e *Engine) openSessionUnsafe() *EngineSession {
	es := &EngineSession{e: e}

	es.timestamp = time.Now()
	es.tx = e.db.Begin()
	_, es.opt = es.GetOptions()

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

func (es *EngineSession) GetTimestamp() time.Time {
	return es.timestamp
}

func (es *EngineSession) ForceEventProcessing() {
	es.e.et.RegisterEvent(time.Now())
}
