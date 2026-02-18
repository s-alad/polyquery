package api

import "github.com/gofiber/fiber/v3"

func Register(app *fiber.App) {
	app.Get("/", func(c fiber.Ctx) error { return c.SendString("ok") })
}
