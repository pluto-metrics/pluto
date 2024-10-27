package main

import (
	"context"
	"flag"
	"log"
	"net/http/pprof"

	"github.com/pluto-metrics/pluto/pkg/config"
	"github.com/pluto-metrics/pluto/pkg/insert"
	"github.com/pluto-metrics/pluto/pkg/listen"
	"github.com/pluto-metrics/pluto/pkg/prom"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

func main() {
	var configFilename string
	var development bool
	flag.StringVar(&configFilename, "config", "/etc/pluto/config.yaml", "Config filename")
	flag.BoolVar(&development, "dev", false, "Use development config by default")
	flag.Parse()

	cfg, err := config.LoadFromFile(configFilename, development)
	if err != nil {
		log.Fatal(err)
	}

	// logging
	logger := zap.Must(cfg.Logging.Build())
	defer logger.Sync()
	defer zap.RedirectStdLog(logger)()
	defer zap.ReplaceGlobals(logger)()

	httpManager := listen.NewHTTP()
	// receiver
	if cfg.Insert.Enabled {
		mux := httpManager.Mux(cfg.Insert.Listen)
		rw := insert.NewPrometheusRemoteWrite(insert.Opts{
			Config: cfg,
		})

		mux.Handle("/api/v1/write", rw)
	}

	//debug
	if cfg.Debug.Enabled {
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
		go func() {
			promErr := prom.Run(ctx, cfg)
			log.Fatal(promErr)
		}()
	}

	go func() {
		listenErr := httpManager.Run(ctx)
		log.Fatal(listenErr)
	}()

	<-ctx.Done()
}
