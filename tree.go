package treerouter

import (
	"net/http"
)

type node struct {
	path string

	// only endpoint node has at least one handlers
	handlers []Chainable
	children map[byte]*node

	// each node has at most one param child and one wildcard child
	paramChild *node
	wildChild  *node

	// only endpoint node has paramNames
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

type routeValue struct {
	params       map[string]string
	handlerChain HandlerChain
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

// get the first param name in the path, return the substring start and end indexes
func getFirstParam(path string) (int, int) {
	start, end := -1, -1
	for i := range path {
		if path[i] == ':' || path[i] == '*' {
			start = i
			for i < len(path) && path[i] != '/' {
				i++
			}
			end = i
			break
		}
	}

	if start > -1 {
		fc := path[start]
		if fc == ':' && start+1 >= end {
			panic("Malformed url path: missing parameter name")
		}
		if fc == '*' && end != len(path) {
			panic("Malformed url path: wildcard '*' must be at the end of path")
		}
	}

	return start, end
}

func (n *node) isLeaf() bool {
	return len(n.handlers) > 0
}

func (n *node) addNode(path string, handlers ...Chainable) {
	// list of param names in the path
	paramNames := make([]string, 0)

	for {
		l := longestCommonString(n.path, path)

		// split node
		if l < len(n.path) {
			newNode := &node{
				path:       n.path[l:],
				handlers:   n.handlers,
				children:   n.children,
				paramChild: n.paramChild,
				wildChild:  n.wildChild,
				paramNames: n.paramNames,
			}

			n.path = n.path[:l]
			n.children = make(map[byte]*node)
			n.children = map[byte]*node{
				newNode.path[0]: newNode,
			}
			n.handlers = nil
			n.paramNames = nil
		}
		if path[0] == ':' {
			start, end := getFirstParam(path)
			paramNames = append(paramNames, path[start+1:end])
			path = path[end:]
		} else {
			path = path[l:]
		}

		if path == "" {
			break
		}

		// check if node has child matching first character of path
		if k, exists := n.children[path[0]]; exists {
			n = k
			continue
		}

		if path[0] == ':' && n.paramChild != nil {
			n = n.paramChild
			continue
		}

		if path[0] == '*' && n.wildChild != nil {
			n = n.wildChild
			break
		}

		n.insertChild(path, handlers, paramNames)
		return
	}
	n.handlers = handlers
	n.paramNames = paramNames
}

func (n *node) insertChild(path string, handlers []Chainable, paramNames []string) {
	for {
		start, end := getFirstParam(path)
		if start == -1 {
			n.addChild(&node{
				path:       path,
				handlers:   handlers,
				paramNames: paramNames,
			})
			return
		}

		var dynamNode, priorNode *node

		if start > 0 {
			priorNode = &node{
				path: path[:start],
			}
		}

		fc := path[start]

		// both param child (:) and wild child (*) uses single character path
		dynamNode = &node{path: string(fc)}
		if fc == ':' {
			paramNames = append(paramNames, path[start+1:end])
		}

		// if there is prior path insert that node as parent node
		if priorNode != nil {
			priorNode.addChild(dynamNode)
			n.addChild(priorNode)
		} else {
			n.addChild(dynamNode)
		}

		if end == len(path) {
			dynamNode.paramNames = paramNames
			dynamNode.handlers = handlers
			return
		}
		n = dynamNode
		path = path[end:]
	}
}

func (n *node) addChild(c *node) {
	if c.path == ":" {
		n.paramChild = c
		return
	}

	if c.path == "*" {
		n.wildChild = c
		return
	}
	if n.children == nil {
		n.children = make(map[byte]*node)
	}
	n.children[c.path[0]] = c
}

func (n *node) match(path string) *routeValue {
	return n.matchRoute(path, []string{})
}

// match routes recursively
func (n *node) matchRoute(path string, paramValues []string) *routeValue {
	if n.path == ":" {
		start := 0
		for start < len(path) && path[start] != '/' {
			start++
		}
		paramValues = append(paramValues, path[:start])
		if start == len(path) {
			if n.isLeaf() {
				paramNames := n.paramNames
				params := make(map[string]string)
				for i, name := range paramNames {
					params[name] = paramValues[i]
				}

				return &routeValue{
					params:       params,
					handlerChain: NewHandlerChain(n.handlers...),
				}
			}
			return nil
		}
		path = path[start:]
		if k, exists := n.children[path[0]]; exists {
			n = k
		}
	}

	l := len(n.path)
	if l <= len(path) && path[:l] == n.path {
		path = path[l:]
		if path == "" {
			if n.isLeaf() {
				paramNames := n.paramNames
				params := make(map[string]string)
				for i, name := range paramNames {
					params[name] = paramValues[i]
				}

				return &routeValue{
					params:       params,
					handlerChain: NewHandlerChain(n.handlers...),
				}
			}
			return nil
		}

		// prioritise matching non parameter/wildcard nodes first
		if k, exists := n.children[path[0]]; exists {
			if v := k.matchRoute(path, paramValues); v != nil {
				return v
			}
		}

		// if no matching segments found try matching parameter nodes first
		if n.paramChild != nil {
			if v := n.paramChild.matchRoute(path, paramValues); v != nil {
				return v
			}
		}

		// match wildcard child last as it has lowest priority
		if n.wildChild != nil {
			paramNames := n.wildChild.paramNames
			params := make(map[string]string)
			for i, name := range paramNames {
				params[name] = paramValues[i]
			}

			// set wildcard param name to "*"
			params["*"] = path
			return &routeValue{
				params:       params,
				handlerChain: NewHandlerChain(n.wildChild.handlers...),
			}
		}
	}

	return nil
}
