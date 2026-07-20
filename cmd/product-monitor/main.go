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
	for _, p := range cfg.Products {
		s := p.Sources
		priceMax := p.EffectivePriceMax(cfg.PriceMax)
		// add wraps a source so its availabilities carry this product's name and cap.
		add := func(src model.Source) { sources = append(sources, source.WithProduct(src, p.Name, priceMax)) }

		if s.BraucheKlima != nil {
			add(source.NewBraucheKlima(httpClient, flareSolverr, s.BraucheKlima.URL, s.BraucheKlima.Products))
		}
		if s.Obi != nil {
			add(source.NewObi(httpClient, s.Obi.ProductIDs, s.Obi.PostalCodes))
		}
		if s.MediaMarkt != nil {
			add(source.NewMediaMarkt(httpClient, flareSolverr, s.MediaMarkt.URLs))
		}
		if s.Euronics != nil {
			add(source.NewEuronics(httpClient, flareSolverr, s.Euronics.URLs))
		}
		if s.Globus != nil {
			add(source.NewGlobus(httpClient, flareSolverr, s.Globus.URLs))
		}
		if s.Amazon != nil {
			add(source.NewAmazon(httpClient, flareSolverr, s.Amazon.URLs))
		}
		if s.Bauhaus != nil {
			add(source.NewBauhaus(httpClient, flareSolverr, s.Bauhaus.URLs))
		}
		if s.Hagebau != nil {
			add(source.NewHagebau(httpClient, flareSolverr, s.Hagebau.URLs))
		}
		if s.Hornbach != nil {
			add(source.NewHornbach(httpClient, flareSolverr, s.Hornbach.URLs))
		}
		if s.Toom != nil {
			add(source.NewToom(httpClient, flareSolverr, s.Toom.URLs))
		}
		if s.SolarProfi != nil {
			// Not anti-bot protected; fetch directly (no FlareSolverr).
			add(source.NewSolarProfi(httpClient, nil, s.SolarProfi.URLs))
		}
		if s.Galaxus != nil {
			// Akamai + CAPTCHA: only reachable through FlareSolverr.
			add(source.NewGalaxus(httpClient, flareSolverr, s.Galaxus.URLs))
		}
		if s.Solario24 != nil {
			add(source.NewSolario24(httpClient, nil, s.Solario24.URLs))
		}
		if s.EvolarShop != nil {
			add(source.NewEvolarShop(httpClient, nil, s.EvolarShop.URLs))
		}
		if s.Bueromarkt != nil {
			// Behind Imperva/Incapsula: route through FlareSolverr.
			add(source.NewBueromarkt(httpClient, flareSolverr, s.Bueromarkt.URLs))
		}
		if s.Expert != nil {
			add(source.NewExpert(httpClient, s.Expert.URLs, s.Expert.StoreID))
		}
		if s.Prosatech != nil {
			add(source.NewProsatech(httpClient, nil, s.Prosatech.URLs))
		}
		if s.Tado != nil {
			add(source.NewTado(httpClient, nil, s.Tado.URLs))
		}
		if s.SolarHandel24 != nil {
			add(source.NewSolarHandel24(httpClient, nil, s.SolarHandel24.URLs))
		}
		if s.SchwabKlima != nil {
			add(source.NewSchwabKlima(httpClient, nil, s.SchwabKlima.URLs))
		}
		if s.Grz != nil {
			add(source.NewGrz(httpClient, nil, s.Grz.URLs))
		}
		if s.Selfio != nil {
			add(source.NewSelfio(httpClient, nil, s.Selfio.URLs))
		}
		if s.KlimaVertrieb != nil {
			add(source.NewKlimaVertrieb(httpClient, nil, s.KlimaVertrieb.URLs))
		}
		if s.GroupSumi != nil {
			add(source.NewGroupSumi(httpClient, nil, s.GroupSumi.URLs))
		}
		if s.WeinmannSchanz != nil {
			add(source.NewWeinmannSchanz(httpClient, nil, s.WeinmannSchanz.URLs))
		}
		if s.TalentKing != nil {
			add(source.NewTalentKing(httpClient, nil, s.TalentKing.URLs))
		}
		if s.HeizungBilliger != nil {
			// Behind Cloudflare (JA3 wall): route through FlareSolverr.
			add(source.NewHeizungBilliger(httpClient, flareSolverr, s.HeizungBilliger.URLs))
		}
		if s.Tecedo != nil {
			add(source.NewTecedo(httpClient, nil, s.Tecedo.URLs))
		}
		if s.MediaDeal != nil {
			add(source.NewMediaDeal(httpClient, nil, s.MediaDeal.URLs))
		}
		if s.Klimafy != nil {
			add(source.NewKlimafy(httpClient, nil, s.Klimafy.URLs))
		}
		if s.Entratek != nil {
			add(source.NewEntratek(httpClient, nil, s.Entratek.URLs))
		}
		if s.BobsElektro != nil {
			add(source.NewBobsElektro(httpClient, nil, s.BobsElektro.URLs))
		}
		if s.GrSolar != nil {
			add(source.NewGrSolar(httpClient, nil, s.GrSolar.URLs))
		}
		if s.BauhausStore != nil {
			if flareSolverr == nil {
				log.Warn("bauhaus-store source needs flaresolverr.url, skipping", "product", p.Name)
			} else {
				stores := make([]source.BauhausStore, len(s.BauhausStore.Stores))
				for i, st := range s.BauhausStore.Stores {
					stores[i] = source.BauhausStore{ID: st.ID, Name: st.Name}
				}
				add(source.NewBauhausStore(httpClient, flareSolverr, s.BauhausStore.ProductIDs, stores))
			}
		}
		if s.ShopifyCollection != nil {
			add(source.NewShopifyCollection(httpClient, nil, s.ShopifyCollection.URLs, s.ShopifyCollection.StoreName))
		}
	}

	if len(sources) == 0 {
		log.Error("no sources configured, exiting")
		os.Exit(2)
	}

	mon := monitor.New(sources, db, notifier, log, cfg.LocalPLZPrefixes, mx)
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
