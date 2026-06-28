// Command klima monitors Midea PortaSplit stock across multiple data sources
// and pushes notifications via Pushover when availability is found.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"portasplit-monitor/internal/config"
	"portasplit-monitor/internal/model"
	"portasplit-monitor/internal/monitor"
	"portasplit-monitor/internal/notify"
	"portasplit-monitor/internal/source"
	"portasplit-monitor/internal/store"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.New(slog.NewTextHandler(os.Stderr, nil)).Error("config error", "err", err)
		os.Exit(2)
	}

	level := slog.LevelInfo
	if cfg.Debug {
		level = slog.LevelDebug
	}
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	slog.SetDefault(log)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	db, err := store.Open(ctx, cfg.DBPath)
	if err != nil {
		log.Error("failed to open database", "path", cfg.DBPath, "err", err)
		os.Exit(1)
	}
	defer db.Close()

	httpClient := &http.Client{Timeout: cfg.HTTPTimeout}

	notifier := notify.NewPushover(httpClient, cfg.PushoverToken, cfg.PushoverUser, cfg.PushoverPriority, cfg.PushoverDevice)

	var flareSolverr *source.FlareSolverr
	if cfg.FlareSolverrURL != "" {
		flareSolverr = source.NewFlareSolverr(cfg.FlareSolverrURL, cfg.FlareSolverrTimeout)
		log.Info("flaresolverr enabled", "url", cfg.FlareSolverrURL)
	}

	var sources []model.Source
	if cfg.BraucheKlimaEnabled {
		sources = append(sources, source.NewBraucheKlima(httpClient, flareSolverr, cfg.BraucheKlimaURL, cfg.BraucheKlimaProduct))
	}
	if cfg.ObiEnabled {
		sources = append(sources, source.NewObi(httpClient, cfg.ObiProductID, cfg.HomePLZ))
	}
	if cfg.MediaMarktEnabled {
		sources = append(sources, source.NewMediaMarkt(httpClient, flareSolverr, cfg.MediaMarktURL))
	}
	if cfg.EuronicsEnabled {
		sources = append(sources, source.NewEuronics(httpClient, flareSolverr, cfg.EuronicsURL))
	}

	if len(sources) == 0 {
		log.Error("no sources enabled, exiting")
		os.Exit(2)
	}

	mon := monitor.New(sources, db, notifier, log, cfg.PriceMax, cfg.LocalPLZPrefixes)
	mon.Run(ctx, cfg.CheckInterval)
}
