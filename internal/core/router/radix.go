package router

import (
	"errors"
	"sort"
	"strings"
	"sync"

	"github.com/apirout/apirout/internal/core/models"
)

var (
	ErrNotFound         = errors.New("route not found")
	ErrMethodNotAllowed = errors.New("method not allowed")
)

// MatchResult contains the matched endpoint and any extracted path parameters
type MatchResult struct {
	Endpoint       *models.Endpoint
	Params         map[string]string
	AllowedMethods []string
}

type trieNode struct {
	segment        string
	isParam        bool
	paramName      string
	isWildcard     bool
	children       map[string]*trieNode
	paramChild     *trieNode
	wildChild      *trieNode
	endpoints      map[string]*models.Endpoint // method -> endpoint
	prefixEndpoint *models.Endpoint            // fallback for prefix matching
}

func newTrieNode(segment string) *trieNode {
	return &trieNode{
		segment:   segment,
		children:  make(map[string]*trieNode),
		endpoints: make(map[string]*models.Endpoint),
	}
}

// RadixRouter is a thread-safe segment-trie in-memory router
type RadixRouter struct {
	mu    sync.RWMutex
	root  *trieNode
	count int
}

// NewRadixRouter creates an empty RadixRouter
func NewRadixRouter() *RadixRouter {
	return &RadixRouter{
		root: newTrieNode("/"),
	}
}

// BuildFromEndpoints builds a new RadixRouter populated with the given endpoints
func BuildFromEndpoints(endpoints []models.Endpoint) *RadixRouter {
	router := NewRadixRouter()
	for i := range endpoints {
		_ = router.AddRoute(&endpoints[i])
	}
	return router
}

func tokenizePath(path string) []string {
	// Strip query string if present
	if idx := strings.IndexByte(path, '?'); idx != -1 {
		path = path[:idx]
	}

	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return []string{}
	}
	return strings.Split(trimmed, "/")
}

// AddRoute registers an endpoint into the Radix tree
func (r *RadixRouter) AddRoute(ep *models.Endpoint) error {
	if ep == nil || ep.Path == "" {
		return errors.New("invalid endpoint or empty path")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	segments := tokenizePath(ep.Path)
	curr := r.root

	for _, seg := range segments {
		if seg == "*" {
			if curr.wildChild == nil {
				curr.wildChild = newTrieNode("*")
				curr.wildChild.isWildcard = true
			}
			curr = curr.wildChild
			break
		} else if strings.HasPrefix(seg, ":") {
			paramName := seg[1:]
			if curr.paramChild == nil {
				curr.paramChild = newTrieNode(seg)
				curr.paramChild.isParam = true
				curr.paramChild.paramName = paramName
			}
			curr = curr.paramChild
		} else {
			child, exists := curr.children[seg]
			if !exists {
				child = newTrieNode(seg)
				curr.children[seg] = child
			}
			curr = child
		}
	}

	method := strings.ToUpper(strings.TrimSpace(ep.Method))
	if method == "" {
		method = "ANY"
	}
	curr.endpoints[method] = ep
	curr.prefixEndpoint = ep
	r.count++

	return nil
}

// Match resolves a path and HTTP method against the compiled Radix tree
func (r *RadixRouter) Match(path, method string) (*MatchResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	segments := tokenizePath(path)
	method = strings.ToUpper(strings.TrimSpace(method))

	params := make(map[string]string)
	targetNode, foundParams := r.search(r.root, segments, 0, params)

	if targetNode == nil || (len(targetNode.endpoints) == 0 && targetNode.prefixEndpoint == nil) {
		return nil, ErrNotFound
	}

	// 1. Check exact method or ANY
	ep, ok := targetNode.endpoints[method]
	if !ok {
		ep, ok = targetNode.endpoints["ANY"]
	}

	// If no exact leaf endpoint, fall back to prefix endpoint
	if !ok && targetNode.prefixEndpoint != nil {
		pEp := targetNode.prefixEndpoint
		pMethod := strings.ToUpper(pEp.Method)
		if pMethod == "ANY" || pMethod == method {
			ep = pEp
			ok = true
		}
	}

	if ok && ep != nil {
		return &MatchResult{
			Endpoint: ep,
			Params:   foundParams,
		}, nil
	}

	// Path matches, but method is not allowed -> Collect allowed methods
	allowedMap := make(map[string]struct{})
	for m := range targetNode.endpoints {
		if m == "ANY" {
			allowedMap["GET"] = struct{}{}
			allowedMap["POST"] = struct{}{}
			allowedMap["PUT"] = struct{}{}
			allowedMap["DELETE"] = struct{}{}
			allowedMap["PATCH"] = struct{}{}
		} else {
			allowedMap[m] = struct{}{}
		}
	}
	if targetNode.prefixEndpoint != nil {
		pm := strings.ToUpper(targetNode.prefixEndpoint.Method)
		if pm == "ANY" {
			allowedMap["GET"] = struct{}{}
			allowedMap["POST"] = struct{}{}
			allowedMap["PUT"] = struct{}{}
			allowedMap["DELETE"] = struct{}{}
		} else {
			allowedMap[pm] = struct{}{}
		}
	}

	allowedList := make([]string, 0, len(allowedMap))
	for m := range allowedMap {
		allowedList = append(allowedList, m)
	}
	sort.Strings(allowedList)

	return &MatchResult{
		AllowedMethods: allowedList,
	}, ErrMethodNotAllowed
}

func (r *RadixRouter) search(node *trieNode, segments []string, idx int, params map[string]string) (*trieNode, map[string]string) {
	if idx == len(segments) {
		return node, params
	}

	seg := segments[idx]

	// 1. Priority 1: Exact child segment
	if child, exists := node.children[seg]; exists {
		res, p := r.search(child, segments, idx+1, params)
		if res != nil {
			return res, p
		}
	}

	// 2. Priority 2: Parameterized child segment (:param)
	if node.paramChild != nil {
		newParams := copyParams(params)
		newParams[node.paramChild.paramName] = seg
		res, p := r.search(node.paramChild, segments, idx+1, newParams)
		if res != nil {
			return res, p
		}
	}

	// 3. Priority 3: Wildcard child segment (*)
	if node.wildChild != nil {
		return node.wildChild, params
	}

	// 4. Priority 4: Fall back to longest prefix ancestor if available
	if node.prefixEndpoint != nil {
		return node, params
	}

	return nil, params
}

func copyParams(src map[string]string) map[string]string {
	dst := make(map[string]string, len(src)+1)
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

// RoutesCount returns the total number of registered routes in the tree
func (r *RadixRouter) RoutesCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.count
}
