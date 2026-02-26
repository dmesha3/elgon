package handlers

import (
	"strconv"
	"strings"
	"time"

	"github.com/meshackkazimoto/elgon"
	"github.com/meshackkazimoto/elgon/auth"
	"github.com/meshackkazimoto/elgon/examples/demo-app/internal/app"
	"github.com/meshackkazimoto/elgon/examples/demo-app/internal/domain"
	"github.com/meshackkazimoto/elgon/jobs"
)

type API struct {
	Todos *app.TodoService
	JWT   *auth.JWTManager
	Queue jobs.Queue
}

func (a *API) Health(c *elgon.Ctx) error {
	return c.JSON(200, map[string]string{"status": "ok"})
}

func (a *API) Login(c *elgon.Ctx) error {
	var req domain.LoginRequest
	if err := c.BindJSON(&req); err != nil {
		return err
	}
	email := strings.TrimSpace(strings.ToLower(req.Email))
	if email == "" {
		return elgon.ErrBadRequest("email is required", nil)
	}
	roles := []string{"user"}
	perms := []string{"todos:read", "todos:write"}
	if email == "admin@example.com" {
		roles = append(roles, "admin")
		perms = append(perms, "todos:admin")
	}
	token, err := a.JWT.Sign(auth.Claims{
		Sub:   email,
		Email: email,
		Roles: roles,
		Perms: perms,
	}, 24*time.Hour)
	if err != nil {
		return elgon.ErrInternal("failed to create token")
	}
	return c.JSON(200, domain.LoginResponse{AccessToken: token, TokenType: "Bearer"})
}

func (a *API) ListTodos(c *elgon.Ctx) error {
	return c.JSON(200, a.Todos.List())
}

func (a *API) CreateTodo(c *elgon.Ctx) error {
	var req domain.CreateTodoRequest
	if err := c.BindJSON(&req); err != nil {
		return err
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return elgon.ErrBadRequest("title is required", nil)
	}
	todo := a.Todos.Create(title)
	if a.Queue != nil {
		_ = a.Queue.Enqueue(c.Request.Context(), jobs.Message{Name: "todo.created", Payload: []byte(title)})
	}
	return c.JSON(201, todo)
}

func (a *API) MarkDone(c *elgon.Ctx) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return elgon.ErrBadRequest("invalid todo id", nil)
	}
	todo, ok := a.Todos.MarkDone(id)
	if !ok {
		return elgon.ErrNotFound("todo not found")
	}
	return c.JSON(200, todo)
}
