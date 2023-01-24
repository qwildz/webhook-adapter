package routes

import (
	"errors"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/qwildz/webhook-adapter/db"
	"github.com/qwildz/webhook-adapter/models"
	"gorm.io/gorm"
)

func Webhook(c *fiber.Ctx) error {
	id := c.Params("id")
	var channel models.Channel
	result := db.Instance.First(&channel, "id = ?", id)

	// check error ErrRecordNotFound
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return c.SendStatus(404)
	}

	if c.Get("X-Webhook-Token", "") != channel.Token {
		return c.SendStatus(403)
	}

	res, ctype, err := channel.Run(c.Context(), string(c.Body()))

	if err != nil {
		log.Printf("An error occurred: %v", err)
		return c.SendStatus(500)
	}

	c.Context().SetContentType(ctype)
	return c.SendString(res)
}
