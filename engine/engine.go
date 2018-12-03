package engine

import (
	"github.com/jinzhu/gorm"
	"log"
	"time"
)

type Engine struct {
	db     *gorm.DB
	et     *eventThread
	logger *log.Logger
}

type EngineSession struct {
	timestamp time.Time
	logger    *log.Logger
	et        *eventThread
	tx        *gorm.DB
	toCommit  bool
}

func NewEngine(db *gorm.DB, logger *log.Logger) *Engine {
	e := &Engine{db: db, logger: logger}

	e.et = NewEventThread(e)

	tx := e.openSessionUnsafe()
	defer tx.Close()
	if nextEventValid, nextEventTs := tx.processEvents(); nextEventValid {
		e.et.RegisterEvent(nextEventTs)
	}
	tx.Commit()

	return e
}

func (e *Engine) OpenSession() *EngineSession {
	es := &EngineSession{logger: e.logger, et: e.et}

	es.timestamp = time.Now()
	e.et.RequestToken(es.timestamp)

	es.tx = e.db.Begin()

	return es
}

func (e *Engine) openSessionUnsafe() *EngineSession {
	es := &EngineSession{logger: e.logger}

	es.timestamp = time.Now()
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

func (es *EngineSession) GetTimestamp() time.Time {
	return es.timestamp
}

func (es *EngineSession) ForceEventProcessing() {
	es.et.RegisterEvent(time.Now())
}
