package treerouter

import (
	"fmt"
	"net/http"
)

type HandlerChain struct {
	Handlers []Chainable
	writer   http.ResponseWriter
	request  *http.Request
	// the index of the current to-run handler in the chain
	index int
}

func NewHandlerChain(chainables ...Chainable) HandlerChain {
	return HandlerChain{Handlers: chainables, index: -1}
}

type Chainable func(*HandlerChain)

func NewChainable(h http.HandlerFunc) Chainable {
	return func(hc *HandlerChain) {
		h(hc.writer, hc.request)
	}
}

func (c *HandlerChain) Next() {
	c.index++
	if c.index >= len(c.Handlers) {
		fmt.Println("not enough handlers")
		return
	}
	c.Handlers[c.index](c)
}

// run handlers in the given handler chain from its index
func (c HandlerChain) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c.writer = w
	c.request = r
	c.Next()
}
