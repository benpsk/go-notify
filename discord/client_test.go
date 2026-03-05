package discord

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	notify "github.com/benpsk/go-notify"
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

func TestNewClientWithHTTPClient_DefaultsWhenNil(t *testing.T) {
	t.Parallel()

	client, err := NewClientWithHTTPClient("https://example.com/webhook", nil)
	if err != nil {
		t.Fatalf("new client with nil http client: %v", err)
	}
	if client.httpClient == nil {
		t.Fatal("expected default http client")
	}
}

func TestName(t *testing.T) {
	t.Parallel()

	client, err := NewClient("https://example.com/webhook")
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	if got := client.Name(); got != "discord" {
		t.Fatalf("got %q want %q", got, "discord")
	}
}

func TestNotify_MapsGenericMessageToDiscordContent(t *testing.T) {
	t.Parallel()

	type payload struct {
		Content string `json:"content"`
	}

	tests := []struct {
		name     string
		msg      notify.Message
		expected string
	}{
		{
			name: "subject and text",
			msg: notify.Message{
				Subject: "Deploy",
				Text:    "Succeeded",
			},
			expected: "Deploy\nSucceeded",
		},
		{
			name: "subject only",
			msg: notify.Message{
				Subject: "Deploy",
			},
			expected: "Deploy",
		},
		{
			name: "text only",
			msg: notify.Message{
				Text: "Succeeded",
			},
			expected: "Succeeded",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var got payload
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				defer r.Body.Close()
				if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
					t.Fatalf("decode payload: %v", err)
				}
				w.WriteHeader(http.StatusNoContent)
			}))
			defer srv.Close()

			client, err := NewClient(srv.URL)
			if err != nil {
				t.Fatalf("new client: %v", err)
			}

			if err := client.Notify(context.Background(), tt.msg); err != nil {
				t.Fatalf("notify: %v", err)
			}
			if got.Content != tt.expected {
				t.Fatalf("got content %q want %q", got.Content, tt.expected)
			}
		})
	}
}

func TestSend_NilClient(t *testing.T) {
	t.Parallel()

	var client *Client
	if err := client.Send(context.Background(), Message{Content: "x"}); err == nil {
		t.Fatal("expected error for nil client")
	}
}

func TestSend_ReturnsErrorWhenRequestBuildFails(t *testing.T) {
	t.Parallel()

	client, err := NewClient(":")
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	if err := client.SendContent(context.Background(), "x"); err == nil {
		t.Fatal("expected request build error for invalid url")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestSend_PropagatesHTTPClientError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("network down")
	httpClient := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return nil, wantErr
		}),
	}

	client, err := NewClientWithHTTPClient("https://example.com/webhook", httpClient)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	err = client.SendContent(context.Background(), "x")
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected wrapped network error, got %v", err)
	}
}

func TestSend_HandlesHTTPStatusBoundary(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(io.Discard, r.Body)
		_ = r.Body.Close()
		w.WriteHeader(http.StatusMultipleChoices)
	}))
	defer srv.Close()

	client, err := NewClient(srv.URL)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	if err := client.SendContent(context.Background(), "x"); err == nil {
		t.Fatal("expected error for 300 status")
	}
}

func TestSend_ReturnsErrorOnInformationalStatus(t *testing.T) {
	t.Parallel()

	httpClient := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusContinue,
				Status:     "100 Continue",
				Body:       io.NopCloser(strings.NewReader("")),
				Header:     make(http.Header),
			}, nil
		}),
	}

	client, err := NewClientWithHTTPClient("https://example.com/webhook", httpClient)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	if err := client.SendContent(context.Background(), "x"); err == nil {
		t.Fatal("expected error for informational status")
	}
}
