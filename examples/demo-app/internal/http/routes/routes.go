package routes

import (
	"net/http"

	"github.com/meshackkazimoto/elgon"
	"github.com/meshackkazimoto/elgon/auth"
	"github.com/meshackkazimoto/elgon/examples/demo-app/internal/domain"
	"github.com/meshackkazimoto/elgon/examples/demo-app/internal/http/handlers"
	"github.com/meshackkazimoto/elgon/openapi"
)

func Register(app *elgon.App, api *handlers.API, jwt *auth.JWTManager) {
	app.GET("/healthz", api.Health)
	app.POST("/auth/login", api.Login)

	secured := app.Group("/api/v1", auth.Auth(jwt), auth.RequirePerm("todos:write"))
	secured.GET("/todos", api.ListTodos)
	secured.POST("/todos", api.CreateTodo)
	secured.PATCH("/todos/:id/done", api.MarkDone)

	docs := openapi.NewGenerator("elgon demo API", elgon.Version)
	docs.Description = "Demo app showing how to build production-style APIs with elgon."
	docs.EnableBearerAuth()
	docs.AddOperation(http.MethodPost, "/auth/login", openapi.Operation{
		Summary:       "Login",
		Description:   "Returns a JWT for demo usage.",
		OperationID:   "login",
		Tags:          []string{"auth"},
		RequestModel:  domain.LoginRequest{},
		ResponseModel: domain.LoginResponse{},
		RequestExample: map[string]any{
			"email": "admin@example.com",
		},
	})
	docs.AddOperation(http.MethodGet, "/api/v1/todos", openapi.Operation{
		Summary:       "List todos",
		OperationID:   "listTodos",
		Tags:          []string{"todos"},
		RequiresAuth:  true,
		ResponseModel: []domain.Todo{},
	})
	docs.AddOperation(http.MethodPost, "/api/v1/todos", openapi.Operation{
		Summary:       "Create todo",
		OperationID:   "createTodo",
		Tags:          []string{"todos"},
		RequiresAuth:  true,
		RequestModel:  domain.CreateTodoRequest{},
		ResponseModel: domain.Todo{},
		ResponseCode:  201,
		RequestExample: map[string]any{
			"title": "Ship demo app",
		},
	})
	docs.AddOperation(http.MethodPatch, "/api/v1/todos/:id/done", openapi.Operation{
		Summary:       "Mark done",
		OperationID:   "markTodoDone",
		Tags:          []string{"todos"},
		RequiresAuth:  true,
		ResponseModel: domain.Todo{},
	})
	docs.Register(app, "/openapi.json", "/docs")
}
