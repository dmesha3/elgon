package elgon

// HandlerFunc handles an HTTP request and returns an error for centralized handling.
type HandlerFunc func(*Ctx) error

// Middleware wraps a handler with extra behavior.
type Middleware func(HandlerFunc) HandlerFunc
