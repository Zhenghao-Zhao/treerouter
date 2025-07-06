# Treerouter
Treerouter is a high performance router component that fits seamlessly with go's built-in http library. It uses [radix tree](https://en.wikipedia.org/wiki/Radix_tree) to store route handlers that allows for fast and efficient matching.

The purpose of treerouter is to mainly serve as a template router for people to add customized features for their projects.

## Features
**High Performance:** Treerouter has been benchmarked using [go-web-framework-benchmark](https://github.com/smallnest/go-web-framework-benchmark) benchmarking tool. Its performances are on par with some of the most popular go frameworks in the market.
- Processing time test: The first test case is to mock 0 ms, 10 ms, 100 ms, 500 ms processing time in handlers.
![image](https://github.com/user-attachments/assets/f8a9cd7f-061d-4618-964e-3e462dd24177)

- Concurrency test (allocations): In 30 ms processing time, the test result for 100, 1000, 5000 clients is:
![image](https://github.com/user-attachments/assets/3c7205c0-5de6-4f30-aada-f2a37076824f)

- CPU bound test:
![image](https://github.com/user-attachments/assets/fc3d3bd9-e6f0-4263-8ee8-4263d7800a7f)

**Auto-redirects malformed paths:** Treerouter can detect and correct extra/missing slashes, such as // or ../ in the paths when the option is enabled.

**Auto-redirects mismatched routes due to case sensitivity:** When the option is enabled, treerouter will match routes without taking into account case sensitivity, and redirects to the correct path if a handler is found.

**Method not allowed support:** When the option is enabled, treerouter will return a 405 response to the request with a header that lists all avaible http methods for that path.

**Supports dynamic params and catch-alls:** Treerouter supports dynamic params (e.g. /user/:name) and catch-all segments (e.g. /user/*) in the path.

**Middleware support and route groups** Treerouter allows you to add middlewares to **route groups**. A route group is essentially a path prefix that allows all following paths to share the same handlers, including middlewares. Simply create a route group using the **Bind** method, and add a middleware using the **Use** function.

**Zero dependencies**

## Usage
Adding a middleware to the router and attaching a handler:
```go
func AuthMiddleware(hc *HandlerChain) {
  v := hc.request.Header.Get(authKey)
  if v != authValue {
    http.Error(hc.writer, "mismatching auth value", http.StatusUnauthorized)
    return  
  }
  hc.Next()
}
func HelloHandler(w http.ResponseWriter, r *http.Request) {
  param := GetParam(r, "name")
  w.Write([]byte("hello, " + param))
}
func main() {
  // r is implicitly a route group
  r := treerouter.New()
  r.Use(AuthMiddleware)
  r.GET("/hello/:name", HelloHandler)
}
```
## Pattern matching
```
Our pattern matching follows a priority order: exact pattern -> params -> catch-all.
If we have the following patterns:
/user/name
/user/:user
/user/*
and the request has path /user/johndoe
it will match /user/:user, but if the path is /user/name, it will match /user/name
```
