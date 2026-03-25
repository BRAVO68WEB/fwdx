package server

import (
	"context"
	"testing"
	"time"
)

func TestStore_UpsertTunnelStatus_AndList(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	err = store.UpsertTunnelStatus(context.Background(), "app.tunnel.example.com", "http://localhost:3000", "running", "running", "", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	list, err := store.ListTunnels(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 tunnel, got %d", len(list))
	}
	if list[0].Hostname != "app.tunnel.example.com" {
		t.Fatalf("unexpected hostname: %+v", list[0])
	}
}

func TestStore_InsertAndReadRequestLogs(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	ctx := context.Background()
	if err := store.UpsertTunnelStatus(ctx, "app.tunnel.example.com", "http://localhost:3000", "running", "running", "", time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := store.InsertRequestLog(ctx, RequestLogRecord{
		Hostname:  "app.tunnel.example.com",
		Timestamp: time.Now(),
		Method:    "GET",
		Host:      "app.tunnel.example.com",
		Path:      "/",
		Status:    200,
		LatencyMS: 15,
		BytesIn:   10,
		BytesOut:  42,
		ClientIP:  "127.0.0.1",
	}); err != nil {
		t.Fatal(err)
	}
	logs, err := store.ListRequestLogs(ctx, "app.tunnel.example.com", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 log row, got %d", len(logs))
	}
	if logs[0].Status != 200 || logs[0].Path != "/" {
		t.Fatalf("unexpected log row: %+v", logs[0])
	}
}

func TestStore_CreateTunnel_DefaultPublicAccessRule(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	tun, err := store.CreateTunnel(context.Background(), 1, "app", "app.tunnel.example.com", "http://localhost:3000", 0)
	if err != nil {
		t.Fatal(err)
	}
	rule, err := store.GetTunnelAccessRule(context.Background(), tun.ID)
	if err != nil {
		t.Fatal(err)
	}
	if rule.AuthMode != "public" {
		t.Fatalf("auth mode=%q want public", rule.AuthMode)
	}
	if len(rule.AllowedIPs) != 0 {
		t.Fatalf("allowed ips=%v want empty", rule.AllowedIPs)
	}
}

func TestStore_UpsertTunnelAccessRule_PreservesSecrets(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	tun, err := store.CreateTunnel(context.Background(), 1, "app", "app.tunnel.example.com", "http://localhost:3000", 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.UpsertTunnelAccessRule(context.Background(), tun.ID, AccessRuleInput{
		AuthMode:          "basic_auth",
		BasicAuthUsername: "demo",
		BasicAuthPassword: "secret-1",
	}); err != nil {
		t.Fatal(err)
	}
	first, err := store.GetTunnelAccessRule(context.Background(), tun.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.UpsertTunnelAccessRule(context.Background(), tun.ID, AccessRuleInput{
		AuthMode:          "basic_auth",
		BasicAuthUsername: "demo2",
	}); err != nil {
		t.Fatal(err)
	}
	second, err := store.GetTunnelAccessRule(context.Background(), tun.ID)
	if err != nil {
		t.Fatal(err)
	}
	if second.BasicAuthPasswordHash == "" || second.BasicAuthPasswordHash != first.BasicAuthPasswordHash {
		t.Fatalf("password hash not preserved: first=%q second=%q", first.BasicAuthPasswordHash, second.BasicAuthPasswordHash)
	}
	if second.BasicAuthUsername != "demo2" {
		t.Fatalf("username=%q want demo2", second.BasicAuthUsername)
	}
}
