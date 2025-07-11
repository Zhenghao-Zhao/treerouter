package treerouter

import (
	"net/http"
	"strings"
	"unicode"
)

type routes []route

type route struct {
	method string
	node   *node
}

type node struct {
	path string

	// only endpoint node has at least one handler
	handler  http.Handler
	children map[byte]*node

	// each node has at most one param child and one wildcard child
	paramChild *node
	wildChild  *node

	// only endpoint node has paramNames
	paramNames []string
}

type routeValue struct {
	params  map[string]string
	handler http.Handler
	tsr     bool
}

func newMethodRoot() routes {
	initPath := "/"
	return routes{
		route{method: http.MethodGet, node: newNode(initPath)},
		route{method: http.MethodPost, node: newNode(initPath)},
		route{method: http.MethodPut, node: newNode(initPath)},
		route{method: http.MethodPatch, node: newNode(initPath)},
		route{method: http.MethodDelete, node: newNode(initPath)},
	}
}

func (m routes) get(method string) *node {
	for _, methodNode := range m {
		if methodNode.method == method {
			return methodNode.node
		}
	}
	return nil
}

func newNode(path string) *node {
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

// get the first param name in the path, return its start and end indexes including ':' or '*'
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
	return n.handler != nil
}

func (n *node) addNode(path string, handlers http.Handler) {
	// list of param names in the path
	paramNames := make([]string, 0)

	for {
		l := longestCommonString(n.path, path)

		// split node
		if l < len(n.path) {
			newNode := &node{
				path:       n.path[l:],
				handler:    n.handler,
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
			n.handler = nil
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
	n.handler = handlers
	n.paramNames = paramNames
}

// insertChild adds node that has no common path with the parent node
func (n *node) insertChild(path string, handlers http.Handler, paramNames []string) {
	for {
		start, end := getFirstParam(path)
		if start == -1 {
			n.addChild(&node{
				path:       path,
				handler:    handlers,
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
			dynamNode.handler = handlers
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

func newRouteValue(n *node, paramVals []string, tsr bool) *routeValue {
	if !n.isLeaf() {
		return nil
	}

	if len(n.paramNames) != len(paramVals) {
		panic("missing param name(s) or param val(s)")
	}

	if n.handler == nil {
		panic("missing handler")
	}

	params := make(map[string]string)
	for i := range len(n.paramNames) {
		params[n.paramNames[i]] = paramVals[i]
	}

	return &routeValue{
		params:  params,
		handler: n.handler,
		tsr:     tsr,
	}
}

func (n *node) matchRoute(path string, paramValues []string) *routeValue {
	l, k := len(n.path), len(path)
	if n.path == ":" {
		// find the index at the next '/' or EOL of path
		end := 0
		for end < len(path) && path[end] != '/' {
			end++
		}
		paramValues = append(paramValues, path[:end])
		path = path[end:]
	} else if l <= k && n.path == path[:l] {
		path = path[l:]
	} else {
		if l == k+1 && n.path[:k] == path && n.path[k] == '/' {
			return newRouteValue(n, paramValues, true)
		}
		return nil
	}

	if len(path) == 0 {
		return newRouteValue(n, paramValues, false)
	}
	if node, exists := n.children[path[0]]; exists {
		if v := node.matchRoute(path, paramValues); v != nil {
			return v
		}
	} else if path == "/" {
		return newRouteValue(n, paramValues, true)
	}

	// if no matching segments found try matching parameter nodes first
	if n.paramChild != nil {
		if v := n.paramChild.matchRoute(path, paramValues); v != nil {
			return v
		}
	}

	if n.wildChild != nil {
		v := newRouteValue(n.wildChild, paramValues, false)
		// set wildcard param name to "*"
		v.params["*"] = path
		return v
	}

	return nil
}

func (n *node) findCaseInsensitivePath(path string, tsr bool) (string, bool) {
	buffer := make([]byte, 0, len(path)+1)
	result := n.findCaseInsensitivePathRec(path, buffer, tsr)

	if result == nil {
		return "", false
	}

	return string(result), true
}

func (n *node) findCaseInsensitivePathRec(path string, buffer []byte, tsr bool) []byte {
	l, k := len(n.path), len(path)
	if n.path == ":" {
		// find the index at the next '/' or EOL of path
		end := 0
		for end < len(path) && path[end] != '/' {
			end++
		}
		buffer = append(buffer, []byte(path[:end])...)
		path = path[end:]
	} else if l <= k && strings.EqualFold(n.path, path[:l]) {
		buffer = append(buffer, []byte(n.path)...)
		path = path[l:]
	} else {
		if tsr && l == k+1 && strings.EqualFold(n.path[:k], path) && n.path[k] == '/' {
			buffer = append(buffer, []byte(n.path[:k])...)
			return buffer
		}
		return nil
	}

	if len(path) == 0 || (tsr && path == "/") {
		return buffer
	}

	lower := byte(unicode.ToLower(rune(path[0])))
	if node, exists := n.children[lower]; exists {
		if v := node.findCaseInsensitivePathRec(path, buffer, tsr); v != nil {
			return v
		}
	}

	upper := byte(unicode.ToUpper(rune(path[0])))
	if node, exists := n.children[upper]; exists {
		if v := node.findCaseInsensitivePathRec(path, buffer, tsr); v != nil {
			return v
		}
	}

	// if no matching segments found try matching parameter nodes first
	if n.paramChild != nil {
		if v := n.paramChild.findCaseInsensitivePathRec(path, buffer, tsr); v != nil {
			return v
		}
	}

	// match wildcard child last as it has lowest priority
	if n.wildChild != nil {
		buffer = append(buffer, []byte(path)...)
		return buffer
	}

	return nil
}
