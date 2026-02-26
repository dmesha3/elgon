//go:build tools
// +build tools

// Package tools pins development tool dependencies in go.mod.
// Use with: go list -tags=tools -f '{{ join .Imports "\n" }}' .
package tools

import (
	_ "golang.org/x/tools/cmd/goimports"
)
