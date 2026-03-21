package app

import (
	"sync"
	"time"

	"github.com/dmesha3/elgon/examples/demo-app/internal/domain"
)

type TodoService struct {
	mu     sync.RWMutex
	nextID int64
	items  []domain.Todo
}

func NewTodoService() *TodoService {
	return &TodoService{nextID: 1, items: make([]domain.Todo, 0, 16)}
}

func (s *TodoService) List() []domain.Todo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]domain.Todo, len(s.items))
	copy(out, s.items)
	return out
}

func (s *TodoService) Create(title string) domain.Todo {
	s.mu.Lock()
	defer s.mu.Unlock()
	t := domain.Todo{ID: s.nextID, Title: title, Done: false, CreatedAt: time.Now().UTC()}
	s.nextID++
	s.items = append(s.items, t)
	return t
}

func (s *TodoService) MarkDone(id int64) (domain.Todo, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.items {
		if s.items[i].ID == id {
			s.items[i].Done = true
			return s.items[i], true
		}
	}
	return domain.Todo{}, false
}
