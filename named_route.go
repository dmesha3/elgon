package elgon

import "net/http"

// NamedRoute allows registering a route and associating it with a name.
type NamedRoute struct {
	app  *App
	name string
}

func (n *NamedRoute) route(method, path string, h HandlerFunc, mw ...Middleware) {
	n.app.named[n.name] = normalizePath(path)
	n.app.route(method, path, h, mw...)
}

func (n *NamedRoute) GET(path string, h HandlerFunc, mw ...Middleware) {
	n.route(http.MethodGet, path, h, mw...)
}
func (n *NamedRoute) POST(path string, h HandlerFunc, mw ...Middleware) {
	n.route(http.MethodPost, path, h, mw...)
}
func (n *NamedRoute) PUT(path string, h HandlerFunc, mw ...Middleware) {
	n.route(http.MethodPut, path, h, mw...)
}
func (n *NamedRoute) PATCH(path string, h HandlerFunc, mw ...Middleware) {
	n.route(http.MethodPatch, path, h, mw...)
}
func (n *NamedRoute) DELETE(path string, h HandlerFunc, mw ...Middleware) {
	n.route(http.MethodDelete, path, h, mw...)
}
