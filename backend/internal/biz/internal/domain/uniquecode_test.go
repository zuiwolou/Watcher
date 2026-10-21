package domain

import (
	"testing"

	"watcher/backend/internal/biz/internal/model"
)

func TestCanTransit(t *testing.T) {
	cases := []struct {
		from, to string
		ok       bool
	}{
		{model.CodeIssued, model.CodeInStock, true},
		{model.CodeInStock, model.CodeLocked, true},
		{model.CodeLocked, model.CodeSigned, true},
		{model.CodeInTransit, model.CodeSigned, true},
		{model.CodeSigned, model.CodeInStock, false},
		{model.CodeIssued, model.CodeSigned, false},
	}
	for _, tc := range cases {
		err := CanTransit(tc.from, tc.to)
		if tc.ok && err != nil {
			t.Fatalf("%s -> %s want ok, got %v", tc.from, tc.to, err)
		}
		if !tc.ok && err == nil {
			t.Fatalf("%s -> %s want error", tc.from, tc.to)
		}
	}
}
