package domain

import "time"

type Todo struct {
	ID        int64     `json:"id" openapi:"example=1"`
	Title     string    `json:"title" description:"Short todo title" openapi:"minLength=1,maxLength=120,example=Ship demo app"`
	Done      bool      `json:"done" openapi:"example=false"`
	CreatedAt time.Time `json:"created_at" description:"Creation timestamp"`
}

type CreateTodoRequest struct {
	Title string `json:"title" description:"Short todo title" openapi:"minLength=1,maxLength=120,example=Ship demo app"`
}

// LoginRequest is demo-only auth payload.
type LoginRequest struct {
	Email string `json:"email" openapi:"format=email,example=admin@example.com"`
}

// LoginResponse returns a signed demo JWT.
type LoginResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type" openapi:"example=Bearer"`
}
