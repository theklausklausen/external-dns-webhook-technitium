package technitium

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestNewClientTrimsTrailingSlash(t *testing.T) {
	client := NewClient("http://example.com/", "token")

	if client.baseURL != "http://example.com" {
		t.Fatalf("expected trimmed base URL, got %q", client.baseURL)
	}
}

func TestDoRequestGetAddsTokenAndParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}

		q := r.URL.Query()
		if got := q.Get("token"); got != "secret" {
			t.Fatalf("expected token query param, got %q", got)
		}
		if got := q.Get("domain"); got != "example.com" {
			t.Fatalf("expected domain query param, got %q", got)
		}

		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "secret")
	params := url.Values{}
	params.Set("domain", "example.com")

	if _, err := client.doRequest(http.MethodGet, "/api/test", params); err != nil {
		t.Fatalf("doRequest failed: %v", err)
	}
}

func TestZoneExistsCaseInsensitive(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/zones/list" {
			http.NotFound(w, r)
			return
		}

		resp := ZonesResponse{
			Status: "ok",
			Zones:  []Zone{{Name: "Example.COM"}},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL, "secret")

	exists, err := client.ZoneExists("example.com")
	if err != nil {
		t.Fatalf("ZoneExists returned error: %v", err)
	}
	if !exists {
		t.Fatal("expected zone to exist")
	}
}

func TestAddRecordUnsupportedType(t *testing.T) {
	client := NewClient("http://example.com", "secret")

	err := client.AddRecord("example.com", "www.example.com", "MX", 300, "mail.example.com")
	if err == nil {
		t.Fatal("expected error for unsupported record type")
	}
	if !strings.Contains(err.Error(), "unsupported record type") {
		t.Fatalf("expected unsupported type error, got: %v", err)
	}
}

func TestCreateZoneAlreadyExistsResponseReturnsNil(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/zones/create" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"status":"error","errorMessage":"zone already exists"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "secret")

	if err := client.CreateZone("example.com"); err != nil {
		t.Fatalf("expected nil when zone already exists, got: %v", err)
	}
}
