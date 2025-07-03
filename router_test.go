package treerouter

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testCase struct {
	request         *http.Request
	expectedMessage string
	expectedCode    int
}

type header struct {
	key   string
	value string
}

// performs a simple http test using httptest package, returns the response recorder.
// does not auto-redirect
func performQuickTest(handler http.Handler, method, path string, headers ...header) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, nil)

	for _, header := range headers {
		request.Header.Set(header.key, header.value)
	}

	writer := httptest.NewRecorder()
	handler.ServeHTTP(writer, request)
	return writer
}

func performRedirectTest(handler http.Handler, method, path string, headers ...header) (*httptest.ResponseRecorder, *httptest.ResponseRecorder) {
	// perform initial request
	request := httptest.NewRequest(method, path, nil)
	for _, header := range headers {
		request.Header.Set(header.key, header.value)
	}
	firstW := httptest.NewRecorder()
	handler.ServeHTTP(firstW, request)

	// perform redirected request
	newPath := firstW.Header().Get("Location")
	if newPath == "" {
		return firstW, nil
	}
	request = httptest.NewRequest(method, newPath, nil)
	secondW := httptest.NewRecorder()
	handler.ServeHTTP(secondW, request)
	return firstW, secondW
}

// performs a http test using http client, returns the response.
// redirects to target location
func performHTTPTest(t *testing.T, server *httptest.Server, method, path string, headers ...header) (*http.Response, string) {
	client := http.DefaultClient
	req, err := http.NewRequest(method, server.URL+path, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, header := range headers {
		req.Header.Set(header.key, header.value)
	}
	res, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	return res, string(resBody)
}

func TestTrailingSlash(t *testing.T) {
	router := NewRouter()
	router.RedirectTrailingSlash = true
	router.RemoveExtraSlash = false
	router.RedirectFixedPath = false

	router.GET("/posts/", func(w http.ResponseWriter, r *http.Request) {})
	first, second := performRedirectTest(router, http.MethodGet, "/posts")
	assert.Equal(t, first.Code, http.StatusMovedPermanently)
	assert.Equal(t, first.Header().Get("Location"), "/posts/")
	assert.Equal(t, second.Code, http.StatusOK)
}

func TestFixedPath(t *testing.T) {
	router := NewRouter()
	router.RedirectFixedPath = true
	router.RedirectTrailingSlash = true

	router.GET("/username", func(w http.ResponseWriter, r *http.Request) {})
	router.GET("/proFile/nAME", func(w http.ResponseWriter, r *http.Request) {})

	first, second := performRedirectTest(router, http.MethodGet, "/userName")
	assert.Equal(t, first.Code, http.StatusMovedPermanently)
	assert.Equal(t, first.Header().Get("Location"), "/username")
	assert.Equal(t, second.Code, http.StatusOK)

	first, second = performRedirectTest(router, http.MethodGet, "/profile/name/")
	assert.Equal(t, first.Code, http.StatusMovedPermanently)
	assert.Equal(t, first.Header().Get("Location"), "/proFile/nAME")
	assert.Equal(t, second.Code, http.StatusOK)
}

func TestMethodNotAllowed(t *testing.T) {
	router := NewRouter()
	router.HandleMethodNotAllowed = true

	router.GET("/username", func(w http.ResponseWriter, r *http.Request) {})
	router.PUT("/username", func(w http.ResponseWriter, r *http.Request) {})
	router.DELETE("/username", func(w http.ResponseWriter, r *http.Request) {})

	res := performQuickTest(router, http.MethodPost, "/username")
	assert.Equal(t, http.StatusMethodNotAllowed, res.Code)
	assert.Equal(t, "GET, PUT, DELETE", res.Header().Get("Allow"))
}

func TestRouteExtraSlash(t *testing.T) {
	router := NewRouter()
	router.RemoveExtraSlash = true
	router.RedirectTrailingSlash = false

	router.GET("/", func(w http.ResponseWriter, r *http.Request) {})
	router.PUT("/users", func(w http.ResponseWriter, r *http.Request) {})

	res := performQuickTest(router, http.MethodGet, "//")
	assert.Equal(t, http.StatusOK, res.Code)

	res = performQuickTest(router, http.MethodPut, "/users/")
	assert.Equal(t, http.StatusOK, res.Code)
}

func TestMixedRoutes(t *testing.T) {
	router := NewRouter()
	groupUser := router.NewGroup("user")

	groupUser.GET("/*", func(w http.ResponseWriter, r *http.Request) {
		param := GetParam(r, "*")
		w.Write([]byte(param))
	})
	// the previous catch-all route should be overriden by the following route
	groupUser.GET("/:name/*", func(w http.ResponseWriter, r *http.Request) {
		param := GetParam(r, "*")
		w.Write([]byte(param))
	})
	// static segment after /:name has higher priority therefore should override catch-all segment
	groupUser.GET("/:name/id", func(w http.ResponseWriter, r *http.Request) {
		paramVal := GetParam(r, "name")
		w.Write([]byte(paramVal))
	})
	groupUser.GET("/:id/name", func(w http.ResponseWriter, r *http.Request) {
		paramVal := GetParam(r, "id")
		w.Write([]byte(paramVal))
	})
	groupUser.GET("/name", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("john doe"))
	})

	server := httptest.NewServer(router)
	defer server.Close()

	res, body := performHTTPTest(t, server, http.MethodGet, "/user/red")
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, "red", body)

	res, body = performHTTPTest(t, server, http.MethodGet, "/user/johndoe/id/name")
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, "id/name", body)

	res, body = performHTTPTest(t, server, http.MethodGet, "/user/johndoe/name")
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, "johndoe", body)

	res, body = performHTTPTest(t, server, http.MethodGet, "/user/name")
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, "john doe", body)

	res, body = performHTTPTest(t, server, http.MethodGet, "/user/name/foo")
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, "foo", body)
}

func TestMiddleware(t *testing.T) {
	router := NewRouter()

	authKey := "auth"
	authValue := "123abc"

	// create a simple auth middleware
	router.Use(func(hc *HandlerChain) {
		v := hc.request.Header.Get(authKey)
		if v == "" {
			http.Error(hc.writer, "failed to find auth header", http.StatusUnauthorized)
			return
		}

		if v != authValue {
			http.Error(hc.writer, "mismatching auth value", http.StatusUnauthorized)
			return
		}
		hc.Next()
	})

	router.GET("/:id", func(w http.ResponseWriter, r *http.Request) {
		param := GetParam(r, "id")
		w.Write([]byte(param))
	})

	server := httptest.NewServer(router)
	defer server.Close()

	res, body := performHTTPTest(t, server, http.MethodGet, "/orange", header{key: authKey, value: authValue})
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, "orange", body)
}

func TestStaticRoutes(t *testing.T) {
	router := NewRouter()

	groupUser := router.NewGroup("user")
	groupArticle := router.NewGroup("article")

	groupUser.GET("/profile", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("user profile")) })
	groupUser.POST("/profession", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("user profession")) })

	groupArticle.GET("/profile", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("article profile")) })
	groupArticle.POST("/profession", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("article profession")) })

	groupUserId := groupUser.Bind("id")
	groupArticleId := groupArticle.Bind("id")

	groupUserId.GET("/profile", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("user id profile")) })
	groupUserId.POST("/profession", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("user id profession")) })

	groupArticleId.GET("/profile", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("article id profile")) })
	groupArticleId.POST("/profession", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("article id profession")) })

	server := httptest.NewServer(router)
	defer server.Close()

	res, body := performHTTPTest(t, server, http.MethodGet, "/user/profile")
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, "user profile", body)

	res, body = performHTTPTest(t, server, http.MethodPost, "/user/profession")
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, "user profession", body)

	res, body = performHTTPTest(t, server, http.MethodGet, "/article/profile")
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, "article profile", body)

	res, body = performHTTPTest(t, server, http.MethodPost, "/article/profession")
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, "article profession", body)

	res, body = performHTTPTest(t, server, http.MethodGet, "/user/id/profile")
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, "user id profile", body)

	res, body = performHTTPTest(t, server, http.MethodPost, "/user/id/profession")
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, "user id profession", body)

	res, body = performHTTPTest(t, server, http.MethodGet, "/article/id/profile")
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, "article id profile", body)

	res, body = performHTTPTest(t, server, http.MethodPost, "/article/id/profession")
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, "article id profession", body)
}
