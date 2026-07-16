package telegram

import (
	"sync"
	"time"
)

type Session struct {
	UserID int64
	Lines  []string
	timer  *time.Timer
	mu     sync.Mutex
}

type SessionManager struct {
	debounce time.Duration
	sessions map[int64]*Session
	onFlush  func(userID int64, lines []string)
	mu       sync.Mutex
}

func NewSessionManager(debounceSeconds int, onFlush func(userID int64, lines []string)) *SessionManager {
	if debounceSeconds <= 0 {
		debounceSeconds = 10
	}
	return &SessionManager{
		debounce: time.Duration(debounceSeconds) * time.Second,
		sessions: make(map[int64]*Session),
		onFlush:  onFlush,
	}
}

func (m *SessionManager) Add(userID int64, lines []string) int {
	m.mu.Lock()
	session, ok := m.sessions[userID]
	if !ok {
		session = &Session{UserID: userID}
		m.sessions[userID] = session
	}
	m.mu.Unlock()

	return session.add(lines, m.debounce, func(uid int64, collected []string) {
		m.mu.Lock()
		delete(m.sessions, uid)
		m.mu.Unlock()
		m.onFlush(uid, collected)
	})
}

func (m *SessionManager) Flush(userID int64) []string {
	m.mu.Lock()
	session, ok := m.sessions[userID]
	if ok {
		delete(m.sessions, userID)
	}
	m.mu.Unlock()
	if !ok {
		return nil
	}
	return session.drain()
}

func (m *SessionManager) Cancel(userID int64) int {
	m.mu.Lock()
	session, ok := m.sessions[userID]
	if ok {
		delete(m.sessions, userID)
	}
	m.mu.Unlock()
	if !ok {
		return 0
	}
	return session.clear()
}

func (m *SessionManager) Count(userID int64) int {
	m.mu.Lock()
	session, ok := m.sessions[userID]
	m.mu.Unlock()
	if !ok {
		return 0
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	return len(session.Lines)
}

func (s *Session) add(lines []string, debounce time.Duration, flushFn func(userID int64, lines []string)) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	seen := make(map[string]struct{}, len(s.Lines))
	for _, line := range s.Lines {
		seen[line] = struct{}{}
	}
	added := 0
	for _, line := range lines {
		if _, ok := seen[line]; ok {
			continue
		}
		seen[line] = struct{}{}
		s.Lines = append(s.Lines, line)
		added++
	}

	if s.timer != nil {
		s.timer.Stop()
	}
	userID := s.UserID
	s.timer = time.AfterFunc(debounce, func() {
		collected := s.drainLocked()
		if len(collected) > 0 {
			flushFn(userID, collected)
		}
	})
	return added
}

func (s *Session) drain() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.drainLocked()
}

func (s *Session) drainLocked() []string {
	if s.timer != nil {
		s.timer.Stop()
		s.timer = nil
	}
	if len(s.Lines) == 0 {
		return nil
	}
	out := append([]string(nil), s.Lines...)
	s.Lines = nil
	return out
}

func (s *Session) clear() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.timer != nil {
		s.timer.Stop()
		s.timer = nil
	}
	count := len(s.Lines)
	s.Lines = nil
	return count
}
