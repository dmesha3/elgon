package elgon

import "net/http"

// Group is a route group with a shared prefix and middleware.
type Group struct {
	app    *App
	prefix string
	mw     []Middleware
}

func (g *Group) Group(prefix string, mw ...Middleware) *Group {
	joined := g.prefix + normalizePath(prefix)
	all := make([]Middleware, 0, len(g.mw)+len(mw))
	all = append(all, g.mw...)
	all = append(all, mw...)
	return &Group{app: g.app, prefix: normalizePath(joined), mw: all}
}

func (g *Group) route(method, path string, h HandlerFunc, mw ...Middleware) {
	full := normalizePath(g.prefix + normalizePath(path))
	all := make([]Middleware, 0, len(g.mw)+len(mw))
	all = append(all, g.mw...)
	all = append(all, mw...)
	g.app.route(method, full, h, all...)
}

func (g *Group) GET(path string, h HandlerFunc, mw ...Middleware) {
	g.route(http.MethodGet, path, h, mw...)
}
func (g *Group) POST(path string, h HandlerFunc, mw ...Middleware) {
	g.route(http.MethodPost, path, h, mw...)
}
func (g *Group) PUT(path string, h HandlerFunc, mw ...Middleware) {
	g.route(http.MethodPut, path, h, mw...)
}
func (g *Group) PATCH(path string, h HandlerFunc, mw ...Middleware) {
	g.route(http.MethodPatch, path, h, mw...)
}
func (g *Group) DELETE(path string, h HandlerFunc, mw ...Middleware) {
	g.route(http.MethodDelete, path, h, mw...)
}
