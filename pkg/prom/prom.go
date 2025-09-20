package prom

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/grafana/regexp"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"
	"github.com/prometheus/common/route"
	"github.com/prometheus/common/server"
	promConfig "github.com/prometheus/prometheus/config"
	"github.com/prometheus/prometheus/notifier"
	"github.com/prometheus/prometheus/promql"
	"github.com/prometheus/prometheus/rules"
	"github.com/prometheus/prometheus/scrape"
	"github.com/prometheus/prometheus/util/logging"
	"github.com/prometheus/prometheus/util/notifications"
	api_v1 "github.com/prometheus/prometheus/web/api/v1"
	"github.com/prometheus/prometheus/web/ui"

	"github.com/pluto-metrics/pluto/pkg/config"
	uiStatic "github.com/pluto-metrics/prometheus-ui-static"
	"github.com/prometheus/common/assets"
)

var newUIReactRouterPaths = []string{
	"/config",
	"/flags",
	"/service-discovery",
	"/alertmanager-discovery",
	"/status",
	"/targets",
}

var newUIReactRouterServerPaths = []string{
	"/alerts",
	"/query", // The old /graph redirects to /query on the server side.
	"/rules",
	"/tsdb-status",
}

// Prom ...
type Prom struct {
	config      config.Config
	birth       time.Time
	apiV1       *api_v1.API
	router      *route.Router
	routePrefix string
}

// New init prometheus server
func New(ctx context.Context, config *config.Config) (*Prom, error) {
	routePrefix := strings.TrimRight(config.Prometheus.RoutePrefix, "/")
	if routePrefix == "" {
		routePrefix = "/"
	}
	p := &Prom{
		config:      *config,
		birth:       time.Now(),
		router:      route.New(),
		routePrefix: routePrefix,
	}

	// use precompiled static from github.com/lomik/prometheus-ui-static
	ui.Assets = http.FS(assets.New(uiStatic.EmbedFS))

	promLogger := slog.Default()

	storage := newStorage(config)

	corsOrigin, err := regexp.Compile("^$")
	if err != nil {
		return nil, err
	}

	queryEngine := promql.NewEngine(promql.EngineOpts{
		Logger:        promLogger,
		Timeout:       time.Minute,
		MaxSamples:    50000000,
		LookbackDelta: config.Prometheus.LookbackDelta,
	})

	scrapeManager, err := scrape.NewManager(
		&scrape.Options{},
		promLogger,
		func(s string) (*logging.JSONFileLogger, error) { return nil, nil },
		storage,
		prometheus.DefaultRegisterer,
	)
	if err != nil {
		return nil, err
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
		queryEngine,             // h.queryEngine
		storage,                 // h.storage
		nil,                     // app
		&nopExemplarQueryable{}, // h.exemplarStorage
		func(_ context.Context) api_v1.ScrapePoolsRetriever { return scrapeManager },    // factorySPr
		func(_ context.Context) api_v1.TargetRetriever { return scrapeManager },         // factoryTr
		func(_ context.Context) api_v1.AlertmanagerRetriever { return notifierManager }, // factoryAr
		func() promConfig.Config {
			return promConfig.Config{}
		},
		nil, // o.Flags
		api_v1.GlobalURLOptions{
			ListenAddress: config.Prometheus.Listen,
			Host:          u.Host,
			Scheme:        u.Scheme,
		},
		func(hf http.HandlerFunc) http.HandlerFunc {
			return hf
		}, // h.testReady
		storage,    // h.options.LocalStorage
		"",         // h.options.TSDBDir
		false,      // h.options.EnableAdminAPI
		promLogger, // logger
		func(_ context.Context) api_v1.RulesRetriever { return rulesManager }, // FactoryRr
		0,                           // h.options.RemoteReadSampleLimit
		0,                           // h.options.RemoteReadConcurrencyLimit
		0,                           // h.options.RemoteReadBytesInFrame
		false,                       // h.options.IsAgent
		corsOrigin,                  // h.options.CORSOrigin
		p.runtimeInfo,               // h.runtimeInfo
		&api_v1.PrometheusVersion{}, // h.versionInfo
		func() []notifications.Notification {
			return nil
		}, // h.options.NotificationsGetter
		func() (<-chan notifications.Notification, func(), bool) {
			return nil, nil, false
		}, // h.options.NotificationsSub
		&nopGatherer{}, // o.Gatherer
		nil,            // o.Registerer
		nil,            // nil
		false,          // o.EnableRemoteWriteReceiver
		nil,            // o.AcceptRemoteWriteProtoMsgs
		false,          // o.EnableOTLPWriteReceiver
		false,          // o.ConvertOTLPDelta
		false,          // o.NativeOTLPDeltaIngestion
		false,          // o.CTZeroIngestionEnabled
	)

	if p.routePrefix != "/" {
		// slog.Info("Router prefix", "prefix", p.routePrefix)
		p.router = p.router.WithPrefix(p.routePrefix)
	}

	homePage := "/query"
	p.router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		// slog.Info("Redirect to home page", "from", r.URL.Path, "to", p.withPrefix(homePage))
		http.Redirect(w, r, p.withPrefix(homePage), http.StatusFound)
	})

	// redirect to new ui from old ui
	p.router.Get("/graph", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, p.withPrefix("/query?"+r.URL.RawQuery), http.StatusFound)
	})

	reactAssetsRoot := "/static/mantine-ui"

	// The console library examples at 'console_libraries/prom.lib' still depend on old asset files being served under `classic`.
	p.router.Get("/classic/static/*filepath", func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = joinPrefix("/static", route.Param(r.Context(), "filepath"))
		fs := server.StaticFileServer(ui.Assets)
		fs.ServeHTTP(w, r)
	})

	p.router.Get("/version", p.version)

	serveReactApp := func(w http.ResponseWriter, _ *http.Request) {
		indexPath := reactAssetsRoot + "/index.html"
		f, err := ui.Assets.Open(indexPath)
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
		replacedIdx = bytes.ReplaceAll(replacedIdx, []byte("TITLE_PLACEHOLDER"), []byte(config.Prometheus.PageTitle))
		replacedIdx = bytes.ReplaceAll(replacedIdx, []byte("AGENT_MODE_PLACEHOLDER"), []byte(strconv.FormatBool(false)))
		replacedIdx = bytes.ReplaceAll(replacedIdx, []byte("READY_PLACEHOLDER"), []byte(strconv.FormatBool(true)))
		replacedIdx = bytes.ReplaceAll(replacedIdx, []byte("LOOKBACKDELTA_PLACEHOLDER"), []byte(model.Duration(config.Prometheus.LookbackDelta).String()))
		w.Write(replacedIdx)
	}

	// Serve the React app.
	reactRouterPaths := newUIReactRouterPaths
	reactRouterServerPaths := newUIReactRouterServerPaths

	for _, pp := range reactRouterPaths {
		p.router.Get(pp, serveReactApp)
	}

	for _, pp := range reactRouterServerPaths {
		p.router.Get(pp, serveReactApp)
	}

	// The favicon and manifest are bundled as part of the React app, but we want to serve
	// them on the root.
	for _, pp := range []string{"/favicon.svg", "/favicon.ico", "/manifest.json"} {
		assetPath := reactAssetsRoot + pp
		p.router.Get(pp, func(w http.ResponseWriter, r *http.Request) {
			r.URL.Path = assetPath
			fs := server.StaticFileServer(ui.Assets)
			fs.ServeHTTP(w, r)
		})
	}

	reactStaticAssetsDir := "/assets"

	// Static files required by the React app.
	p.router.Get(reactStaticAssetsDir+"/*filepath", func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = joinPrefix(joinPrefix(reactAssetsRoot, reactStaticAssetsDir), route.Param(r.Context(), "filepath"))
		fs := server.StaticFileServer(ui.Assets)
		fs.ServeHTTP(w, r)
	})

	// if o.UserAssetsPath != "" {
	// 	router.Get("/user/*filepath", route.FileServe(o.UserAssetsPath))
	// }

	constHandler := func(code int, format string, v ...interface{}) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(code)
			if format != "" {
				fmt.Fprintf(w, format, v...)
			}
		}
	}

	p.router.Post("/-/quit", constHandler(http.StatusForbidden, "Lifecycle API is not enabled"))
	p.router.Put("/-/quit", constHandler(http.StatusForbidden, "Lifecycle API is not enabled"))
	p.router.Post("/-/reload", constHandler(http.StatusForbidden, "Lifecycle API is not enabled"))
	p.router.Put("/-/reload", constHandler(http.StatusForbidden, "Lifecycle API is not enabled"))
	p.router.Get("/-/quit", constHandler(http.StatusMethodNotAllowed, "Only POST or PUT requests allowed"))
	p.router.Get("/-/reload", constHandler(http.StatusMethodNotAllowed, "Only POST or PUT requests allowed"))
	p.router.Get("/-/healthy", constHandler(http.StatusOK, "%s is Healthy.\n", "pluto"))
	p.router.Head("/-/healthy", constHandler(http.StatusOK, ""))
	p.router.Get("/-/ready", constHandler(http.StatusOK, "%s is Ready.\n", "pluto"))
	p.router.Head("/-/ready", constHandler(http.StatusOK, ""))

	return p, nil
}

func (p *Prom) runtimeInfo() (api_v1.RuntimeInfo, error) {
	status := api_v1.RuntimeInfo{
		StartTime:      p.birth,
		CWD:            "",
		GoroutineCount: runtime.NumGoroutine(),
		GOMAXPROCS:     runtime.GOMAXPROCS(0),
		GOMEMLIMIT:     debug.SetMemoryLimit(-1),
		GOGC:           os.Getenv("GOGC"),
		GODEBUG:        os.Getenv("GODEBUG"),
	}

	hostname, err := os.Hostname()
	if err != nil {
		return status, fmt.Errorf("error getting hostname: %w", err)
	}
	status.Hostname = hostname
	status.ServerTime = time.Now().UTC()

	return status, nil
}

func (p *Prom) version(w http.ResponseWriter, _ *http.Request) {
	dec := json.NewEncoder(w)
	if err := dec.Encode(&api_v1.PrometheusVersion{}); err != nil {
		http.Error(w, fmt.Sprintf("error encoding JSON: %s", err), http.StatusInternalServerError)
	}
}

// Register ...
func (p *Prom) Register(mux *http.ServeMux) {
	mux.Handle("/", p.router)

	// register api
	apiPath := p.withPrefix("/api")
	// slog.Info("api path", "path", apiPath)

	av1 := route.New()
	p.apiV1.Register(av1)

	mux.Handle(joinPrefix(apiPath, "/v1/"), http.StripPrefix(joinPrefix(apiPath, "/v1"), av1))
}

func (p *Prom) withPrefix(path string) string {
	return joinPrefix(p.routePrefix, path)
}

func joinPrefix(pfx, p string) string {
	pfx = strings.Trim(pfx, "/")
	if pfx == "" {
		return p
	}
	return "/" + pfx + "/" + strings.TrimPrefix(p, "/")
}
