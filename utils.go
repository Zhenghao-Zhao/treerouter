package treerouter

import (
	"context"
	"net/http"
	"path"
)

type RouteParams map[string]string

type contextKey struct {
	name string
}

var paramKey = &contextKey{
	name: "params",
}

func GetParam(r *http.Request, key string) string {
	params, ok := r.Context().Value(paramKey).(map[string]string)
	if !ok {
		panic("Invalid context value")
	}
	return params[key]
}

func addParams(r *http.Request, params map[string]string) *http.Request {
	if len(params) == 0 {
		return r
	}
	ctx := r.Context()
	ctx = context.WithValue(ctx, paramKey, params)
	return r.WithContext(ctx)
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
