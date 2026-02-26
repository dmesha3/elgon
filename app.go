package elgon

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/meshackkazimoto/elgon/db"
	"github.com/meshackkazimoto/elgon/orm"
)

// ErrorResponse is the default structured error payload.
type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

// ErrorBody describes API error details.
type ErrorBody struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Details   any    `json:"details,omitempty"`
	RequestID string `json:"request_id,omitempty"`
}

// App is the main elgon application.
type App struct {
	cfg        Config
	router     *router
	globalMW   []Middleware
	validator  Validator
	ctxPool    sync.Pool
	server     *http.Server
	named      map[string]string
	routes     []RouteInfo
	plugins    map[string]Plugin
	dataMu     sync.RWMutex
	sqlDB      db.Adapter
	ormDialect string
	ormClient  *orm.Client
}

// RouteInfo describes a registered route method and path.
type RouteInfo struct {
	Method string
	Path   string
}

func New(cfg Config) *App {
	cfg = cfg.withDefaults()
	a := &App{
		cfg:     cfg,
		router:  newRouter(),
		named:   make(map[string]string),
		plugins: make(map[string]Plugin),
	}
	a.ctxPool.New = func() any {
		return &Ctx{values: make(map[string]any, 4), params: make([]routeParam, 0, 4)}
	}
	a.server = &http.Server{
		Addr:         cfg.Addr,
		Handler:      a,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}
	if !cfg.DisableHealthz {
		a.registerDefaultHealthRoutes()
	}
	if cfg.EnableMetricsStub {
		a.GET("/metrics", func(c *Ctx) error {
			return c.Text(http.StatusOK, "# metrics stub\n")
		})
	}
	return a
}

func (a *App) registerDefaultHealthRoutes() {
	health := func(c *Ctx) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	}
	a.GET("/health", health)
	a.GET("/ready", health)
	a.GET("/live", health)
}

func (a *App) SetValidator(v Validator) {
	a.validator = v
}

func (a *App) Use(mw ...Middleware) {
	a.globalMW = append(a.globalMW, mw...)
}

func (a *App) Group(prefix string, mw ...Middleware) *Group {
	return &Group{app: a, prefix: normalizePath(prefix), mw: mw}
}

func (a *App) Named(name string) *NamedRoute {
	return &NamedRoute{app: a, name: name}
}

func (a *App) route(method, path string, h HandlerFunc, mw ...Middleware) {
	full := normalizePath(path)
	allMW := make([]Middleware, 0, len(a.globalMW)+len(mw))
	allMW = append(allMW, a.globalMW...)
	allMW = append(allMW, mw...)
	a.router.add(method, full, chain(h, allMW...))
	a.routes = append(a.routes, RouteInfo{Method: method, Path: full})
}

func (a *App) GET(path string, h HandlerFunc, mw ...Middleware) {
	a.route(http.MethodGet, path, h, mw...)
}
func (a *App) POST(path string, h HandlerFunc, mw ...Middleware) {
	a.route(http.MethodPost, path, h, mw...)
}
func (a *App) PUT(path string, h HandlerFunc, mw ...Middleware) {
	a.route(http.MethodPut, path, h, mw...)
}
func (a *App) PATCH(path string, h HandlerFunc, mw ...Middleware) {
	a.route(http.MethodPatch, path, h, mw...)
}
func (a *App) DELETE(path string, h HandlerFunc, mw ...Middleware) {
	a.route(http.MethodDelete, path, h, mw...)
}

func (a *App) Run() error {
	errCh := make(chan error, 1)
	go func() {
		err := a.server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	select {
	case err, ok := <-errCh:
		if ok {
			return err
		}
		return nil
	case <-sigCh:
		ctx, cancel := context.WithTimeout(context.Background(), a.cfg.ShutdownTimeout)
		defer cancel()
		return a.server.Shutdown(ctx)
	}
}

func (a *App) writeError(c *Ctx, err error) {
	if err == nil {
		return
	}
	status := http.StatusInternalServerError
	code := CodeInternal
	message := "internal server error"
	var details any
	if he, ok := asHTTPError(err); ok {
		status = he.Status
		code = he.Code
		message = he.Message
		details = he.Details
	}
	requestID, _ := c.Get("request_id")
	_ = c.JSON(status, ErrorResponse{Error: ErrorBody{
		Code:      code,
		Message:   message,
		Details:   details,
		RequestID: anyToString(requestID),
	}})
}

func anyToString(v any) string {
	s, _ := v.(string)
	return s
}

func normalizePath(path string) string {
	if path == "" || path == "/" {
		return "/"
	}
	if path[0] != '/' {
		path = "/" + path
	}
	if len(path) > 1 && path[len(path)-1] == '/' {
		path = path[:len(path)-1]
	}
	return path
}

func chain(h HandlerFunc, mw ...Middleware) HandlerFunc {
	for i := len(mw) - 1; i >= 0; i-- {
		h = mw[i](h)
	}
	return h
}

// Routes returns a copy of the registered route table.
func (a *App) Routes() []RouteInfo {
	out := make([]RouteInfo, len(a.routes))
	copy(out, a.routes)
	return out
}

// RegisterPlugins initializes and registers one or more plugins.
func (a *App) RegisterPlugins(plugins ...Plugin) error {
	for _, p := range plugins {
		if p == nil {
			continue
		}
		name := p.Name()
		if name == "" {
			return ErrInternal("plugin name must not be empty")
		}
		if _, exists := a.plugins[name]; exists {
			return ErrConflict("plugin already registered: " + name)
		}
		if err := p.Init(a); err != nil {
			return err
		}
		a.plugins[name] = p
	}
	return nil
}

// Plugins returns the list of registered plugin names.
func (a *App) Plugins() []string {
	out := make([]string, 0, len(a.plugins))
	for name := range a.plugins {
		out = append(out, name)
	}
	return out
}
