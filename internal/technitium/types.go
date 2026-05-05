package technitium

import "time"

// Zone represents a DNS zone in Technitium
type Zone struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Internal bool   `json:"internal"`
	Disabled bool   `json:"disabled"`
}

// Record represents a DNS record in Technitium
type Record struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	TTL      int    `json:"ttl"`
	RData    RData  `json:"rData"`
	Disabled bool   `json:"disabled,omitempty"`
	Comments string `json:"comments,omitempty"`
}

// RData represents the resource data for different record types
type RData struct {
	// Common fields
	Value string `json:"value,omitempty"`

	// A/AAAA records
	IPAddress string `json:"ipAddress,omitempty"`

	// CNAME records
	CName string `json:"cname,omitempty"`

	// TXT records
	Text string `json:"text,omitempty"`

	// MX records
	Preference int    `json:"preference,omitempty"`
	Exchange   string `json:"exchange,omitempty"`

	// SRV records
	Priority int    `json:"priority,omitempty"`
	Weight   int    `json:"weight,omitempty"`
	Port     int    `json:"port,omitempty"`
	Target   string `json:"target,omitempty"`
}

// ZonesResponse represents the response from the zones API
type ZonesResponse struct {
	Status string `json:"status"`
	Zones  []Zone `json:"zones"`
}

// RecordsResponse represents the response from the records API
type RecordsResponse struct {
	Status  string   `json:"status"`
	Records []Record `json:"records"`
}

// APIResponse represents a generic API response
type APIResponse struct {
	Status   string `json:"status"`
	ErrorMsg string `json:"errorMessage,omitempty"`
}

// LoginResponse represents the response from the login API
type LoginResponse struct {
	Status string `json:"status"`
	Token  string `json:"token"`
}

// ZoneInfo contains zone-related information
type ZoneInfo struct {
	Name         string    `json:"name"`
	Type         string    `json:"type"`
	Internal     bool      `json:"internal"`
	Disabled     bool      `json:"disabled"`
	Expiry       time.Time `json:"expiry,omitempty"`
	LastModified time.Time `json:"lastModified,omitempty"`
}
