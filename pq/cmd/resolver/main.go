package main

import (
	"os"

	"github.com/charmbracelet/log"
	"github.com/gofiber/fiber/v3"
	"github.com/spf13/cobra"
)

var rcmd = &cobra.Command{Use: "resolver", Short: "Polyquery resolver", Run: run}

func main() {
	if err := rcmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) {
	app := fiber.New()
	api := app.Group("/api")
	api.Get("/health", func(c fiber.Ctx) error { return c.SendString("ok") })
	log.Info("resolver started", "addr", ":8745")
	app.Listen(":8745")
}
