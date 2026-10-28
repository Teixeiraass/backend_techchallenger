//go:build tools

// Package tools pins build-time tools so `make mock` works with the module
// versions in go.mod.
package tools

import (
	_ "github.com/golang/mock/mockgen/model"
)
