package domain

import (
	"fmt"

	"watcher/backend/pkg/errcode"
	"watcher/backend/internal/biz/internal/model"
)

// CanTransit returns nil if transition is allowed.
func CanTransit(from, to string) error {
	ok := map[string]map[string]bool{
		model.CodeIssued: {
			model.CodeInStock: true,
			model.CodeVoided:  true,
		},
		model.CodeInStock: {
			model.CodeLocked:    true,
			model.CodeInTransit: true,
			model.CodeVoided:    true,
		},
		model.CodeLocked: {
			model.CodeInTransit: true,
			model.CodeSigned:    true,
			model.CodeInStock:   true, // unlock cancel path
			model.CodeVoided:    true,
		},
		model.CodeInTransit: {
			model.CodeSigned: true,
			model.CodeVoided: true,
		},
	}
	if next, exists := ok[from]; exists && next[to] {
		return nil
	}
	return fmt.Errorf("%s: %s -> %s", errcode.StateInvalid, from, to)
}
