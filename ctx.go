package elgon

import (
	"encoding/json"
	"io"
	"net/http"
)

// Ctx wraps request and response objects plus request-scoped helpers.
type Ctx struct {
	Writer  http.ResponseWriter
	Request *http.Request

	params       []routeParam
	values       map[string]any
	app          *App
	routePattern string
}

func (c *Ctx) reset(w http.ResponseWriter, r *http.Request) {
	c.Writer = w
	c.Request = r
	c.params = c.params[:0]
	c.routePattern = ""
	clear(c.values)
}

func (c *Ctx) Param(name string) string {
	for i := range c.params {
		if c.params[i].key == name {
			return c.params[i].value
		}
	}
	return ""
}

func (c *Ctx) Query(name string) string {
	return c.Request.URL.Query().Get(name)
}

func (c *Ctx) Header(name string) string {
	return c.Request.Header.Get(name)
}

func (c *Ctx) RoutePattern() string {
	return c.routePattern
}

func (c *Ctx) Set(key string, value any) {
	if c.values == nil {
		c.values = make(map[string]any, 4)
	}
	c.values[key] = value
}

func (c *Ctx) Get(key string) (any, bool) {
	v, ok := c.values[key]
	return v, ok
}

func (c *Ctx) BindJSON(dst any) error {
	dec := json.NewDecoder(c.Request.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return ErrBadRequest("invalid JSON body", err.Error())
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return ErrBadRequest("invalid JSON body", "multiple JSON values are not allowed")
	}
	return nil
}

func (c *Ctx) Validate(v any) error {
	if c.app == nil || c.app.validator == nil {
		return nil
	}
	if err := c.app.validator.Validate(v); err != nil {
		return ErrBadRequest("validation failed", err.Error())
	}
	return nil
}

func (c *Ctx) JSON(status int, body any) error {
	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.WriteHeader(status)
	return json.NewEncoder(c.Writer).Encode(body)
}

func (c *Ctx) Text(status int, msg string) error {
	c.Writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
	c.Writer.WriteHeader(status)
	_, err := c.Writer.Write([]byte(msg))
	return err
}
