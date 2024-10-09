package main

import (
	"context"
	"flag"
	"log"

	"github.com/pluto-metrics/pluto/cmd/pluto/config"
	"github.com/pluto-metrics/pluto/pkg/insert"
	"github.com/pluto-metrics/pluto/pkg/listen"
	"go.uber.org/zap"
)

func main() {
	var configFilename string
	var development bool
	flag.StringVar(&configFilename, "config", "config.yaml", "Config filename")
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
			ClickhouseDSN:      cfg.Insert.Target.DSN,
			ClickhouseDatabase: cfg.Insert.Target.Database,
			ClickhouseTable:    cfg.Insert.Target.Table,
			IDFunc:             cfg.Insert.IDFunc,
		})

		mux.Handle("/api/v1/write", rw)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		listenErr := httpManager.Run(ctx)
		log.Fatal(listenErr)
	}()

	<-ctx.Done()
}
