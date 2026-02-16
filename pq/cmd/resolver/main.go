package main

import (
	"context"
	"encoding/json"
	"os"
	"os/signal"
	"time"

	"github.com/charmbracelet/log"
	"github.com/gofiber/fiber/v3"
	"github.com/salad/polyquery/pq/pkg/api"
	"github.com/salad/polyquery/pq/pkg/gamma"
	"github.com/salad/polyquery/pq/pkg/pipeline"
	"github.com/salad/polyquery/pq/pkg/resolve"
	"github.com/salad/polyquery/pq/pkg/store"
	"github.com/salad/polyquery/pq/resolvers"
	"github.com/spf13/cobra"
)

type Config struct {
	Addr         string  `json:"addr"`
	DBPath       string  `json:"db_path"`
	Confidence   float64 `json:"confidence"`
	VolumeMax    float64 `json:"volume_max"`
	VolumeMin    float64 `json:"volume_min"`
	PollInterval string  `json:"poll_interval"`
}

var defaultCfg = Config{
	Addr:         ":8745",
	DBPath:       "resolver.db",
	Confidence:   0.95,
	VolumeMax:    10000,
	PollInterval: "5m",
}

var cfgPath string

var rootCmd = &cobra.Command{
	Use:   "resolver",
	Short: "Polyquery resolution pipeline + API",
	Run:   run,
}

func init() {
	rootCmd.Flags().StringVar(&cfgPath, "config", "", "path to config.json")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) {
	cfg := defaultCfg
	if cfgPath != "" {
		f, err := os.Open(cfgPath)
		if err != nil {
			log.Fatal("config", "err", err)
		}
		defer f.Close()
		json.NewDecoder(f).Decode(&cfg)
	}

	queries, db, err := store.Open(cfg.DBPath)
	if err != nil {
		log.Fatal("db", "err", err)
	}
	defer db.Close()

	pollInterval, _ := time.ParseDuration(cfg.PollInterval)
	if pollInterval == 0 {
		pollInterval = 5 * time.Minute
	}

	pipe := pipeline.New(gamma.NewClient(), queries,
		resolve.NewRegistry(resolvers.NewNBA()),
		pipeline.Config{
			VolumeMax:  cfg.VolumeMax,
			VolumeMin:  cfg.VolumeMin,
			PollEvery:  pollInterval,
			Confidence: cfg.Confidence,
		})

	app := fiber.New()
	app.Get("/api/health", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})
	api.New(queries).Register(app)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	go pipe.Run(ctx)
	go func() {
		<-ctx.Done()
		app.Shutdown()
	}()

	log.Info("resolver started", "addr", cfg.Addr, "db", cfg.DBPath)
	if err := app.Listen(cfg.Addr, fiber.ListenConfig{DisableStartupMessage: true}); err != nil {
		log.Fatal("server", "err", err)
	}
}
