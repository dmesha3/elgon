package elgon

import (
	"net/http"
	"strings"
)

type routeEntry struct {
	h HandlerFunc
}

type node struct {
	static        map[string]*node
	paramChild    *node
	paramName     string
	wildcardChild *node
	wildcardName  string
	route         *routeEntry
}

type routeParam struct {
	key   string
	value string
}

type router struct {
	methodRoots map[string]*node
}

func newRouter() *router {
	return &router{methodRoots: make(map[string]*node)}
}

func newNode() *node {
	return &node{static: make(map[string]*node)}
}

func (r *router) add(method, path string, h HandlerFunc) {
	root, ok := r.methodRoots[method]
	if !ok {
		root = newNode()
		r.methodRoots[method] = root
	}
	curr := root
	segs := parseSegments(path)
	for i, seg := range segs {
		switch seg.kind {
		case segmentStatic:
			nxt, ok := curr.static[seg.value]
			if !ok {
				nxt = newNode()
				curr.static[seg.value] = nxt
			}
			curr = nxt
		case segmentParam:
			if curr.paramChild == nil {
				curr.paramChild = newNode()
				curr.paramName = seg.value
			}
			curr = curr.paramChild
		case segmentWildcard:
			if curr.wildcardChild == nil {
				curr.wildcardChild = newNode()
				curr.wildcardName = seg.value
			}
			curr = curr.wildcardChild
			if i != len(segs)-1 {
				panic("wildcard segment must be last")
			}
		}
	}
	curr.route = &routeEntry{h: h}
}

func (r *router) find(method, path string, params []routeParam) (HandlerFunc, []routeParam, bool) {
	root, ok := r.methodRoots[method]
	if !ok {
		return nil, params[:0], false
	}
	params = params[:0]
	curr := root
	if path == "/" || path == "" {
		if curr.route == nil {
			return nil, params, false
		}
		return curr.route.h, params, true
	}

	parts := splitPath(path)
	for i := 0; i < len(parts); i++ {
		part := parts[i]
		if nxt := curr.static[part]; nxt != nil {
			curr = nxt
			continue
		}
		if curr.paramChild != nil {
			params = append(params, routeParam{key: curr.paramName, value: part})
			curr = curr.paramChild
			continue
		}
		if curr.wildcardChild != nil {
			params = append(params, routeParam{key: curr.wildcardName, value: strings.Join(parts[i:], "/")})
			curr = curr.wildcardChild
			break
		}
		return nil, params, false
	}

	if curr.route == nil {
		return nil, params, false
	}
	return curr.route.h, params, true
}

func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := a.ctxPool.Get().(*Ctx)
	ctx.reset(w, r)
	ctx.app = a

	h, params, ok := a.router.find(r.Method, r.URL.Path, ctx.params)
	ctx.params = params
	if !ok {
		a.writeError(ctx, ErrNotFound("route not found"))
		a.ctxPool.Put(ctx)
		return
	}

	if err := h(ctx); err != nil {
		a.writeError(ctx, err)
	}
	a.ctxPool.Put(ctx)
}
