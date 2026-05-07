package webhook

import (
	"context"
	"fmt"
	"strings"

	"github.com/klausklausen/external-dns-webhook-technitium/internal/technitium"
	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/external-dns/endpoint"
	"sigs.k8s.io/external-dns/plan"
	"sigs.k8s.io/external-dns/provider"
)

// TechnitiumProvider implements the external-dns provider interface
type TechnitiumProvider struct {
	client       *technitium.Client
	domainFilter endpoint.DomainFilter
	dryRun       bool
}

// NewTechnitiumProvider creates a new TechnitiumProvider
func NewTechnitiumProvider(client *technitium.Client, domainFilter endpoint.DomainFilter, dryRun bool) (*TechnitiumProvider, error) {
	return &TechnitiumProvider{
		client:       client,
		domainFilter: domainFilter,
		dryRun:       dryRun,
	}, nil
}

// Records returns the list of records in all managed zones
func (p *TechnitiumProvider) Records(ctx context.Context) ([]*endpoint.Endpoint, error) {
	log.Debug("Fetching DNS records from Technitium")

	zones, err := p.client.ListZones()
	if err != nil {
		return nil, fmt.Errorf("failed to list zones: %w", err)
	}

	var endpoints []*endpoint.Endpoint

	for _, zone := range zones {
		// Skip zones that don't match the domain filter
		if !p.domainFilter.Match(zone.Name) {
			log.Debugf("Skipping zone %s (doesn't match domain filter)", zone.Name)
			continue
		}

		// Skip internal and disabled zones
		if zone.Internal || zone.Disabled {
			log.Debugf("Skipping zone %s (internal or disabled)", zone.Name)
			continue
		}

		records, err := p.client.GetRecords(zone.Name)
		if err != nil {
			log.Warnf("Failed to get records for zone %s: %v", zone.Name, err)
			continue
		}

		for _, record := range records {
			// Skip disabled records
			if record.Disabled {
				continue
			}

			// Only process supported record types
			if !isSupportedRecordType(record.Type) {
				continue
			}

			ep := p.convertToEndpoint(record, zone.Name)
			if ep != nil {
				endpoints = append(endpoints, ep)
			}
		}
	}

	log.Infof("Found %d DNS records", len(endpoints))
	return endpoints, nil
}

// ApplyChanges applies the changes to DNS records
func (p *TechnitiumProvider) ApplyChanges(ctx context.Context, changes *plan.Changes) error {
	if p.dryRun {
		log.Info("DRY RUN: Would apply the following changes:")
		log.Infof("Create: %d records", len(changes.Create))
		log.Infof("UpdateOld: %d records", len(changes.UpdateOld))
		log.Infof("UpdateNew: %d records", len(changes.UpdateNew))
		log.Infof("Delete: %d records", len(changes.Delete))
		return nil
	}

	// Process deletions
	for _, ep := range changes.Delete {
		if err := p.deleteEndpoint(ep); err != nil {
			log.Errorf("Failed to delete endpoint %s: %v", ep.DNSName, err)
		}
	}

	// Process updates (delete old, create new)
	for i := range changes.UpdateOld {
		oldEp := changes.UpdateOld[i]
		newEp := changes.UpdateNew[i]

		if err := p.deleteEndpoint(oldEp); err != nil {
			log.Errorf("Failed to delete old endpoint %s: %v", oldEp.DNSName, err)
			continue
		}

		if err := p.createEndpoint(newEp); err != nil {
			log.Errorf("Failed to create new endpoint %s: %v", newEp.DNSName, err)
		}
	}

	// Process creations
	for _, ep := range changes.Create {
		if err := p.createEndpoint(ep); err != nil {
			log.Errorf("Failed to create endpoint %s: %v", ep.DNSName, err)
		}
	}

	return nil
}

// AdjustEndpoints modifies endpoints before they are processed
func (p *TechnitiumProvider) AdjustEndpoints(endpoints []*endpoint.Endpoint) ([]*endpoint.Endpoint, error) {
	// No adjustments needed for Technitium
	return endpoints, nil
}

// GetDomainFilter returns the domain filter
func (p *TechnitiumProvider) GetDomainFilter() endpoint.DomainFilter {
	return p.domainFilter
}

// convertToEndpoint converts a Technitium record to an external-dns endpoint
func (p *TechnitiumProvider) convertToEndpoint(record technitium.Record, zoneName string) *endpoint.Endpoint {
	var target string

	switch record.Type {
	case "A", "AAAA":
		target = record.RData.IPAddress
	case "CNAME":
		target = record.RData.CName
	case "TXT":
		target = record.RData.Text
	default:
		return nil
	}

	if target == "" {
		return nil
	}

	return &endpoint.Endpoint{
		DNSName:    record.Name,
		RecordType: record.Type,
		Targets:    endpoint.Targets{target},
		RecordTTL:  endpoint.TTL(record.TTL),
	}
}

// createEndpoint creates a new DNS record from an endpoint
func (p *TechnitiumProvider) createEndpoint(ep *endpoint.Endpoint) error {
	zone := p.extractZone(ep.DNSName)
	if zone == "" {
		return fmt.Errorf("could not determine zone for %s", ep.DNSName)
	}

	// Ensure zone exists
	if err := p.client.CreateZone(zone); err != nil {
		return fmt.Errorf("failed to ensure zone exists: %w", err)
	}

	// Create records for each target
	for _, target := range ep.Targets {
		ttl := int(ep.RecordTTL)
		if ttl == 0 {
			ttl = 300 // Default TTL
		}

		log.Debugf("Attempting to create record: zone=%s name=%s type=%s ttl=%d target=%s", zone, ep.DNSName, ep.RecordType, ttl, target)
		if err := p.client.AddRecord(zone, ep.DNSName, ep.RecordType, ttl, target); err != nil {
			// If record already exists, try to update it
			if strings.Contains(err.Error(), "already exists") {
				log.Infof("Record %s already exists, skipping", ep.DNSName)
				continue
			}
			log.Errorf("Failed to create record: zone=%s name=%s type=%s target=%s error=%v", zone, ep.DNSName, ep.RecordType, target, err)
			return err
		}
		log.Debugf("Successfully created record: zone=%s name=%s type=%s ttl=%d target=%s", zone, ep.DNSName, ep.RecordType, ttl, target)
	}

	return nil
}

// deleteEndpoint deletes a DNS record from an endpoint
func (p *TechnitiumProvider) deleteEndpoint(ep *endpoint.Endpoint) error {
	zone := p.extractZone(ep.DNSName)
	if zone == "" {
		return fmt.Errorf("could not determine zone for %s", ep.DNSName)
	}

	// Delete records for each target
	for _, target := range ep.Targets {
		if err := p.client.DeleteRecord(zone, ep.DNSName, ep.RecordType, target); err != nil {
			// Ignore "not found" errors
			if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "does not exist") {
				log.Infof("Record %s not found, skipping deletion", ep.DNSName)
				continue
			}
			return err
		}
	}

	return nil
}

// extractZone extracts the zone name from a DNS name
func (p *TechnitiumProvider) extractZone(dnsName string) string {
	zones, err := p.client.ListZones()
	if err != nil {
		log.Warnf("Failed to list zones: %v", err)
		return ""
	}

	// Find the longest matching zone
	var longestZone string
	for _, zone := range zones {
		if strings.HasSuffix(dnsName, zone.Name) {
			if len(zone.Name) > len(longestZone) {
				longestZone = zone.Name
			}
		}
		// Also check if the DNS name equals the zone name
		if dnsName == zone.Name {
			return zone.Name
		}
	}

	// If no zone found, extract from DNS name
	if longestZone == "" {
		parts := strings.Split(dnsName, ".")
		if len(parts) >= 2 {
			longestZone = strings.Join(parts[len(parts)-2:], ".")
		}
	}

	return longestZone
}

// isSupportedRecordType checks if a record type is supported
func isSupportedRecordType(recordType string) bool {
	supportedTypes := map[string]bool{
		"A":     true,
		"AAAA":  true,
		"CNAME": true,
		"TXT":   true,
	}
	return supportedTypes[recordType]
}

// Ensure TechnitiumProvider implements the provider.Provider interface
var _ provider.Provider = &TechnitiumProvider{}
