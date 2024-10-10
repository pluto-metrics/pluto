package prom

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/grafana/regexp"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	promConfig "github.com/prometheus/prometheus/config"
	"github.com/prometheus/prometheus/notifier"
	"github.com/prometheus/prometheus/promql"
	"github.com/prometheus/prometheus/rules"
	"github.com/prometheus/prometheus/scrape"
	"github.com/prometheus/prometheus/web"
	"github.com/prometheus/prometheus/web/ui"
	"go.uber.org/zap"

	"github.com/pluto-metrics/pluto/pkg/config"
	uiStatic "github.com/pluto-metrics/prometheus-ui-static"
	"github.com/prometheus/common/assets"
)

func Run(ctx context.Context, config *config.Config) error {
	// use precompiled static from github.com/lomik/prometheus-ui-static
	ui.Assets = http.FS(assets.New(uiStatic.EmbedFS))

	promLogger := &logger{
		z: zap.L().With(zap.String("module", "prometheus")),
	}

	storage := newStorage(config)

	corsOrigin, err := regexp.Compile("^$")
	if err != nil {
		return err
	}

	queryEngine := promql.NewEngine(promql.EngineOpts{
		Logger:        promLogger,
		Timeout:       time.Minute,
		MaxSamples:    50000000,
		LookbackDelta: config.Prometheus.LookbackDelta,
	})

	scrapeManager, err := scrape.NewManager(&scrape.Options{}, promLogger, storage, prometheus.DefaultRegisterer)
	if err != nil {
		return err
	}

	rulesManager := rules.NewManager(&rules.ManagerOptions{
		Logger:     promLogger,
		Appendable: storage,
		Queryable:  storage,
	})

	notifierManager := notifier.NewManager(&notifier.Options{}, promLogger)

	u, err := url.Parse(config.Prometheus.ExternalURL)
	if err != nil {
		return errors.WithStack(err)
	}

	promHandler := web.New(promLogger, &web.Options{
		ListenAddress:              config.Prometheus.Listen,
		MaxConnections:             500,
		Storage:                    storage,
		ExemplarStorage:            &nopExemplarQueryable{},
		ExternalURL:                u,
		RoutePrefix:                "/",
		QueryEngine:                queryEngine,
		ScrapeManager:              scrapeManager,
		RuleManager:                rulesManager,
		Flags:                      make(map[string]string),
		LocalStorage:               storage,
		Gatherer:                   &nopGatherer{},
		Notifier:                   notifierManager,
		CORSOrigin:                 corsOrigin,
		PageTitle:                  config.Prometheus.PageTitle,
		LookbackDelta:              config.Prometheus.LookbackDelta,
		RemoteReadConcurrencyLimit: config.Prometheus.RemoteReadConcurrencyLimit,
	})

	promHandler.ApplyConfig(&promConfig.Config{})
	promHandler.SetReady(true)

	return promHandler.Run(ctx, nil, "")
}
