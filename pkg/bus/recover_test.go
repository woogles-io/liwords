package bus

import (
	"context"
	"errors"
	"testing"
)

func TestRecoverToError(t *testing.T) {
	ctx := context.Background()

	if err := recoverToError(ctx, func() error { return nil }); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}

	want := errors.New("boom")
	if err := recoverToError(ctx, func() error { return want }); err != want {
		t.Fatalf("expected %v, got %v", want, err)
	}

	err := recoverToError(ctx, func() error {
		var p *struct{ x int }
		_ = p.x // nil pointer dereference
		return nil
	})
	if err == nil {
		t.Fatal("expected an error from a recovered panic")
	}
}
