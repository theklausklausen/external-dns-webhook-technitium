package webhook

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/klausklausen/external-dns-webhook-technitium/internal/technitium"
	"sigs.k8s.io/external-dns/endpoint"
)

func TestIsSupportedRecordType(t *testing.T) {
	tests := []struct {
		name       string
		recordType string
		expected   bool
	}{
		{name: "supports A", recordType: "A", expected: true},
		{name: "supports AAAA", recordType: "AAAA", expected: true},
		{name: "supports CNAME", recordType: "CNAME", expected: true},
		{name: "supports TXT", recordType: "TXT", expected: true},
		{name: "does not support MX", recordType: "MX", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isSupportedRecordType(tt.recordType)
			if got != tt.expected {
				t.Fatalf("expected %v, got %v", tt.expected, got)
			}
		})
	}
}

func TestConvertToEndpoint(t *testing.T) {
	provider := &TechnitiumProvider{}

	tests := []struct {
		name   string
		record technitium.Record
		target string
	}{
		{
			name: "A record",
			record: technitium.Record{
				Name: "a.example.com",
				Type: "A",
				TTL:  120,
				RData: technitium.RData{
					IPAddress: "1.2.3.4",
				},
			},
			target: "1.2.3.4",
		},
		{
			name: "CNAME record",
			record: technitium.Record{
				Name: "www.example.com",
				Type: "CNAME",
				TTL:  300,
				RData: technitium.RData{
					CName: "example.com",
				},
			},
			target: "example.com",
		},
		{
			name: "TXT record",
			record: technitium.Record{
				Name: "txt.example.com",
				Type: "TXT",
				TTL:  60,
				RData: technitium.RData{
					Text: "hello",
				},
			},
			target: "hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ep := provider.convertToEndpoint(tt.record, "example.com")
			if ep == nil {
				t.Fatal("expected endpoint, got nil")
			}
			if ep.DNSName != tt.record.Name {
				t.Fatalf("expected DNS name %q, got %q", tt.record.Name, ep.DNSName)
			}
			if ep.RecordType != tt.record.Type {
				t.Fatalf("expected record type %q, got %q", tt.record.Type, ep.RecordType)
			}
			if len(ep.Targets) != 1 || ep.Targets[0] != tt.target {
				t.Fatalf("expected target %q, got %v", tt.target, ep.Targets)
			}
			if ep.RecordTTL != endpoint.TTL(tt.record.TTL) {
				t.Fatalf("expected TTL %d, got %d", tt.record.TTL, ep.RecordTTL)
			}
		})
	}
}

func TestConvertToEndpointReturnsNilForUnsupportedOrEmptyValue(t *testing.T) {
	provider := &TechnitiumProvider{}

	if ep := provider.convertToEndpoint(technitium.Record{Type: "MX"}, "example.com"); ep != nil {
		t.Fatalf("expected nil endpoint for unsupported type, got %+v", ep)
	}

	if ep := provider.convertToEndpoint(technitium.Record{Type: "A", RData: technitium.RData{}}, "example.com"); ep != nil {
		t.Fatalf("expected nil endpoint for empty target, got %+v", ep)
	}
}

func TestExtractZoneLongestMatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/zones/list" {
			http.NotFound(w, r)
			return
		}

		_ = json.NewEncoder(w).Encode(technitium.ZonesResponse{
			Status: "ok",
			Zones: []technitium.Zone{
				{Name: "example.com"},
				{Name: "dev.example.com"},
			},
		})
	}))
	defer server.Close()

	provider := &TechnitiumProvider{client: technitium.NewClient(server.URL, "secret")}

	zone := provider.extractZone("api.dev.example.com")
	if zone != "dev.example.com" {
		t.Fatalf("expected longest matching zone dev.example.com, got %q", zone)
	}
}

func TestExtractZoneFallsBackToLastTwoLabels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/zones/list" {
			http.NotFound(w, r)
			return
		}

		_ = json.NewEncoder(w).Encode(technitium.ZonesResponse{Status: "ok", Zones: []technitium.Zone{}})
	}))
	defer server.Close()

	provider := &TechnitiumProvider{client: technitium.NewClient(server.URL, "secret")}

	zone := provider.extractZone("app.internal.example.org")
	if zone != "example.org" {
		t.Fatalf("expected fallback zone example.org, got %q", zone)
	}
}