package engine

import (
	"log"
	"time"

	"github.com/jinzhu/gorm"
	. "github.com/vincenscotti/impero/model"
)

type TimeProvider interface {
	Now() time.Time
}

type Engine struct {
	db          *gorm.DB
	et          *eventThread
	logger      *log.Logger
	tp          TimeProvider
	notificator Notificator
	jwtPass     []byte
}

type EngineSession struct {
	e         *Engine
	timestamp time.Time
	tx        *gorm.DB
	opt       Options
	toCommit  bool
}

func NewEngine(db *gorm.DB, logger *log.Logger, tp TimeProvider, jwtPass []byte) *Engine {
	e := &Engine{db: db, logger: logger, tp: tp, jwtPass: jwtPass}

	e.et = NewEventThread(e)

	return e
}

func (e *Engine) Boot() {
	e.db.AutoMigrate(&Options{}, &Node{}, &Player{}, &Message{}, &Report{},
		&ChatMessage{}, &Company{}, &Partnership{}, &Shareholder{}, &Rental{},
		&ShareAuction{}, &ShareAuctionParticipation{},
		&TransferProposal{}, &ShareOffer{}, &Token{})

	opt := &Options{}
	if err := e.db.First(opt).Error; err == gorm.ErrRecordNotFound {
		// insert sane default options
		opt.CompanyActionPoints = 5
		opt.CompanyPureIncomePercentage = 30
		opt.CostPerYield = 1.5
		opt.EndGame = 14
		opt.InitialShares = 20
		opt.BlackoutProbPerDollar = 0.001
		opt.StabilityLevels = 5
		opt.MaxBlackoutDeltaPerDollar = 0.0004
		opt.GameStart = e.tp.Now()
		opt.LastTurnCalculated = e.tp.Now()
		opt.NewCompanyCost = 5
		opt.PlayerActionPoints = 8
		opt.PlayerBudget = 10000
		opt.TurnDuration = 5
		opt.Turn = 1

		e.db.Create(opt)
	}

	tx := e.openSessionUnsafe()
	defer tx.Close()

	if nextEventValid, nextEventTs := tx.processEvents(); nextEventValid {
		e.et.RegisterEvent(nextEventTs)
	}
	tx.Commit()
}

func (e *Engine) OpenSession() *EngineSession {
	es := &EngineSession{e: e}

	es.timestamp = e.tp.Now()
	e.et.RequestToken(es.timestamp)

	es.tx = e.db.Begin()
	_, es.opt = es.GetOptions()

	return es
}

func (e *Engine) openSessionUnsafe() *EngineSession {
	es := &EngineSession{e: e}

	es.timestamp = e.tp.Now()
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
	es.e.et.RegisterEvent(es.e.tp.Now())
}
