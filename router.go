package treerouter

import (
	"net/http"
	"path"
	"strings"
)

const (
	mGet int = iota
	mPost
	mPut
	mPatch
	mDelete
)

type Router struct {
	*RouterGroup
	Methods               requestMethods
	RedirectTrailingSlash bool
	RedirectFixedPath     bool
	RemoveExtraSlash      bool
}

var methodOrder = map[string]int{
	http.MethodGet:    mGet,
	http.MethodPost:   mPost,
	http.MethodPut:    mPut,
	http.MethodPatch:  mPatch,
	http.MethodDelete: mDelete,
}

func NewRouter() *Router {
	methods := newMethodRoot()
	return &Router{
		RouterGroup: &RouterGroup{
			BasePath: "/",
			Methods:  methods,
		},
		Methods:               methods,
		RedirectTrailingSlash: true,
		RedirectFixedPath:     false,
		RemoveExtraSlash:      false,
	}
}

func (router *Router) NewGroup(path string) *RouterGroup {
	return router.Bind(path)
}

func (router *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var handler http.Handler
	rPath := r.URL.Path

	if router.RemoveExtraSlash {
		rPath = path.Clean(rPath)
	}
	methodNode, exists := router.Methods[r.Method]
	if !exists {
		handler = router.methodNotAllowedHandler()
		goto handle
	}

	if routeValue := methodNode.match(rPath); routeValue != nil {
		if !routeValue.tsr {
			r = AddParams(r, routeValue.params)
			handler = routeValue.handler
			goto handle
		}
		if !router.RedirectTrailingSlash {
			handler = http.NotFoundHandler()
			goto handle
		}
		redirectTrailingSlash(w, r)
		return
	}

	if router.RedirectFixedPath {
		if p, ok := methodNode.findCaseInsensitivePath(rPath, router.RedirectFixedPath); ok {
			r.URL.Path = p
			redirectRoute(w, r)
			return
		}
		handler = http.NotFoundHandler()
	}

handle:
	if handler != nil {
		handler.ServeHTTP(w, r)
	}
}

func (router *Router) methodNotAllowedHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		allowedMethods := make([]string, 0, len(methodOrder))
		p := r.URL.Path
		for method, methodNode := range router.Methods {
			if method == r.Method {
				continue
			}

			if result := methodNode.match(p); result != nil {
				allowedMethods = append(allowedMethods, method)
			}
		}
		w.Header().Set("Allow", strings.Join(allowedMethods, ","))
		http.Error(w, "405 method not allowed", http.StatusMethodNotAllowed)
	}
}

func redirectTrailingSlash(w http.ResponseWriter, r *http.Request) {
	rPath := r.URL.Path
	pLength := len(rPath)

	if rPath[pLength-1] == '/' {
		rPath = rPath[:pLength-1]
	} else {
		rPath += "/"
	}

	// path.Clean returns "." when arg is empty string
	if prefix := path.Clean(r.Header.Get("X-Forwarded-Prefix")); prefix != "." {
		rPath = prefix + "/" + rPath
	}

	r.URL.Path = rPath
	redirectRoute(w, r)
}

func redirectRoute(w http.ResponseWriter, r *http.Request) {
	// set 301 status for Get requests, 308 for non-get requests
	code := http.StatusMovedPermanently
	if r.Method != http.MethodGet {
		code = http.StatusPermanentRedirect
	}
	http.Redirect(w, r, r.URL.Path, code)
}
