package dns

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// CreateTXTRecord creates a TXT record for ACME DNS-01 challenge
func (p *CloudflareProvider) CreateTXTRecord(ctx context.Context, fqdn, value string) error {
	// Check if record already exists
	existing, err := p.getTXTRecordID(ctx, fqdn)
	if err != nil {
		return err
	}
	if existing != "" {
		// Update existing record
		return p.updateTXTRecordByID(ctx, existing, value)
	}

	// Create new record
	payload := map[string]interface{}{
		"type":    "TXT",
		"name":    fqdn,
		"content": value,
		"ttl":     120, // Short TTL for ACME challenge
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	url := fmt.Sprintf("%s/zones/%s/dns_records", cloudflareAPIBase, p.zoneID)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := p.doRequest(req)
	if err != nil {
		return err
	}

	if !resp.Success {
		return fmt.Errorf("cloudflare API error: %v", resp.Errors)
	}

	return nil
}

// DeleteTXTRecord deletes a TXT record
func (p *CloudflareProvider) DeleteTXTRecord(ctx context.Context, fqdn string) error {
	recordID, err := p.getTXTRecordID(ctx, fqdn)
	if err != nil {
		return err
	}
	if recordID == "" {
		return nil // Record doesn't exist
	}

	url := fmt.Sprintf("%s/zones/%s/dns_records/%s", cloudflareAPIBase, p.zoneID, recordID)
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := p.doRequest(req)
	if err != nil {
		return err
	}

	if !resp.Success {
		return fmt.Errorf("cloudflare API error: %v", resp.Errors)
	}

	return nil
}

// getTXTRecordID returns the record ID for a TXT record
func (p *CloudflareProvider) getTXTRecordID(ctx context.Context, fqdn string) (string, error) {
	url := fmt.Sprintf("%s/zones/%s/dns_records?type=TXT&name=%s", cloudflareAPIBase, p.zoneID, fqdn)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	resp, err := p.doRequest(req)
	if err != nil {
		return "", err
	}

	if !resp.Success {
		return "", fmt.Errorf("cloudflare API error: %v", resp.Errors)
	}

	var records []cloudflareRecord
	if err := json.Unmarshal(resp.Result, &records); err != nil {
		return "", fmt.Errorf("unmarshal records: %w", err)
	}

	if len(records) == 0 {
		return "", nil
	}
	return records[0].ID, nil
}

// updateTXTRecordByID updates a TXT record by its ID
func (p *CloudflareProvider) updateTXTRecordByID(ctx context.Context, recordID, value string) error {
	payload := map[string]interface{}{
		"type":    "TXT",
		"content": value,
		"ttl":     120,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	url := fmt.Sprintf("%s/zones/%s/dns_records/%s", cloudflareAPIBase, p.zoneID, recordID)
	req, err := http.NewRequestWithContext(ctx, "PATCH", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := p.doRequest(req)
	if err != nil {
		return err
	}

	if !resp.Success {
		return fmt.Errorf("cloudflare API error: %v", resp.Errors)
	}

	return nil
}
