package technitium

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

// Client represents a Technitium DNS server client
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// NewClient creates a new Technitium DNS client
func NewClient(baseURL, token string) *Client {
	return &Client{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		token:   token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewClientWithAuth creates a new client and authenticates with username/password
func NewClientWithAuth(baseURL, username, password string) (*Client, error) {
	client := &Client{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	// Authenticate
	data := url.Values{}
	data.Set("user", username)
	data.Set("pass", password)

	resp, err := client.httpClient.PostForm(client.baseURL+"/api/user/login", data)
	if err != nil {
		return nil, fmt.Errorf("login failed: %w", err)
	}
	defer resp.Body.Close()

	var loginResp LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return nil, fmt.Errorf("failed to decode login response: %w", err)
	}

	if loginResp.Status != "ok" {
		return nil, fmt.Errorf("login failed: invalid credentials")
	}

	client.token = loginResp.Token
	log.Infof("Successfully authenticated to Technitium DNS server")

	return client, nil
}

// doRequest performs an HTTP request with authentication
func (c *Client) doRequest(method, endpoint string, params url.Values) ([]byte, error) {
	// Add token to params
	if params == nil {
		params = url.Values{}
	}
	params.Set("token", c.token)

	var reqURL string
	var body io.Reader

	if method == http.MethodGet {
		reqURL = fmt.Sprintf("%s%s?%s", c.baseURL, endpoint, params.Encode())
	} else {
		reqURL = fmt.Sprintf("%s%s", c.baseURL, endpoint)
		body = bytes.NewBufferString(params.Encode())
	}

	req, err := http.NewRequest(method, reqURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(data))
	}

	return data, nil
}

// ListZones lists all DNS zones
func (c *Client) ListZones() ([]Zone, error) {
	data, err := c.doRequest(http.MethodGet, "/api/zones/list", nil)
	if err != nil {
		return nil, err
	}

	var resp ZonesResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("failed to decode zones response: %w", err)
	}

	if resp.Status != "ok" {
		return nil, fmt.Errorf("API returned status: %s", resp.Status)
	}

	return resp.Zones, nil
}

// GetRecords retrieves all records for a specific zone
func (c *Client) GetRecords(zone string) ([]Record, error) {
	params := url.Values{}
	params.Set("domain", zone)
	params.Set("listZone", "true")

	data, err := c.doRequest(http.MethodGet, "/api/zones/records/get", params)
	if err != nil {
		return nil, err
	}

	var resp RecordsResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("failed to decode records response: %w", err)
	}

	if resp.Status != "ok" {
		return nil, fmt.Errorf("API returned status: %s", resp.Status)
	}

	return resp.Records, nil
}

// CreateZone creates a new DNS zone
func (c *Client) CreateZone(zone string) error {
	params := url.Values{}
	params.Set("zone", zone)
	params.Set("type", "Primary")

	data, err := c.doRequest(http.MethodGet, "/api/zones/create", params)
	if err != nil {
		// Check if zone already exists (handle multiple message formats)
		if strings.Contains(err.Error(), "Zone already exists") {
			log.Infof("Zone %s already exists", zone)
			return nil
		}
		return fmt.Errorf("failed to create zone: %w", err)
	}

	var resp APIResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return fmt.Errorf("failed to decode create zone response: %w", err)
	}

	if resp.Status != "ok" {
		return fmt.Errorf("failed to create zone: %s", resp.ErrorMsg)
	}

	log.Infof("Successfully created zone: %s", zone)
	return nil
}

// AddRecord adds a new DNS record
func (c *Client) AddRecord(zone, name, recordType string, ttl int, value string) error {
	params := url.Values{}
	params.Set("domain", name)
	params.Set("zone", zone)
	params.Set("type", recordType)
	params.Set("ttl", fmt.Sprintf("%d", ttl))

	switch recordType {
	case "A", "AAAA":
		params.Set("ipAddress", value)
	case "CNAME":
		params.Set("cname", value)
	case "TXT":
		params.Set("text", value)
	default:
		return fmt.Errorf("unsupported record type: %s", recordType)
	}

	data, err := c.doRequest(http.MethodGet, "/api/zones/records/add", params)
	if err != nil {
		return fmt.Errorf("failed to add record: %w", err)
	}

	var resp APIResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return fmt.Errorf("failed to decode add record response: %w", err)
	}

	if resp.Status != "ok" {
		return fmt.Errorf("failed to add record: %s", resp.ErrorMsg)
	}

	log.Infof("Successfully added record: %s %s %s", name, recordType, value)
	return nil
}

// UpdateRecord updates an existing DNS record
func (c *Client) UpdateRecord(zone, name, recordType, oldValue, newValue string, ttl int) error {
	// Delete old record
	if err := c.DeleteRecord(zone, name, recordType, oldValue); err != nil {
		return fmt.Errorf("failed to delete old record: %w", err)
	}

	// Add new record
	if err := c.AddRecord(zone, name, recordType, ttl, newValue); err != nil {
		return fmt.Errorf("failed to add new record: %w", err)
	}

	return nil
}

// DeleteRecord deletes a DNS record
func (c *Client) DeleteRecord(zone, name, recordType, value string) error {
	params := url.Values{}
	params.Set("domain", name)
	params.Set("zone", zone)
	params.Set("type", recordType)

	switch recordType {
	case "A", "AAAA":
		params.Set("ipAddress", value)
	case "CNAME":
		params.Set("cname", value)
	case "TXT":
		params.Set("text", value)
	}

	data, err := c.doRequest(http.MethodGet, "/api/zones/records/delete", params)
	if err != nil {
		return fmt.Errorf("failed to delete record: %w", err)
	}

	var resp APIResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return fmt.Errorf("failed to decode delete record response: %w", err)
	}

	if resp.Status != "ok" {
		return fmt.Errorf("failed to delete record: %s", resp.ErrorMsg)
	}

	log.Infof("Successfully deleted record: %s %s %s", name, recordType, value)
	return nil
}

// HealthCheck performs a health check on the Technitium server
func (c *Client) HealthCheck() error {
	_, err := c.ListZones()
	return err
}
