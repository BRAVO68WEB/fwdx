package tunnel

import (
	"context"
	"testing"
)

func TestConnect_InvalidURL(t *testing.T) {
	ctx := context.Background()
	err := Connect(ctx, "://invalid", "token", "app.example.com", "http://localhost:8080", false)
	if err == nil {
		t.Error("expected error for invalid URL")
	}
}
