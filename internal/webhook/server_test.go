package webhook

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"sigs.k8s.io/external-dns/endpoint"
	"sigs.k8s.io/external-dns/plan"
)

type mockProvider struct {
	recordsFn         func(context.Context) ([]*endpoint.Endpoint, error)
	applyChangesFn    func(context.Context, *plan.Changes) error
	adjustEndpointsFn func([]*endpoint.Endpoint) ([]*endpoint.Endpoint, error)
	domainFilter      endpoint.DomainFilter
}

func (m *mockProvider) Records(ctx context.Context) ([]*endpoint.Endpoint, error) {
	if m.recordsFn != nil {
		return m.recordsFn(ctx)
	}
	return nil, nil
}

func (m *mockProvider) ApplyChanges(ctx context.Context, changes *plan.Changes) error {
	if m.applyChangesFn != nil {
		return m.applyChangesFn(ctx, changes)
	}
	return nil
}

func (m *mockProvider) AdjustEndpoints(endpoints []*endpoint.Endpoint) ([]*endpoint.Endpoint, error) {
	if m.adjustEndpointsFn != nil {
		return m.adjustEndpointsFn(endpoints)
	}
	return endpoints, nil
}

func (m *mockProvider) GetDomainFilter() endpoint.DomainFilter {
	return m.domainFilter
}

func TestHandleRootNegotiatesAcceptAndReturnsDomainFilter(t *testing.T) {
	p := &mockProvider{domainFilter: endpoint.NewDomainFilter([]string{"example.com"})}
	s := NewServer(p, ":0")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept", "application/external.dns.webhook+json;version=1")
	rec := httptest.NewRecorder()

	s.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/external.dns.webhook+json;version=1" {
		t.Fatalf("expected negotiated content-type, got %q", got)
	}
	if !strings.Contains(rec.Body.String(), "example.com") {
		t.Fatalf("expected body to contain domain filter, got %s", rec.Body.String())
	}
}

func TestHandleGetRecordsSuccess(t *testing.T) {
	p := &mockProvider{
		recordsFn: func(context.Context) ([]*endpoint.Endpoint, error) {
			return []*endpoint.Endpoint{endpoint.NewEndpoint("a.example.com", "A", "1.2.3.4")}, nil
		},
	}
	s := NewServer(p, ":0")

	req := httptest.NewRequest(http.MethodGet, "/records", nil)
	rec := httptest.NewRecorder()

	s.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "a.example.com") {
		t.Fatalf("expected response body to include record, got %s", rec.Body.String())
	}
}

func TestHandleGetRecordsError(t *testing.T) {
	p := &mockProvider{
		recordsFn: func(context.Context) ([]*endpoint.Endpoint, error) {
			return nil, errors.New("backend unavailable")
		},
	}
	s := NewServer(p, ":0")

	req := httptest.NewRequest(http.MethodGet, "/records", nil)
	rec := httptest.NewRecorder()

	s.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Failed to get records") {
		t.Fatalf("expected error response, got %s", rec.Body.String())
	}
}

func TestHandleApplyChangesInvalidJSON(t *testing.T) {
	p := &mockProvider{}
	s := NewServer(p, ":0")

	req := httptest.NewRequest(http.MethodPost, "/records", bytes.NewBufferString("not-json"))
	rec := httptest.NewRecorder()

	s.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

func TestHandleApplyChangesSuccess(t *testing.T) {
	called := false
	p := &mockProvider{
		applyChangesFn: func(_ context.Context, changes *plan.Changes) error {
			called = true
			if len(changes.Create) != 1 {
				t.Fatalf("expected one create endpoint, got %d", len(changes.Create))
			}
			return nil
		},
	}
	s := NewServer(p, ":0")

	body := `{"Create":[{"dnsName":"new.example.com","recordType":"A","targets":["1.2.3.4"],"recordTTL":300}]}`
	req := httptest.NewRequest(http.MethodPost, "/records", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	s.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", rec.Code)
	}
	if !called {
		t.Fatal("expected ApplyChanges to be called")
	}
}

func TestHandleAdjustEndpointsSuccess(t *testing.T) {
	p := &mockProvider{
		adjustEndpointsFn: func(eps []*endpoint.Endpoint) ([]*endpoint.Endpoint, error) {
			return eps, nil
		},
	}
	s := NewServer(p, ":0")

	body := `[{"dnsName":"adjust.example.com","recordType":"TXT","targets":["hello"],"recordTTL":60}]`
	req := httptest.NewRequest(http.MethodPost, "/adjustendpoints", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	s.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "adjust.example.com") {
		t.Fatalf("expected adjusted endpoint in response, got %s", rec.Body.String())
	}
}
