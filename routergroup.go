package treerouter

import (
	"net/http"
)

type RouteGroup struct {
	BasePath    string
	Middlewares []chainable
	routes      routes
}

func NewGroup(basePath string, methods routes, middlewares ...chainable) *RouteGroup {
	return &RouteGroup{
		BasePath:    basePath,
		Middlewares: middlewares,
		routes:      methods,
	}
}

// Bind adds suffix to existing group path and returns a new group
func (group *RouteGroup) Bind(path string) *RouteGroup {
	newGroup := &RouteGroup{
		BasePath:    joinPaths(group.BasePath, path),
		Middlewares: group.Middlewares,
		routes:      group.routes,
	}

	return newGroup
}

func (group *RouteGroup) Use(middlewares ...chainable) {
	group.Middlewares = append(group.Middlewares, middlewares...)
}

// GET is a helper function for creating Get route in treerouter
func (group *RouteGroup) GET(path string, handler http.HandlerFunc) *RouteGroup {
	combinedPath := group.addRoute(path, http.MethodGet, handler)
	return NewGroup(combinedPath, group.routes)
}

// POST is a helper function for creating Post route in treerouter
func (group *RouteGroup) POST(path string, handler http.HandlerFunc) *RouteGroup {
	combinedPath := group.addRoute(path, http.MethodPost, handler)
	return NewGroup(combinedPath, group.routes)
}

// PUT is a helper function for creating Put route in treerouter
func (group *RouteGroup) PUT(path string, handler http.HandlerFunc) *RouteGroup {
	combinedPath := group.addRoute(path, http.MethodPut, handler)
	return NewGroup(combinedPath, group.routes)
}

// PATCH is a helper function for creating Patch route in treerouter
func (group *RouteGroup) PATCH(path string, handler http.HandlerFunc) *RouteGroup {
	combinedPath := group.addRoute(path, http.MethodPatch, handler)
	return NewGroup(combinedPath, group.routes)
}

// DELETE is a helper function for creating Delete route in treerouter
func (group *RouteGroup) DELETE(path string, handler http.HandlerFunc) *RouteGroup {
	combinedPath := group.addRoute(path, http.MethodDelete, handler)
	return NewGroup(combinedPath, group.routes)
}

// addRoute appends route handler to the middlewares and forms a HandlerChain
func (group *RouteGroup) addRoute(relativePath, method string, handler http.HandlerFunc) string {
	combinedPath := joinPaths(group.BasePath, relativePath)
	handlers := append(group.Middlewares, NewChainable(handler))
	hChain := NewHandlerChain(handlers...)

	methodRoot := group.routes.get(method)
	if methodRoot != nil {
		methodRoot.addNode(combinedPath, hChain)
	}

	return combinedPath
}
