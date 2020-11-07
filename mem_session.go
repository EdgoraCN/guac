package guac

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/sirupsen/logrus"
)

// MemorySessionStore is a simple in-memory store of connected sessions that is used by
// the WebsocketServer to store active sessions.
type MemorySessionStore struct {
	sync.RWMutex
	ConnIds map[string]int
}

// NewMemorySessionStore creates a new store
func NewMemorySessionStore() *MemorySessionStore {
	return &MemorySessionStore{
		ConnIds: map[string]int{},
	}
}

// Get returns a connection by uuid
func (s *MemorySessionStore) Get(id string) int {
	s.RLock()
	defer s.RUnlock()
	return s.ConnIds[id]
}

// Add inserts a new connection by uuid
func (s *MemorySessionStore) Add(id string, req *http.Request) {
	s.Lock()
	defer s.Unlock()
	n, ok := s.ConnIds[id]
	if !ok {
		s.ConnIds[id] = 1
		return
	}
	n++
	s.ConnIds[id] = n
	return
}

// Delete removes a connection by uuid
func (s *MemorySessionStore) Delete(id string, req *http.Request, tunnel Tunnel) {
	s.Lock()
	defer s.Unlock()
	n, ok := s.ConnIds[id]
	if !ok {
		return
	}
	if n == 1 {
		delete(s.ConnIds, id)
		return
	}
	s.ConnIds[id]--
	return
}

// HandleSession handle session
func (s *MemorySessionStore) HandleSession(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	s.RLock()
	defer s.RUnlock()

	type ConnIds struct {
		Uuid string `json:"uuid"`
		Num  int    `json:"num"`
	}

	connIds := make([]*ConnIds, len(s.ConnIds))

	i := 0
	for id, num := range s.ConnIds {
		connIds[i] = &ConnIds{
			Uuid: id,
			Num:  num,
		}
	}

	if err := json.NewEncoder(w).Encode(connIds); err != nil {
		logrus.Error(err)
	}
}
