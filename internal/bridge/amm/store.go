package amm

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type Store struct {
	mu    sync.RWMutex
	path  string
	Pools map[string]*Pool
}

func NewStore(dir string) *Store {
	return &Store{
		path:  filepath.Join(dir, "pools.json"),
		Pools: make(map[string]*Pool),
	}
}

func (s *Store) Load(seed []Pool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			for i := range seed {
				p := seed[i]
				if p.FeeBps == 0 {
					p.FeeBps = DefaultFeeBps
				}
				if p.ID == "" {
					p.ID = PoolID(p.Token0, p.Token1)
				}
				s.Pools[p.ID] = &p
			}
			return s.persistLocked()
		}
		return err
	}
	var list []Pool
	if err := json.Unmarshal(data, &list); err != nil {
		return err
	}
	s.Pools = make(map[string]*Pool, len(list))
	for i := range list {
		p := list[i]
		s.Pools[p.ID] = &p
	}
	return nil
}

func (s *Store) persistLocked() error {
	_ = os.MkdirAll(filepath.Dir(s.path), 0o755)
	list := make([]Pool, 0, len(s.Pools))
	for _, p := range s.Pools {
		list = append(list, *p)
	}
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o644)
}

func (s *Store) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.persistLocked()
}

func (s *Store) Get(id string) (*Pool, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.Pools[id]
	if !ok {
		return nil, false
	}
	cp := *p
	return &cp, true
}

func (s *Store) FindPool(tokenA, tokenB string) (*Pool, bool) {
	id := PoolID(tokenA, tokenB)
	return s.Get(id)
}

func (s *Store) List() []Pool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Pool, 0, len(s.Pools))
	for _, p := range s.Pools {
		out = append(out, *p)
	}
	return out
}

func (s *Store) Update(p *Pool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Pools[p.ID] = p
	return s.persistLocked()
}
