package triad

import (
	"strings"
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
func (rg *Routes) Get(methodType, pattern string) (RouteInfo, bool) {
	idx, ok := rg.index[key(methodType, pattern)]
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

// All returns all registered route info data
func (rg *Routes) All() []RouteInfo {
	routes := make([]RouteInfo, 0, len(rg.routes))
	routes = append(routes, rg.routes...)
	return routes
}

// RouteInfo contains information of registered route.
type RouteInfo struct {
	Method     string
	Pattern    string
	Handler    string
	Middleware []string // middleware in registered order
}

func (r RouteInfo) String() string {
	var sb strings.Builder
	defer sb.Reset()
	sb.WriteString(r.Method)
	sb.WriteString(" ")
	sb.WriteString(r.Pattern)
	sb.WriteString(" ")
	for _, m := range r.Middleware {
		sb.WriteString(m)
		sb.WriteString("-->")
	}
	sb.WriteString(r.Handler)
	return sb.String()
}
