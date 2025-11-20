package opensearch_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	opensearchClient "github.com/sonuudigital/microservices/search-service/internal/opensearch"
)

const unexpectedErrFmt = "unexpected error: %v"

func TestNewClientValidation(t *testing.T) {
	_, err := opensearchClient.NewClient([]string{}, "u", "p")
	if err == nil {
		t.Fatalf("expected error for empty addresses")
	}
	_, err = opensearchClient.NewClient([]string{"http://x"}, "", "p")
	if err == nil {
		t.Fatalf("expected error for empty username")
	}
	_, err = opensearchClient.NewClient([]string{"http://x"}, "u", "")
	if err == nil {
		t.Fatalf("expected error for empty password")
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }))
	defer srv.Close()
	c, err := opensearchClient.NewClient([]string{srv.URL}, "u", "p")
	if err != nil {
		t.Fatalf(unexpectedErrFmt, err)
	}
	if c == nil {
		t.Fatalf("expected client instance")
	}
}

func TestClientIndexSuccess(t *testing.T) {
	var receivedBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		receivedBody = b
		w.WriteHeader(201)
		w.Write([]byte(`{"result":"created"}`))
	}))
	defer srv.Close()
	c, err := opensearchClient.NewClient([]string{srv.URL}, "u", "p")
	if err != nil {
		t.Fatalf(unexpectedErrFmt, err)
	}
	body := []byte(`{"name":"test"}`)
	res, err := c.Index(context.Background(), "products", "id123", body)
	if err != nil {
		t.Fatalf("unexpected index error: %v", err)
	}
	if res == nil {
		t.Fatalf("expected response")
	}
	if res.StatusCode != 201 {
		t.Fatalf("expected 201 got %d", res.StatusCode)
	}
	if !bytes.Equal(body, receivedBody) {
		t.Fatalf("body mismatch")
	}
}

func TestClientIndexServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(500) }))
	defer srv.Close()
	c, err := opensearchClient.NewClient([]string{srv.URL}, "u", "p")
	if err != nil {
		t.Fatalf(unexpectedErrFmt, err)
	}
	res, err := c.Index(context.Background(), "products", "id123", []byte(`{"x":1}`))
	if err != nil {
		t.Fatalf("did not expect transport error: %v", err)
	}
	if res.StatusCode != 500 {
		t.Fatalf("expected 500 got %d", res.StatusCode)
	}
}

func TestClientIndexNetworkError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }))
	url := srv.URL
	srv.Close()
	c, err := opensearchClient.NewClient([]string{url}, "u", "p")
	if err != nil {
		t.Fatalf(unexpectedErrFmt, err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	_, err = c.Index(ctx, "products", "id123", []byte(`{"x":1}`))
	if err == nil {
		t.Fatalf("expected network error")
	}
}
