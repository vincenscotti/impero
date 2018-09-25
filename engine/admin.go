package engine

import (
	"github.com/jinzhu/gorm"
	. "github.com/vincenscotti/impero/model"
)

func (es *EngineSession) GetOptions() (err error, opt Options) {
	if err := es.tx.First(&opt).Error; err != nil && err != gorm.ErrRecordNotFound {
		panic(err)
	}

	return
}

func (es *EngineSession) SaveOptions(opt Options) error {
	if err := es.tx.Save(&opt).Error; err != nil && err != gorm.ErrRecordNotFound {
		panic(err)
	}

	return nil
}
