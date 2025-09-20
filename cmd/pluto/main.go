package main

import (
	"context"
	"flag"
	"log"
	"log/slog"
	"net/http/pprof"
	"os"

	"github.com/pluto-metrics/pluto/pkg/config"
	"github.com/pluto-metrics/pluto/pkg/insert"
	"github.com/pluto-metrics/pluto/pkg/lg"
	"github.com/pluto-metrics/pluto/pkg/listen"
	"github.com/pluto-metrics/pluto/pkg/prom"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	var configFilename string
	var development bool
	flag.StringVar(&configFilename, "config", "/etc/pluto/config.yaml", "Config filename")
	flag.BoolVar(&development, "dev", false, "Use development config by default")
	flag.Parse()

	var logLevel = new(slog.LevelVar) // Info by default
	slog.SetDefault(
		slog.New(
			lg.NewHandler(
				slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}),
			),
		),
	)

	cfg, err := config.LoadFromFile(configFilename)
	if err != nil {
		log.Fatal(err)
	}

	// set log level from config
	logLevel.Set(cfg.Logging.Level)

	httpManager := listen.NewHTTP()
	// receiver
	if cfg.Insert.Enabled {
		slog.Info("insert enabled", slog.String("listen", cfg.Insert.Listen))
		mux := httpManager.Mux(cfg.Insert.Listen)
		rw := insert.NewPrometheusRemoteWrite(insert.Opts{
			Config: cfg,
		})

		mux.Handle("/api/v1/write", rw)
	}

	//debug
	if cfg.Debug.Enabled {
		slog.Info("debug enabled", slog.String("listen", cfg.Debug.Listen))
		mux := httpManager.Mux(cfg.Debug.Listen)

		if cfg.Debug.Metrics {
			prometheus.MustRegister(
				collectors.NewBuildInfoCollector(),
			)

			mux.Handle("/metrics", promhttp.HandlerFor(
				prometheus.DefaultGatherer, promhttp.HandlerOpts{
					Registry: prometheus.DefaultRegisterer,
				}))
		}

		if cfg.Debug.Pprof {
			mux.HandleFunc("/debug/pprof/", pprof.Index)
			mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
			mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
			mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
			mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
		}

	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// prometheus
	if cfg.Prometheus.Enabled {
		slog.Info("prometheus enabled", slog.String("listen", cfg.Prometheus.Listen))
		mux := httpManager.Mux(cfg.Prometheus.Listen)

		p, err := prom.New(ctx, cfg)
		if err != nil {
			log.Fatal(err)
		}

		p.Register(mux)
	}

	go func() {
		listenErr := httpManager.Run(ctx)
		log.Fatal(listenErr)
	}()

	<-ctx.Done()
}
