package tunnel

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestIsIdempotentMethod(t *testing.T) {
	if !IsIdempotentMethod(http.MethodGet) {
		t.Fatal("GET should be idempotent")
	}
	if IsIdempotentMethod(http.MethodPost) {
		t.Fatal("POST should not be idempotent")
	}
}

func TestProxyToLocal_ResponseTooLarge(t *testing.T) {
	_ = os.Setenv("FWDX_MAX_RESPONSE_BODY_BYTES", "8")
	defer os.Unsetenv("FWDX_MAX_RESPONSE_BODY_BYTES")

	local := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("123456789"))
	}))
	defer local.Close()

	req := &ProxyReq{
		ID:     "id1",
		Method: http.MethodGet,
		Path:   "/",
		Header: make(http.Header),
	}
	_, err := ProxyToLocal(local.URL, req)
	if err == nil {
		t.Fatal("expected response-too-large error")
	}
	if err.Error() == "" {
		t.Fatal("expected non-empty error")
	}
}
