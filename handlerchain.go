package treerouter

import (
	"net/http"
)

type HandlerChain struct {
	Handlers []chainable
	writer   http.ResponseWriter
	request  *http.Request
	// the index of the current to-run handler in the chain
	index int
}

type chainable func(*HandlerChain)

func (c *HandlerChain) Next() {
	c.index++
	if c.index >= len(c.Handlers) {
		panic("not enough handlers")
	}
	c.Handlers[c.index](c)
}

// run handlers in the given handler chain from its index
func (c HandlerChain) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c.writer = w
	c.request = r
	c.Next()
}

func NewHandlerChain(chainables ...chainable) HandlerChain {
	return HandlerChain{Handlers: chainables, index: -1}
}

func NewChainable(h http.HandlerFunc) chainable {
	return func(hc *HandlerChain) {
		h(hc.writer, hc.request)
	}
}
