package treerouter

import (
	"net/http"
	"path"
)

type RouterGroup struct {
	BasePath string
	Handlers []Chainable

	// the root of method tree (usually a reference to method tree at treerouter module)
	Root MethodRoot
}

func NewGroup(basePath string, root MethodRoot, handlers ...Chainable) *RouterGroup {
	return &RouterGroup{
		BasePath: basePath,
		Handlers: handlers,
		Root:     root,
	}
}

// Bind adds suffix to existing group path and returns a new group
func (group *RouterGroup) Bind(path string) *RouterGroup {
	newGroup := &RouterGroup{
		BasePath: joinPaths(group.BasePath, path),
		Handlers: group.Handlers,
		Root:     group.Root,
	}

	return newGroup
}

func (group *RouterGroup) addRoute(relativePath, method string, handler http.HandlerFunc) string {
	// if len(relativePath) == 0 || relativePath[0] != '/' {
	// 	panic(fmt.Sprintf("treerouter: routing pattern must begin with '/' in '%s'", relativePath))
	// }
	// get the matching root node for the given http method
	methodRoot, exists := group.Root[method]
	combinedPath := joinPaths(group.BasePath, relativePath)
	combinedHandlers := append(group.Handlers, NewChainable(handler))
	if exists {
		methodRoot.addNode(combinedPath, combinedHandlers...)
	}

	return combinedPath
}

func (group *RouterGroup) Use(middlewares ...Chainable) {
	group.Handlers = append(group.Handlers, middlewares...)
}

// GET is a helper function for creating Get route in treerouter
func (group *RouterGroup) GET(path string, handler http.HandlerFunc) *RouterGroup {
	combinedPath := group.addRoute(path, http.MethodGet, handler)
	return NewGroup(combinedPath, group.Root)
}

// POST is a helper function for creating Post route in treerouter
func (group *RouterGroup) POST(path string, handler http.HandlerFunc) *RouterGroup {
	combinedPath := group.addRoute(path, http.MethodPost, handler)
	return NewGroup(combinedPath, group.Root)
}

// PATCH is a helper function for creating Patch route in treerouter
func (group *RouterGroup) PATCH(path string, handler http.HandlerFunc) *RouterGroup {
	combinedPath := group.addRoute(path, http.MethodPatch, handler)
	return NewGroup(combinedPath, group.Root)
}

// DELETE is a helper function for creating Delete route in treerouter
func (group *RouterGroup) DELETE(path string, handler http.HandlerFunc) *RouterGroup {
	combinedPath := group.addRoute(path, http.MethodDelete, handler)
	return NewGroup(combinedPath, group.Root)
}

// OPTIONS is a helper function for creating Options route in treerouter
func (group *RouterGroup) OPTIONS(path string, handler http.HandlerFunc) *RouterGroup {
	combinedPath := group.addRoute(path, http.MethodOptions, handler)
	return NewGroup(combinedPath, group.Root)
}

func joinPaths(absolutePath, relativePath string) string {
	if relativePath == "" {
		return absolutePath
	}

	combinedPath := path.Join(absolutePath, relativePath)

	if combinedPath[0] != '/' {
		combinedPath = "/" + combinedPath
	}

	return combinedPath
}
