package prom

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"runtime"
	"runtime/debug"
	"strconv"
	"time"

	"github.com/grafana/regexp"
	"github.com/pkg/errors"
	"github.com/pluto-metrics/pluto/pkg/config"
	uiStatic "github.com/pluto-metrics/prometheus-ui-static"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/assets"
	"github.com/prometheus/common/route"
	"github.com/prometheus/common/server"
	prometheusConfig "github.com/prometheus/prometheus/config"
	"github.com/prometheus/prometheus/notifier"
	"github.com/prometheus/prometheus/promql"
	"github.com/prometheus/prometheus/rules"
	"github.com/prometheus/prometheus/scrape"
	api_v1 "github.com/prometheus/prometheus/web/api/v1"
)

type Prom struct {
	apiV1     *api_v1.API
	router    *route.Router
	assets    http.FileSystem
	pageTitle string
}

func New(ctx context.Context, config *config.Config) (*Prom, error) {
	p := &Prom{
		router: route.New(),
		// use precompiled static from github.com/pluto-metrics/prometheus-ui-static
		assets:    http.FS(assets.New(uiStatic.EmbedFS)),
		pageTitle: config.Prometheus.PageTitle,
	}

	promLogger := &logger{}

	storage := newStorage(config)

	corsOrigin, err := regexp.Compile("^$")
	if err != nil {
		return nil, errors.WithStack(err)
	}

	queryEngine := promql.NewEngine(promql.EngineOpts{
		Logger:        promLogger,
		Timeout:       time.Minute,
		MaxSamples:    50000000,
		LookbackDelta: config.Prometheus.LookbackDelta,
	})

	scrapeManager, err := scrape.NewManager(&scrape.Options{}, promLogger, storage, prometheus.DefaultRegisterer)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	rulesManager := rules.NewManager(&rules.ManagerOptions{
		Logger:     promLogger,
		Appendable: storage,
		Queryable:  storage,
	})

	notifierManager := notifier.NewManager(&notifier.Options{}, promLogger)

	u, err := url.Parse(config.Prometheus.ExternalURL)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	p.apiV1 = api_v1.NewAPI(
		queryEngine,             // qe promql.QueryEngine
		storage,                 // q storage.SampleAndChunkQueryable
		nil,                     // ap storage.Appendable
		&nopExemplarQueryable{}, // eq storage.ExemplarQueryable
		func(_ context.Context) api_v1.ScrapePoolsRetriever { return scrapeManager },    // spsr func(context.Context) ScrapePoolsRetriever
		func(_ context.Context) api_v1.TargetRetriever { return scrapeManager },         // tr func(context.Context) TargetRetriever
		func(_ context.Context) api_v1.AlertmanagerRetriever { return notifierManager }, // ar func(context.Context) AlertmanagerRetriever
		func() prometheusConfig.Config { return prometheusConfig.Config{} },             // configFunc func() config.Config
		make(map[string]string), // flagsMap map[string]string
		api_v1.GlobalURLOptions{ // globalURLOptions GlobalURLOptions
			ListenAddress: config.Prometheus.Listen,
			Host:          u.Host,
			Scheme:        u.Scheme,
		},
		p.testReady, // readyFunc func(http.HandlerFunc) http.HandlerFunc
		storage,     // db TSDBAdminStats
		"/var/tmp",  // enableAdmin bool
		false,       // enableAdmin bool
		promLogger,  // logger log.Logger
		func(_ context.Context) api_v1.RulesRetriever { return rulesManager }, // rr func(context.Context) RulesRetriever
		0,                           // remoteReadSampleLimit int
		10,                          // remoteReadConcurrencyLimit int
		0,                           // remoteReadMaxBytesInFrame int
		false,                       // isAgent bool
		corsOrigin,                  // corsOrigin *regexp.Regexp
		p.runtimeInfo,               // runtimeInfo func() (RuntimeInfo, error)
		&api_v1.PrometheusVersion{}, // buildInfo *PrometheusVersion
		&nopGatherer{},              // gatherer prometheus.Gatherer
		nil,                         // registerer prometheus.Registerer
		nil,                         // statsRenderer StatsRenderer
		false,                       // rwEnabled bool
		nil,                         // acceptRemoteWriteProtoMsgs []config.RemoteWriteProtoMsg
		false,                       // otlpEnabled bool
	)

	for _, path := range []string{
		"/config",
		"/flags",
		"/service-discovery",
		"/status",
		"/targets",
		"/starting",
		"/alerts",
		"/graph",
		"/rules",
		"/tsdb-status",
	} {
		p.router.Get(path, p.serveReactApp)
	}

	av1 := route.New()
	p.apiV1.Register(av1)
	p.router.Get("/api/v1/*path", http.StripPrefix("/api/v1", av1).ServeHTTP)
	p.router.Options("/api/v1/*path", http.StripPrefix("/api/v1", av1).ServeHTTP)
	p.router.Post("/api/v1/*path", http.StripPrefix("/api/v1", av1).ServeHTTP)
	p.router.Del("/api/v1/*path", http.StripPrefix("/api/v1", av1).ServeHTTP)

	p.router.Get("/static/*filepath", p.serveReactStatic)
	p.router.Get("/favicon.ico", p.serveReactStatic)
	p.router.Get("/manifest.json", p.serveReactStatic)

	p.router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/graph", http.StatusFound)
	})

	p.router.Get("/-/healthy", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "pluto is Healthy.\n")
	})
	p.router.Head("/-/healthy", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	p.router.Get("/-/ready", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "pluto is Ready.\n")
	})
	p.router.Head("/-/ready", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	return p, nil
}

func (p *Prom) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.router.ServeHTTP(w, r)
}

func (p *Prom) serveReactApp(w http.ResponseWriter, r *http.Request) {
	f, err := p.assets.Open("/static/react/index.html")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error opening React index.html: %v", err)
		return
	}
	defer func() { _ = f.Close() }()
	idx, err := io.ReadAll(f)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error reading React index.html: %v", err)
		return
	}
	replacedIdx := bytes.ReplaceAll(idx, []byte("CONSOLES_LINK_PLACEHOLDER"), []byte(""))
	replacedIdx = bytes.ReplaceAll(replacedIdx, []byte("TITLE_PLACEHOLDER"), []byte(p.pageTitle))
	replacedIdx = bytes.ReplaceAll(replacedIdx, []byte("AGENT_MODE_PLACEHOLDER"), []byte(strconv.FormatBool(false)))
	replacedIdx = bytes.ReplaceAll(replacedIdx, []byte("READY_PLACEHOLDER"), []byte(strconv.FormatBool(true)))
	w.Write(replacedIdx)
}

func (p *Prom) serveReactStatic(w http.ResponseWriter, r *http.Request) {
	r.URL.Path = path.Join("/static/react", r.URL.Path[1:])
	fs := server.StaticFileServer(p.assets)
	fs.ServeHTTP(w, r)
}

func (p *Prom) testReady(f http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f(w, r)
	}
}

func (p *Prom) runtimeInfo() (api_v1.RuntimeInfo, error) {
	status := api_v1.RuntimeInfo{
		GoroutineCount: runtime.NumGoroutine(),
		GOMAXPROCS:     runtime.GOMAXPROCS(0),
		GOMEMLIMIT:     debug.SetMemoryLimit(-1),
		GOGC:           os.Getenv("GOGC"),
		GODEBUG:        os.Getenv("GODEBUG"),
	}

	return status, nil
}
