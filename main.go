package main

import (
	"github.com/gofiber/fiber/v2"
	"github.com/qwildz/webhook-adapter/db"
	"github.com/qwildz/webhook-adapter/routes"
)

func main() {
	db.Init()

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		if c.Get("X-Webhook-Token", "") == "" {
			return c.SendStatus(403)
		}

		return c.Next()
	})
	app.Post("/:id", routes.Webhook)
	app.Listen(":3000")
}
