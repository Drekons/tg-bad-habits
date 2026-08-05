package bot

import "sync"

// userSerial serializes handlers per Telegram user to avoid FSM/DB races.
type userSerial struct {
	mu    sync.Mutex
	gates map[int64]*sync.Mutex
}

func newUserSerial() *userSerial {
	return &userSerial{gates: make(map[int64]*sync.Mutex)}
}

func (s *userSerial) run(userID int64, fn func()) {
	s.mu.Lock()
	g, ok := s.gates[userID]
	if !ok {
		g = &sync.Mutex{}
		s.gates[userID] = g
	}
	s.mu.Unlock()

	g.Lock()
	defer g.Unlock()
	fn()
}
