package db

import (
	"github.com/qwildz/webhook-adapter/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var Instance *gorm.DB

func Init() {
	var err error

	Instance, err = gorm.Open(sqlite.Open("webhook.db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	Instance.AutoMigrate(&models.Channel{})
}
