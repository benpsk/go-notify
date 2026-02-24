package discord

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClient_RequiresWebhookURL(t *testing.T) {
	t.Parallel()

	_, err := NewClient("   ")
	if err == nil {
		t.Fatal("expected error for empty webhook url")
	}
}

func TestSendContent_PostsJSONPayload(t *testing.T) {
	t.Parallel()

	type payload struct {
		Content string `json:"content"`
	}

	var gotMethod string
	var gotContentType string
	var gotPayload payload

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotContentType = r.Header.Get("Content-Type")

		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&gotPayload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client, err := NewClient(srv.URL)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	if err := client.SendContent(context.Background(), " hello "); err != nil {
		t.Fatalf("send content: %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("got method %q want %q", gotMethod, http.MethodPost)
	}
	if gotContentType != "application/json" {
		t.Fatalf("got content type %q", gotContentType)
	}
	if gotPayload.Content != "hello" {
		t.Fatalf("got content %q want %q", gotPayload.Content, "hello")
	}
}

func TestSend_ReturnsErrorOnNon2xxStatus(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	client, err := NewClient(srv.URL)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	err = client.SendContent(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error for non-2xx status")
	}
}

func TestSend_RequiresContent(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client, err := NewClient(srv.URL)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	if err := client.SendContent(context.Background(), "   "); err == nil {
		t.Fatal("expected error for empty content")
	}
}
