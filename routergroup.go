package treerouter

import (
	"net/http"
	"path"
)

type RouterGroup struct {
	BasePath string
	Handlers []chainable

	Routes routes
}

func NewGroup(basePath string, methods routes, handlers ...chainable) *RouterGroup {
	return &RouterGroup{
		BasePath: basePath,
		Handlers: handlers,
		Routes:   methods,
	}
}

// Bind adds suffix to existing group path and returns a new group
func (group *RouterGroup) Bind(path string) *RouterGroup {
	newGroup := &RouterGroup{
		BasePath: joinPaths(group.BasePath, path),
		Handlers: group.Handlers,
		Routes:   group.Routes,
	}

	return newGroup
}

func (group *RouterGroup) Use(middlewares ...chainable) {
	group.Handlers = append(group.Handlers, middlewares...)
}

// GET is a helper function for creating Get route in treerouter
func (group *RouterGroup) GET(path string, handler http.HandlerFunc) *RouterGroup {
	combinedPath := group.addRoute(path, http.MethodGet, handler)
	return NewGroup(combinedPath, group.Routes)
}

// POST is a helper function for creating Post route in treerouter
func (group *RouterGroup) POST(path string, handler http.HandlerFunc) *RouterGroup {
	combinedPath := group.addRoute(path, http.MethodPost, handler)
	return NewGroup(combinedPath, group.Routes)
}

// PUT is a helper function for creating Put route in treerouter
func (group *RouterGroup) PUT(path string, handler http.HandlerFunc) *RouterGroup {
	combinedPath := group.addRoute(path, http.MethodPut, handler)
	return NewGroup(combinedPath, group.Routes)
}

// PATCH is a helper function for creating Patch route in treerouter
func (group *RouterGroup) PATCH(path string, handler http.HandlerFunc) *RouterGroup {
	combinedPath := group.addRoute(path, http.MethodPatch, handler)
	return NewGroup(combinedPath, group.Routes)
}

// DELETE is a helper function for creating Delete route in treerouter
func (group *RouterGroup) DELETE(path string, handler http.HandlerFunc) *RouterGroup {
	combinedPath := group.addRoute(path, http.MethodDelete, handler)
	return NewGroup(combinedPath, group.Routes)
}

func (group *RouterGroup) addRoute(relativePath, method string, handler http.HandlerFunc) string {
	combinedPath := joinPaths(group.BasePath, relativePath)
	combinedHandlers := append(group.Handlers, newChainable(handler))
	methodRoot := group.Routes.get(method)
	if methodRoot != nil {
		methodRoot.addNode(combinedPath, combinedHandlers...)
	}

	return combinedPath
}

func lastChar(s string) byte {
	if s == "" {
		panic("path cannot be empty")
	}

	return s[len(s)-1]
}

// joins absolute path with relative path while preserving end slash
func joinPaths(absolutePath, relativePath string) string {
	if relativePath == "" {
		return absolutePath
	}

	combinedPath := path.Join(absolutePath, relativePath)

	if combinedPath[0] != '/' {
		combinedPath = "/" + combinedPath
	}

	if lastChar(relativePath) == '/' && lastChar(combinedPath) != '/' {
		return combinedPath + "/"
	}
	return combinedPath
}
