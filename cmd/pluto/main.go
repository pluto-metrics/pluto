package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"net/http/pprof"
	"strings"

	"github.com/justinas/alice"
	"github.com/pluto-metrics/pluto/pkg/config"
	"github.com/pluto-metrics/pluto/pkg/insert"
	"github.com/pluto-metrics/pluto/pkg/listen"
	"github.com/pluto-metrics/pluto/pkg/otelcfg"
	"github.com/pluto-metrics/pluto/pkg/prom"
	"github.com/pluto-metrics/pluto/pkg/trace"
	"github.com/pluto-metrics/pluto/pkg/when"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

var pathKeepOperation = map[string]bool{
	"/api/v1/query": true,
	"/api/v1/write": true,
}

func otelOperation(r *http.Request) string {
	if pathKeepOperation[r.URL.Path] {
		return r.URL.Path
	}

	if strings.HasPrefix(r.URL.Path, "/api/v1/label/") {
		return "/api/v1/label/"
	}

	return "other"
}

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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// open telemetry init
	tm, err := otelcfg.New(ctx, cfg.Otel, cfg.Logging)
	if err != nil {
		log.Fatal(err)
	}
	defer tm.Shutdown(ctx)

	httpManager := listen.NewHTTP(alice.New(trace.Middleware(otelOperation), when.Middleware))
	// receiver
	if cfg.Insert.Enabled {
		trace.Log(ctx).Info("insert enabled", zap.String("addr", cfg.Insert.Listen))
		mux := httpManager.Mux(cfg.Insert.Listen)
		rw := insert.NewPrometheusRemoteWrite(insert.Opts{
			Config: cfg,
		})

		mux.HandleFunc("/api/v1/write", rw.ServeHTTP)
	}

	//debug
	if cfg.Debug.Enabled {
		trace.Log(ctx).Info("debug enabled", zap.String("addr", cfg.Debug.Listen))
		mux := httpManager.Mux(cfg.Debug.Listen)

		if cfg.Debug.Metrics {
			prometheus.MustRegister(
				collectors.NewBuildInfoCollector(),
			)

			mux.Handle("/metrics", promhttp.Handler())
		}

		if cfg.Debug.Pprof {
			mux.HandleFunc("/debug/pprof/", pprof.Index)
			mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
			mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
			mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
			mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
		}

	}

	// prometheus
	if cfg.Prometheus.Enabled {
		trace.Log(ctx).Info("prometheus enabled", zap.String("addr", cfg.Debug.Listen))
		p, err := prom.New(ctx, cfg)
		if err != nil {
			log.Fatal(err)
		}

		mux := httpManager.Mux(cfg.Prometheus.Listen)
		mux.Handle("/", p)
	}

	go func() {
		listenErr := httpManager.Run(ctx)
		log.Fatal(listenErr)
	}()

	trace.Log(ctx).Info("ready")

	<-ctx.Done()
}
