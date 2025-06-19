package treerouter

import (
	"context"
	"net/http"
)

type RouteParams map[string]string

type contextKey struct {
	name string
}

var paramKey = &contextKey{
	name: "params",
}

func AddParams(r *http.Request, params map[string]string) *http.Request {
	if len(params) == 0 {
		return r
	}
	ctx := r.Context()
	ctx = context.WithValue(ctx, paramKey, params)
	return r.WithContext(ctx)
}

func GetParam(r *http.Request, key string) string {
	params, ok := r.Context().Value(paramKey).(map[string]string)
	if !ok {
		panic("Invalid context value")
	}
	return params[key]
}
