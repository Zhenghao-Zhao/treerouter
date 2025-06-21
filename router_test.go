package treerouter

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testCase struct {
	request  *http.Request
	expected string
}

type header struct {
	key   string
	value string
}

func test(t *testing.T, testCases []testCase) {
	for _, c := range testCases {
		if c.request == nil {
			t.Fatal("failed to create request")
		}

		res, err := http.DefaultClient.Do(c.request)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v",
				res.StatusCode, http.StatusOK)
		}
		reqBody, err := io.ReadAll(res.Body)
		if err != nil {
			t.Errorf("failed to read response body:%v", err)
		}
		assert.Equal(t, c.expected, string(reqBody))
	}
}

func createRequest(method, url string, headers ...header) *http.Request {
	request, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil
	}
	for _, h := range headers {
		request.Header.Set(h.key, h.value)
	}
	return request
}

func TestRouteFormats(t *testing.T) {
	router := NewRouter()

	router.GET("users", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("users"))
	})

	router.GET("profiles/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("profiles"))
	})

	router.GET("/posts/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("posts"))
	})

	server := httptest.NewServer(router)
	defer server.Close()

	testCases := []testCase{
		{request: createRequest(http.MethodGet, server.URL+"/users/"), expected: "users"},
		{request: createRequest(http.MethodGet, server.URL+"/profiles"), expected: "profiles"},
		{request: createRequest(http.MethodGet, server.URL+"/posts"), expected: "posts"},
		{request: createRequest(http.MethodGet, server.URL+"/posts?id=1"), expected: "posts"},
	}

	test(t, testCases)
}

func TestMixedRoutes(t *testing.T) {
	router := NewRouter()
	groupUser := router.NewGroup("user")

	groupUser.GET("/*", func(w http.ResponseWriter, r *http.Request) {
		param := GetParam(r, "*")
		w.Write([]byte(param))
	})

	groupUser.GET("/:name/id", func(w http.ResponseWriter, r *http.Request) {
		paramVal := GetParam(r, "name")
		w.Write([]byte(paramVal))
	})

	groupUser.GET("/name", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("john doe"))
	})

	s := httptest.NewServer(router)
	defer s.Close()

	testCases := []testCase{
		{request: createRequest(http.MethodGet, s.URL+"/user/orange/blue"), expected: "orange/blue"},
		{request: createRequest(http.MethodGet, s.URL+"/user/red"), expected: "red"},
		{request: createRequest(http.MethodGet, s.URL+"/user/johndoe/id"), expected: "johndoe"},
		{request: createRequest(http.MethodGet, s.URL+"/user/name"), expected: "john doe"},
	}

	test(t, testCases)
}

func TestDynamicRoutes(t *testing.T) {
	router := NewRouter()
	groupUser := router.NewGroup("user")

	groupUser.GET("/:id", func(w http.ResponseWriter, r *http.Request) {
		param := GetParam(r, "id")
		w.Write([]byte(param))
	})

	groupUser.POST("/:id/profile", func(w http.ResponseWriter, r *http.Request) {
		param := GetParam(r, "id")
		w.Write([]byte(param))
	})

	groupUser.PATCH("/:id/:name", func(w http.ResponseWriter, r *http.Request) {
		first := GetParam(r, "id")
		second := GetParam(r, "name")
		fmt.Fprintf(w, "%s %s", first, second)
	})

	groupUser.DELETE("/johndoe/profile", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("johndoe"))
	})

	s := httptest.NewServer(router)
	defer s.Close()

	testCases := []testCase{
		{request: createRequest(http.MethodGet, s.URL+"/user/orange"), expected: "orange"},
		{request: createRequest(http.MethodPost, s.URL+"/user/apple/profile"), expected: "apple"},
		{request: createRequest(http.MethodPost, s.URL+"/user/123/profile"), expected: "123"},
		{request: createRequest(http.MethodPatch, s.URL+"/user/123/profil"), expected: "123 profil"},
		{request: createRequest(http.MethodDelete, s.URL+"/user/johndoe/profile"), expected: "johndoe"},
	}

	test(t, testCases)
}

func TestMiddleware(t *testing.T) {
	router := NewRouter()
	groupUser := router.NewGroup("user")

	AuthKey := "auth"
	AuthValue := "123abc"

	// create a simple auth middleware
	groupUser.Use(func(hc *HandlerChain) {
		authValue := hc.request.Header.Get(AuthKey)
		if authValue == "" {
			http.Error(hc.writer, "failed to find auth header", http.StatusUnauthorized)
			return
		}

		if authValue != AuthValue {
			http.Error(hc.writer, "mismatching auth value", http.StatusUnauthorized)
			return
		}
		hc.Next()
	})

	groupUser.GET("/:id", func(w http.ResponseWriter, r *http.Request) {
		param := GetParam(r, "id")
		w.Write([]byte(param))
	})

	server := httptest.NewServer(router)
	defer server.Close()

	testCases := []testCase{
		{request: createRequest(http.MethodGet, server.URL+"/user/orange", header{key: AuthKey, value: AuthValue}), expected: "orange"},
	}

	test(t, testCases)
}

func TestStaticRoutes(t *testing.T) {
	router := NewRouter()

	groupCompany := router.NewGroup("company")

	// depth 1: /company/{name}
	groupCompany.GET("/apple", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("apple"))
	})
	groupCompany.POST("/microsoft", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("microsoft")) })
	groupCompany.DELETE("/google", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("google")) })

	// // depth 2: /company/staff/{name}
	groupCompanyStaff := groupCompany.Bind("/staff")
	groupCompanyStaff.GET("/alice", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("alice")) })
	groupCompanyStaff.POST("/benjamin", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("benjamin")) })
	groupCompanyStaff.DELETE("/chuck", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("chuck")) })
	//
	// // depth 3: /company/staff/job/{name}
	groupCompanyStaffTitle := groupCompanyStaff.Bind("/job")
	groupCompanyStaffTitle.GET("/engineer", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("engineer")) })
	groupCompanyStaffTitle.POST("/accountant", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("accountant")) })
	groupCompanyStaffTitle.DELETE("/hr", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("hr")) })

	server := httptest.NewServer(router)
	defer server.Close()

	results := []testCase{
		{request: createRequest(http.MethodGet, server.URL+"/company/apple"), expected: "apple"},
		{request: createRequest(http.MethodPost, server.URL+"/company/microsoft"), expected: "microsoft"},
		{request: createRequest(http.MethodGet, server.URL+"/company/staff/alice"), expected: "alice"},
		{request: createRequest(http.MethodPost, server.URL+"/company/staff/benjamin"), expected: "benjamin"},
		{request: createRequest(http.MethodGet, server.URL+"/company/staff/job/engineer"), expected: "engineer"},
		{request: createRequest(http.MethodPost, server.URL+"/company/staff/job/accountant"), expected: "accountant"},
	}

	test(t, results)
}
