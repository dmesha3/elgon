package middleware

import "github.com/meshackkazimoto/elgon"

// CORSConfig controls CORS middleware behavior.
type CORSConfig struct {
	AllowedOrigins []string
	AllowedMethods []string
	AllowedHeaders []string
}

// CORS applies CORS headers. By default all origins are denied.
func CORS(cfg CORSConfig) elgon.Middleware {
	origins := make(map[string]struct{}, len(cfg.AllowedOrigins))
	for _, o := range cfg.AllowedOrigins {
		origins[o] = struct{}{}
	}
	methods := "GET,POST,PUT,PATCH,DELETE,OPTIONS"
	if len(cfg.AllowedMethods) > 0 {
		methods = joinCSV(cfg.AllowedMethods)
	}
	headers := "Authorization,Content-Type,X-Request-Id"
	if len(cfg.AllowedHeaders) > 0 {
		headers = joinCSV(cfg.AllowedHeaders)
	}

	return func(next elgon.HandlerFunc) elgon.HandlerFunc {
		return func(c *elgon.Ctx) error {
			origin := c.Header("Origin")
			if _, ok := origins[origin]; ok {
				c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
				c.Writer.Header().Set("Access-Control-Allow-Methods", methods)
				c.Writer.Header().Set("Access-Control-Allow-Headers", headers)
			}
			if c.Request.Method == "OPTIONS" {
				c.Writer.WriteHeader(204)
				return nil
			}
			return next(c)
		}
	}
}

func joinCSV(items []string) string {
	if len(items) == 0 {
		return ""
	}
	out := items[0]
	for _, item := range items[1:] {
		out += "," + item
	}
	return out
}
