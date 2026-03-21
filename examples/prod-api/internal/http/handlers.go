package httpx

import (
	"strconv"
	"strings"
	"time"

	"github.com/dmesha3/elgon"
	"github.com/dmesha3/elgon/auth"
	"github.com/dmesha3/elgon/examples/prod-api/internal/app"
	"github.com/dmesha3/elgon/examples/prod-api/internal/domain"
	"github.com/dmesha3/elgon/jobs"
)

type Handlers struct {
	Repo  *app.TodoRepo
	JWT   *auth.JWTManager
	Queue jobs.Queue
}

func (h *Handlers) Health(c *elgon.Ctx) error {
	return c.JSON(200, map[string]string{"status": "ok"})
}

func (h *Handlers) Login(c *elgon.Ctx) error {
	var req domain.LoginRequest
	if err := c.BindJSON(&req); err != nil {
		return err
	}
	email := strings.TrimSpace(strings.ToLower(req.Email))
	if email == "" {
		return elgon.ErrBadRequest("email is required", nil)
	}
	claims := auth.Claims{Sub: email, Email: email, Roles: []string{"user"}, Perms: []string{"todos:read", "todos:write", "users:write"}}
	tok, err := h.JWT.Sign(claims, 24*time.Hour)
	if err != nil {
		return elgon.ErrInternal("token generation failed")
	}
	return c.JSON(200, domain.LoginResponse{AccessToken: tok, TokenType: "Bearer"})
}

func (h *Handlers) CreateUser(c *elgon.Ctx) error {
	var req domain.CreateUserRequest
	if err := c.BindJSON(&req); err != nil {
		return err
	}
	email := strings.TrimSpace(strings.ToLower(req.Email))
	name := strings.TrimSpace(req.Name)
	if email == "" {
		return elgon.ErrBadRequest("email is required", nil)
	}
	if name == "" {
		return elgon.ErrBadRequest("name is required", nil)
	}
	u, err := h.Repo.CreateUser(c.Request.Context(), email, name)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return elgon.ErrConflict("user already exists")
		}
		return elgon.ErrInternal("failed to create user")
	}
	return c.JSON(201, u)
}

func (h *Handlers) ListTodos(c *elgon.Ctx) error {
	items, err := h.Repo.List(c.Request.Context())
	if err != nil {
		return elgon.ErrInternal("failed to list todos")
	}
	return c.JSON(200, items)
}

func (h *Handlers) CreateTodo(c *elgon.Ctx) error {
	var req domain.CreateTodoRequest
	if err := c.BindJSON(&req); err != nil {
		return err
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return elgon.ErrBadRequest("title is required", nil)
	}
	t, err := h.Repo.Create(c.Request.Context(), title)
	if err != nil {
		return elgon.ErrInternal("failed to create todo")
	}
	if h.Queue != nil {
		_ = h.Queue.Enqueue(c.Request.Context(), jobs.Message{Name: "todo.created", Payload: []byte(strconv.FormatInt(t.ID, 10))})
	}

	return c.JSON(201, t)
}

func (h *Handlers) MarkDone(c *elgon.Ctx) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return elgon.ErrBadRequest("invalid id", nil)
	}
	t, ok, err := h.Repo.MarkDone(c.Request.Context(), id)
	if err != nil {
		return elgon.ErrInternal("failed to update todo")
	}
	if !ok {
		return elgon.ErrNotFound("todo not found")
	}
	return c.JSON(200, t)
}
