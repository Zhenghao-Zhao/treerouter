package treerouter

import (
	"net/http"
)

type node struct {
	path     string
	handlers []Chainable
	children map[byte]*node

	// each node has at most one param child
	paramChild *node
	isParam    bool
	paramNames []string
}

type MethodRoot map[string]*node

func NewMethodRoot() MethodRoot {
	initPath := "/"
	return MethodRoot{
		http.MethodGet:     NewNode(initPath),
		http.MethodPost:    NewNode(initPath),
		http.MethodPatch:   NewNode(initPath),
		http.MethodDelete:  NewNode(initPath),
		http.MethodOptions: NewNode(initPath),
	}
}

func NewNode(path string) *node {
	return &node{
		path: path,
	}
}

// returns the length of the longest common substring between s1 and s2
func longestCommonString(s1, s2 string) int {
	l := min(len(s1), len(s2))
	i := 0

	for i < l && s1[i] == s2[i] {
		i++
	}

	return i
}

// get the first param name in the path
func getFirstParam(path string) (int, int) {
	start, end := -1, -1
	for i := range path {
		if path[i] == ':' {
			start = i
			for i < len(path) && path[i] != '/' {
				i++
			}
			end = i
			break
		}
	}

	return start, end
}

func (t *node) addNode(path string, handlers ...Chainable) {
	// list of param names in the path
	paramNames := make([]string, 0)

	for {
		l := longestCommonString(t.path, path)

		// split node
		if l < len(t.path) {
			newNode := &node{
				path:       t.path[l:],
				handlers:   t.handlers,
				children:   t.children,
				paramChild: t.paramChild,
				paramNames: t.paramNames,
			}

			t.path = t.path[:l]
			t.children = make(map[byte]*node)
			t.children = map[byte]*node{
				newNode.path[0]: newNode,
			}
			t.handlers = nil
			t.paramNames = nil
		}
		if l < len(path) {
			if t.path == ":" && path[0] == ':' {
				start, end := getFirstParam(path)

				if start == -1 {
					panic("missing param name")
				}
				paramNames = append(paramNames, path[start+1:end])
				path = path[end:]
				if len(path) == 0 {
					break
				}
			} else {
				path = path[l:]
			}

			// check if node has child matching first character of path
			if k, exists := t.children[path[0]]; exists {
				t = k
				continue
			}

			if t.paramChild != nil && path[0] == ':' {
				t = t.paramChild
				continue
			}

			t.insertChild(path, handlers, paramNames)
			return
		}

		// if path matches the current node, set handlers and paramNames
		t.handlers = handlers
		t.paramNames = paramNames
		return
	}
}

func (t *node) insertChild(path string, handlers []Chainable, paramNames []string) {
	for {
		start, end := getFirstParam(path)
		if start == -1 {
			t.addChild(&node{
				path:       path,
				handlers:   handlers,
				paramNames: paramNames,
			})
			return
		}

		if start == end {
			panic("Malformed url path: missing parameter name")
		}
		paramNames = append(paramNames, path[start+1:end])

		// add paramNode
		paramNode := &node{path: ":", isParam: true}

		if start > 0 {
			priorPath := path[:start]
			priorNode := &node{
				path: priorPath,
			}
			priorNode.addChild(paramNode)
			t.addChild(priorNode)
		} else {
			t.addChild(paramNode)
		}
		if end == len(path) {
			paramNode.paramNames = paramNames
			paramNode.handlers = handlers
			return
		}
		t = paramNode
		path = path[end:]
	}
}

func (t *node) addChild(n *node) {
	if n.isParam {
		t.paramChild = n
		return
	}
	if t.children == nil {
		t.children = make(map[byte]*node)
	}
	t.children[n.path[0]] = n
}

type routeValue struct {
	params       map[string]string
	handlerChain HandlerChain
}

func (t *node) match(path string) *routeValue {
	return t.matchRoute(path, []string{})
}

// match routes recursively
func (t *node) matchRoute(path string, paramValues []string) *routeValue {
	if t.path == ":" {
		start := 0
		for start < len(path) && path[start] != '/' {
			start++
		}
		paramValues = append(paramValues, path[:start])
		if start == len(path) {
			paramNames := t.paramNames
			params := make(map[string]string)
			for i, name := range paramNames {
				params[name] = paramValues[i]
			}

			return &routeValue{
				params:       params,
				handlerChain: NewHandlerChain(t.handlers...),
			}
		}
		path = path[start:]
		if k, exists := t.children[path[0]]; exists {
			t = k
		}
	}

	l := len(t.path)

	if l <= len(path) && path[:l] == t.path {
		path = path[l:]
		if path == "" {
			paramNames := t.paramNames
			params := make(map[string]string)
			for i, name := range paramNames {
				params[name] = paramValues[i]
			}

			return &routeValue{
				params:       params,
				handlerChain: NewHandlerChain(t.handlers...),
			}
		}

		// prioritise matching non parameter nodes first
		if k, exists := t.children[path[0]]; exists {
			if v := k.matchRoute(path, paramValues); v != nil {
				return v
			}
		}

		// if no matching segments found try matching parameter nodes
		if t.paramChild != nil {
			return t.paramChild.matchRoute(path, paramValues)
		}
	}

	return nil
}
