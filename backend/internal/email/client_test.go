package email

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newTestClient(apiKey, endpoint string) *Client {
	return &Client{
		apiKey:     apiKey,
		from:       "noreply@test.local",
		fromName:   "TestBI",
		endpoint:   endpoint,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

func TestClient_Send_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("missing/wrong Authorization: %q", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("wrong Content-Type: %q", got)
		}
		if got := r.Header.Get("User-Agent"); got == "" {
			t.Error("missing User-Agent")
		}
		body, _ := io.ReadAll(r.Body)
		var req resendRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("bad json: %v", err)
		}
		if !strings.Contains(req.From, "TestBI") || !strings.Contains(req.From, "noreply@test.local") {
			t.Errorf("unexpected From: %s", req.From)
		}
		if len(req.To) != 1 || req.To[0] != "user@example.com" {
			t.Errorf("unexpected To: %v", req.To)
		}
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"id":"abc"}`))
	}))
	defer srv.Close()

	c := newTestClient("test-key", srv.URL)
	if err := c.Send(context.Background(), "user@example.com", "Hi", "<p>hi</p>"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClient_Send_400_Terminal(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		_, _ = w.Write([]byte(`{"error":"invalid from"}`))
	}))
	defer srv.Close()

	c := newTestClient("k", srv.URL)
	err := c.Send(context.Background(), "u@e.com", "s", "b")
	if err == nil {
		t.Fatal("expected error")
	}
	var se *SendError
	if !errors.As(err, &se) {
		t.Fatalf("expected *SendError, got %T", err)
	}
	if se.StatusCode != 400 || se.Retryable {
		t.Errorf("want 400 non-retryable, got %d retryable=%t", se.StatusCode, se.Retryable)
	}
	if IsRetryable(err) {
		t.Error("IsRetryable should be false for 400")
	}
}

func TestClient_Send_429_Retryable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(429)
		_, _ = w.Write([]byte(`{"error":"rate limited"}`))
	}))
	defer srv.Close()

	c := newTestClient("k", srv.URL)
	err := c.Send(context.Background(), "u@e.com", "s", "b")
	var se *SendError
	if !errors.As(err, &se) || se.StatusCode != 429 || !se.Retryable {
		t.Fatalf("want 429 retryable, got %v", err)
	}
	if !IsRetryable(err) {
		t.Error("IsRetryable should be true for 429")
	}
}

func TestClient_Send_500_Retryable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()

	c := newTestClient("k", srv.URL)
	err := c.Send(context.Background(), "u@e.com", "s", "b")
	if !IsRetryable(err) {
		t.Errorf("IsRetryable should be true for 500, got err=%v", err)
	}
}

func TestClient_DryRun_NoKey(t *testing.T) {
	c := NewClient("", "from@x", "Name")
	if !c.DryRun() {
		t.Fatal("empty apiKey should be DryRun")
	}
	// Send should be a no-op, no panic.
	if err := c.Send(context.Background(), "u@e.com", "s", "b"); err != nil {
		t.Errorf("dry-run Send should return nil, got %v", err)
	}
}

func TestClient_From_Format(t *testing.T) {
	c := NewClient("k", "noreply@foo", "FooBI")
	if got := c.From(); got != "FooBI <noreply@foo>" {
		t.Errorf("From()=%q", got)
	}
	// Empty fromName → NewClient defaults to "FoodBI".
	c2 := NewClient("k", "noreply@foo", "")
	if got := c2.From(); got != "FoodBI <noreply@foo>" {
		t.Errorf("From() default name=%q", got)
	}
	// Empty from → NewClient defaults to noreply@foodbi.local.
	c3 := NewClient("k", "", "FooBI")
	if got := c3.From(); got != "FooBI <noreply@foodbi.local>" {
		t.Errorf("From() default addr=%q", got)
	}
}
