package treerouter

import (
	"net/http"
	"path"
	"strings"
)

type Router struct {
	*RouterGroup
	RedirectTrailingSlash  bool
	RedirectFixedPath      bool
	RemoveExtraSlash       bool
	HandleMethodNotAllowed bool
}

func NewRouter() *Router {
	routes := newMethodRoot()
	return &Router{
		RouterGroup: &RouterGroup{
			BasePath: "/",
			Routes:   routes,
		},
		RedirectTrailingSlash:  false,
		RedirectFixedPath:      false,
		RemoveExtraSlash:       false,
		HandleMethodNotAllowed: false,
	}
}

func (router *Router) NewGroup(path string) *RouterGroup {
	return router.Bind(path)
}

func (router *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rPath := r.URL.Path

	if router.RemoveExtraSlash {
		rPath = path.Clean(rPath)
	}

	route := router.Routes.get(r.Method)
	if route != nil {
		if routeValue := route.match(rPath); routeValue != nil {
			// if there is no trailing slash mismatch it means an exact match has been found
			if !routeValue.tsr {
				r = AddParams(r, routeValue.params)
				routeValue.handlerChain.ServeHTTP(w, r)
				return
			}
			if router.RedirectTrailingSlash {
				redirectTrailingSlash(w, r)
				return
			}
		}

		// if a route is not found, try finding case insensitive matches
		if router.RedirectFixedPath {
			if path, ok := route.findCaseInsensitivePath(rPath, router.RedirectFixedPath); ok {
				r.URL.Path = path
				redirectRoute(w, r)
				return
			}
		}
	}

	if router.HandleMethodNotAllowed {
		router.methodNotAllowedHandler().ServeHTTP(w, r)
		return
	}

	http.NotFoundHandler().ServeHTTP(w, r)
}

func (router *Router) methodNotAllowedHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		allowedMethods := make([]string, 0, len(router.Routes))
		p := r.URL.Path
		for _, methodNode := range router.Routes {
			if methodNode.method == r.Method {
				continue
			}

			if result := methodNode.node.match(p); result != nil {
				allowedMethods = append(allowedMethods, methodNode.method)
			}
		}

		if len(allowedMethods) > 0 {
			w.Header().Set("Allow", strings.Join(allowedMethods, ", "))
			http.Error(w, "405 method not allowed", http.StatusMethodNotAllowed)
		} else {
			http.NotFoundHandler().ServeHTTP(w, r)
		}
	})
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
	// set 301 status for Get requests, 308 for non-Get requests
	code := http.StatusMovedPermanently
	if r.Method != http.MethodGet {
		code = http.StatusPermanentRedirect
	}
	http.Redirect(w, r, r.URL.Path, code)
}
