package targetpool

import (
	"crypto/md5"
	"encoding/binary"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/apirout/apirout/internal/core/httputil"
	"github.com/apirout/apirout/internal/core/models"
)

var (
	// ErrNoTargets is returned when selecting a target from an empty candidate list.
	ErrNoTargets = errors.New("no healthy upstream targets available")
)

// TargetPool selects upstream target instances using configured load-balancing algorithms
// and tracks scoped connection lease lifecycles.
type TargetPool interface {
	SelectTarget(req *http.Request, upstream *models.Upstream, availableTargets []models.Target) (*models.Target, func(), error)
	ActiveConns(targetID string) int64
	ActiveTargetCount() int
}

// RingPoint represents a virtual node point on the 32-bit Ketama hash ring.
type RingPoint struct {
	Hash   uint32
	Target models.Target
}

// Ring implements a Ketama consistent hash ring with 160 virtual nodes per target instance.
type Ring struct {
	points []RingPoint
}

// NewRing constructs a Ketama hash ring for the specified targets with 160 virtual nodes per target.
func NewRing(targets []models.Target) *Ring {
	if len(targets) == 0 {
		return &Ring{points: nil}
	}

	points := make([]RingPoint, 0, len(targets)*160)
	for _, target := range targets {
		nodeKey := target.ID
		if nodeKey == "" {
			nodeKey = fmt.Sprintf("%s:%d", target.Host, target.Port)
		}
		for i := 0; i < 160; i++ {
			sum := md5.Sum([]byte(fmt.Sprintf("%s-%d", nodeKey, i)))
			point := binary.BigEndian.Uint32(sum[0:4])
			points = append(points, RingPoint{
				Hash:   point,
				Target: target,
			})
		}
	}

	sort.Slice(points, func(i, j int) bool {
		return points[i].Hash < points[j].Hash
	})

	return &Ring{points: points}
}

// Get finds the target corresponding to the given hash key clockwise on the ring.
func (r *Ring) Get(key string) *models.Target {
	if len(r.points) == 0 {
		return nil
	}
	sum := md5.Sum([]byte(key))
	h := binary.BigEndian.Uint32(sum[0:4])

	idx := sort.Search(len(r.points), func(i int) bool {
		return r.points[i].Hash >= h
	})
	if idx == len(r.points) {
		idx = 0 // circular ring wrap-around
	}

	target := r.points[idx].Target
	return &target
}

type ringCacheEntry struct {
	fingerprint string
	ring        *Ring
}

// Pool manages upstream target balancing and connection leases.
type Pool struct {
	mu          sync.RWMutex
	rrCounters  map[string]*uint64
	activeConns map[string]int64
	rings       map[string]*ringCacheEntry
}

// New creates an initialized TargetPool.
func New() *Pool {
	return &Pool{
		rrCounters:  make(map[string]*uint64),
		activeConns: make(map[string]int64),
		rings:       make(map[string]*ringCacheEntry),
	}
}

// ExtractHashKey extracts consistent hashing key in priority order:
// 1. X-Consumer-ID header
// 2. X-User-ID header
// 3. Client IP fallback (from proxy headers or RemoteAddr)
func ExtractHashKey(req *http.Request) string {
	if req == nil {
		return "127.0.0.1"
	}
	if cid := strings.TrimSpace(req.Header.Get("X-Consumer-ID")); cid != "" {
		return cid
	}
	if uid := strings.TrimSpace(req.Header.Get("X-User-ID")); uid != "" {
		return uid
	}
	clientIP := strings.TrimSpace(httputil.GetClientIP(req))
	if clientIP != "" {
		return clientIP
	}
	if req.RemoteAddr != "" {
		return strings.TrimSpace(req.RemoteAddr)
	}
	return "127.0.0.1"
}

// SelectTarget chooses a target instance using the specified Upstream algorithm:
// "round-robin", "consistent-hashing", or "least-connections".
// It returns the selected target, a release closure to decrement the connection lease, or an error.
func (p *Pool) SelectTarget(req *http.Request, upstream *models.Upstream, availableTargets []models.Target) (*models.Target, func(), error) {
	if len(availableTargets) == 0 {
		return nil, func() {}, ErrNoTargets
	}
	if len(availableTargets) == 1 {
		t := availableTargets[0]
		return &t, p.acquireConn(t.ID), nil
	}

	algo := "round-robin"
	upstreamID := "default"
	if upstream != nil {
		if upstream.Algorithm != "" {
			algo = upstream.Algorithm
		}
		if upstream.ID != "" {
			upstreamID = upstream.ID
		}
	}

	switch strings.ToLower(algo) {
	case "round-robin":
		idx := p.nextRoundRobin(upstreamID, len(availableTargets))
		t := availableTargets[idx]
		return &t, p.acquireConn(t.ID), nil

	case "consistent-hashing":
		hashKey := ExtractHashKey(req)
		t := p.selectConsistentHash(upstreamID, availableTargets, hashKey)
		if t == nil {
			t = &availableTargets[0]
		}
		return t, p.acquireConn(t.ID), nil

	case "least-connections":
		t := p.selectLeastConnections(availableTargets)
		return t, p.acquireConn(t.ID), nil

	default: // round-robin fallback
		idx := p.nextRoundRobin(upstreamID, len(availableTargets))
		t := availableTargets[idx]
		return &t, p.acquireConn(t.ID), nil
	}
}

func (p *Pool) nextRoundRobin(upstreamID string, numTargets int) int {
	if numTargets <= 0 {
		return 0
	}

	p.mu.RLock()
	counter, exists := p.rrCounters[upstreamID]
	p.mu.RUnlock()

	if !exists {
		p.mu.Lock()
		counter, exists = p.rrCounters[upstreamID]
		if !exists {
			var n uint64
			counter = &n
			p.rrCounters[upstreamID] = counter
		}
		p.mu.Unlock()
	}

	val := atomic.AddUint64(counter, 1) - 1
	return int(val % uint64(numTargets))
}

func targetsFingerprint(targets []models.Target) string {
	ids := make([]string, len(targets))
	for i, t := range targets {
		if t.ID != "" {
			ids[i] = t.ID
		} else {
			ids[i] = fmt.Sprintf("%s:%d", t.Host, t.Port)
		}
	}
	sort.Strings(ids)
	return strings.Join(ids, ",")
}

func (p *Pool) selectConsistentHash(upstreamID string, targets []models.Target, hashKey string) *models.Target {
	fp := targetsFingerprint(targets)

	p.mu.RLock()
	cached, ok := p.rings[upstreamID]
	if ok && cached.fingerprint == fp {
		ring := cached.ring
		p.mu.RUnlock()
		return ring.Get(hashKey)
	}
	p.mu.RUnlock()

	newRing := NewRing(targets)

	p.mu.Lock()
	p.rings[upstreamID] = &ringCacheEntry{
		fingerprint: fp,
		ring:        newRing,
	}
	p.mu.Unlock()

	return newRing.Get(hashKey)
}

func (p *Pool) selectLeastConnections(targets []models.Target) *models.Target {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var minConn int64 = 1<<62 - 1
	var selected *models.Target = &targets[0]

	for i := range targets {
		conns := p.activeConns[targets[i].ID]
		if conns < minConn {
			minConn = conns
			selected = &targets[i]
		}
	}

	cpy := *selected
	return &cpy
}

func (p *Pool) acquireConn(targetID string) func() {
	if targetID == "" {
		return func() {}
	}

	p.mu.Lock()
	p.activeConns[targetID]++
	p.mu.Unlock()

	var once sync.Once
	return func() {
		once.Do(func() {
			p.releaseConn(targetID)
		})
	}
}

func (p *Pool) releaseConn(targetID string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	count, exists := p.activeConns[targetID]
	if !exists {
		return
	}
	if count <= 1 {
		delete(p.activeConns, targetID)
	} else {
		p.activeConns[targetID]--
	}
}

// ActiveConns returns the number of active connection leases currently held for the target.
func (p *Pool) ActiveConns(targetID string) int64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.activeConns[targetID]
}

// ActiveTargetCount returns the count of targets that currently have active connections.
func (p *Pool) ActiveTargetCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.activeConns)
}
