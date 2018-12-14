package engine

import (
	"errors"
	. "github.com/vincenscotti/impero/model"
)

func (es *EngineSession) GetReports(p *Player) (err error, reports []*Report) {
	reports = make([]*Report, 0)
	if err := es.tx.Where("`player_id` = ?", p.ID).Order("`Date` desc", true).Find(&reports).Error; err != nil {
		panic(err)
	}

	return
}

func (es *EngineSession) GetReport(p *Player, id int) (err error, report *Report) {
	report = &Report{}
	if err := es.tx.Where(id).First(report).Error; err != nil {
		panic(err)
	}

	if report.PlayerID != p.ID {
		err = errors.New("Non hai i permessi per vedere questo report!")

		return
	}

	report.Read = true
	if err := es.tx.Save(&report).Error; err != nil {
		panic(err)
	}

	return
}

func (es *EngineSession) DeleteReports(p *Player, ids []int) error {
	notmine := 0

	if err := es.tx.Model(&Report{}).Where("`id` in (?) and `player_id` != ?", ids, p.ID).Count(&notmine).Error; err != nil {
		panic(err)
	}

	if notmine > 0 {
		return errors.New("Non hai i permessi per cancellare questi report!")
	}

	if err := es.tx.Delete(&Report{}, "id in (?)", ids).Error; err != nil {
		panic(err)
	}

	return nil
}
