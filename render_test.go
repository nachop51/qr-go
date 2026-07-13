package qr

import (
	"errors"
	"testing"

	"github.com/nachop51/qr-go/render/terminal"
)

func TestDefaultRendererIsTerminal(t *testing.T) {
	code, err := NewBinaryBuilder([]byte("hi")).Build()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := code.renderer.(terminal.Terminal); !ok {
		t.Fatalf("default renderer = %T, want terminal.Terminal", code.renderer)
	}
}

func TestRenderNoRenderer(t *testing.T) {
	_, err := NewBinaryBuilder([]byte("hi")).SetRenderer(nil).Build()
	if !errors.Is(err, ErrInvalidOptions) {
		t.Fatalf("want ErrInvalidOptions, got %v", err)
	}
}
