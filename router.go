package treerouter

import (
	"net/http"
	"path"
)

type Router struct {
	*RouterGroup
	Methods MethodRoot
}

func NewRouter() *Router {
	methods := NewMethodRoot()
	return &Router{
		RouterGroup: &RouterGroup{
			BasePath: "/",
			Root:     methods,
		},
		Methods: methods,
	}
}

func (r *Router) NewGroup(path string) *RouterGroup {
	return r.Bind(path)
}

func (r *Router) match(path, method string) (*routeValue, bool) {
	methodNode, exists := r.Methods[method]
	if !exists {
		return nil, false
	}
	return methodNode.match(path), true
}

func MethodNotAllowedHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "405 method not allowed", http.StatusMethodNotAllowed)
	}
}

func (router *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var handler http.Handler
	path := path.Clean(r.URL.Path)
	routeValue, ok := router.match(path, r.Method)

	if !ok {
		handler = MethodNotAllowedHandler()
	} else if routeValue == nil {
		handler = http.NotFoundHandler()
	} else {
		handler = routeValue.handlerChain
		r = AddParams(r, routeValue.params)
	}
	handler.ServeHTTP(w, r)
}
