// Command klima monitors Midea PortaSplit stock across multiple data sources
// and pushes notifications via Pushover when availability is found.
package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"product-monitor/internal/config"
	"product-monitor/internal/metrics"
	"product-monitor/internal/model"
	"product-monitor/internal/monitor"
	"product-monitor/internal/notify"
	"product-monitor/internal/source"
	"product-monitor/internal/store"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to the YAML config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.New(slog.NewTextHandler(os.Stderr, nil)).Error("config error", "err", err)
		os.Exit(2)
	}

	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
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

	mx := metrics.New()
	go serveMetrics(cfg.MetricsAddr, mx, log)

	notifier := notify.NewPushover(httpClient, cfg.PushoverToken, cfg.PushoverUser, cfg.PushoverPriority, cfg.PushoverDevice, cfg.PushoverRetry, cfg.PushoverExpire)

	var flareSolverr *source.FlareSolverr
	if cfg.FlareSolverrURL != "" {
		flareSolverr = source.NewFlareSolverr(cfg.FlareSolverrURL, cfg.FlareSolverrTimeout)
		log.Info("flaresolverr enabled", "url", cfg.FlareSolverrURL)
	}

	var sources []model.Source
	if cfg.BraucheKlimaEnabled {
		sources = append(sources, source.NewBraucheKlima(httpClient, flareSolverr, cfg.BraucheKlimaURL, cfg.BraucheKlimaProducts))
	}
	if cfg.ObiEnabled {
		sources = append(sources, source.NewObi(httpClient, cfg.ObiProductIDs, cfg.HomePLZ))
	}
	if cfg.MediaMarktEnabled {
		sources = append(sources, source.NewMediaMarkt(httpClient, flareSolverr, cfg.MediaMarktURLs))
	}
	if cfg.EuronicsEnabled {
		sources = append(sources, source.NewEuronics(httpClient, flareSolverr, cfg.EuronicsURLs))
	}
	if cfg.GlobusEnabled {
		sources = append(sources, source.NewGlobus(httpClient, flareSolverr, cfg.GlobusURLs))
	}
	if cfg.AmazonEnabled {
		sources = append(sources, source.NewAmazon(httpClient, flareSolverr, cfg.AmazonURLs))
	}
	if cfg.BauhausEnabled {
		sources = append(sources, source.NewBauhaus(httpClient, flareSolverr, cfg.BauhausURLs))
	}
	if cfg.HagebauEnabled {
		sources = append(sources, source.NewHagebau(httpClient, flareSolverr, cfg.HagebauURLs))
	}
	if cfg.HornbachEnabled {
		sources = append(sources, source.NewHornbach(httpClient, flareSolverr, cfg.HornbachURLs))
	}
	if cfg.ToomEnabled {
		sources = append(sources, source.NewToom(httpClient, flareSolverr, cfg.ToomURLs))
	}
	if cfg.BauhausStoreEnabled {
		if flareSolverr == nil {
			log.Warn("bauhaus-store source needs flaresolverr.url, skipping")
		} else {
			sources = append(sources, source.NewBauhausStore(httpClient, flareSolverr, cfg.BauhausStoreProductIDs, cfg.BauhausStoreIDs, cfg.BauhausStoreName))
		}
	}

	if len(sources) == 0 {
		log.Error("no sources enabled, exiting")
		os.Exit(2)
	}

	mon := monitor.New(sources, db, notifier, log, cfg.PriceMax, cfg.LocalPLZPrefixes, mx)
	mon.Run(ctx, cfg.CheckInterval)
}

// serveMetrics runs the Prometheus metrics HTTP server until the process exits.
func serveMetrics(addr string, mx *metrics.Metrics, log *slog.Logger) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", mx.Handler())
	log.Info("metrics server listening", "addr", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Error("metrics server stopped", "err", err)
	}
}
