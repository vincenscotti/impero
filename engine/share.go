package engine

import (
	. "github.com/vincenscotti/impero/model"
)

func (es *EngineSession) GetSharesForPlayer(p *Player) (err error, shares []*SharesPerPlayer) {
	shares = make([]*SharesPerPlayer, 0)

	if err := es.tx.Table("shares").Select("DISTINCT company_id, count(company_id) as shares").Where("`owner_id` = ?", p.ID).Group("`company_id`").Order("`shares` desc").Find(&shares).Error; err != nil {
		panic(err)
	}

	return
}

func (es *EngineSession) CalculateSharesIncome(shares []*SharesPerPlayer) (err error, income int) {
	for _, sh := range shares {
		cmp := &sh.Company

		if err := es.tx.Where(sh.CompanyID).Find(cmp).Error; err != nil {
			panic(err)
		}

		err, _, sh.ValuePerShare = es.GetCompanyFinancials(cmp)

		if err != nil {
			return
		}

		income += sh.Shares * sh.ValuePerShare
	}

	return
}

func (es *EngineSession) GetShareAuctionsWithPlayerParticipation(p *Player) (err error, shareauctions []*ShareAuction) {
	shareauctions = make([]*ShareAuction, 0)

	if err := es.tx.Model(&ShareAuction{}).Preload("Share").Order("`expiration`").Find(&shareauctions).Error; err != nil {
		panic(err)
	}

	for _, sa := range shareauctions {
		if err := es.tx.Where(sa.Share.CompanyID).Find(&sa.Share.Company).Error; err != nil {
			panic(err)
		}

		participations := make([]*ShareAuctionParticipation, 0)
		if err := es.tx.Where("`share_auction_id` = ? and `player_id` = ?", sa.ID, p.ID).Find(&participations).Error; err != nil {
			panic(err)
		}

		sa.Participations = participations
	}

	return
}
