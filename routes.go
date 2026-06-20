package triad

import (
	"iter"
)

// Routes contains the registered routes
type Routes struct {
	index  map[string]int
	routes []RouteInfo
}

func NewRoutes() *Routes {
	return &Routes{
		index: make(map[string]int),
	}
}

func key(mType, pattern string) string {
	return mType + " " + pattern
}

// Get retrieve single route info if present in registry
func (rg *Routes) Get(method, pattern string) (RouteInfo, bool) {
	idx, ok := rg.index[key(method, pattern)]
	var route RouteInfo
	if ok {
		route = rg.routes[idx]
	}
	return route, ok
}

// add the route
func (rg *Routes) add(r RouteInfo) {
	rg.index[key(r.Method, r.Pattern)] = len(rg.routes)
	rg.routes = append(rg.routes, r)
}

func (rg *Routes) Iter() iter.Seq[RouteInfo] {
	return func(yield func(RouteInfo) bool) {
		for _, r := range rg.routes {
			if !yield(r) {
				return
			}
		}
	}
}

// RouteInfo contains information of registered route.
type RouteInfo struct {
	Method     string
	Pattern    string
	Handler    string
	Middleware []string // middleware in registered order
}

func (r RouteInfo) String() string {
	return r.Method + " " + r.Pattern
}
