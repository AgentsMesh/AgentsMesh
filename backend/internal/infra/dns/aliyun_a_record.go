package dns

import (
	"context"
	"fmt"
)

// CreateRecord creates an A record
// subdomain should be the full domain name (e.g., "us-east-1.relay.agentsmesh.cn")
func (p *AliyunProvider) CreateRecord(ctx context.Context, subdomain, ip string) error {
	// Parse subdomain and domain
	rr, domainName := p.parseSubdomain(subdomain)

	// Check if record exists
	existing, err := p.getRecordByRR(ctx, domainName, rr)
	if err != nil {
		return err
	}
	if existing != nil {
		// Update existing record
		return p.updateRecordByID(ctx, existing.RecordID, rr, ip)
	}

	// Create new record
	params := map[string]string{
		"Action":     "AddDomainRecord",
		"DomainName": domainName,
		"RR":         rr,
		"Type":       "A",
		"Value":      ip,
		"TTL":        "600", // Aliyun minimum TTL is 600
	}

	resp, err := p.doRequest(ctx, params)
	if err != nil {
		return err
	}

	if resp.Code != "" {
		return fmt.Errorf("aliyun API error: %s - %s", resp.Code, resp.Message)
	}

	return nil
}

// DeleteRecord deletes an A record
func (p *AliyunProvider) DeleteRecord(ctx context.Context, subdomain string) error {
	rr, domainName := p.parseSubdomain(subdomain)

	record, err := p.getRecordByRR(ctx, domainName, rr)
	if err != nil {
		return err
	}
	if record == nil {
		return nil // Record doesn't exist
	}

	params := map[string]string{
		"Action":   "DeleteDomainRecord",
		"RecordId": record.RecordID,
	}

	resp, err := p.doRequest(ctx, params)
	if err != nil {
		return err
	}

	if resp.Code != "" {
		return fmt.Errorf("aliyun API error: %s - %s", resp.Code, resp.Message)
	}

	return nil
}

// GetRecord returns the IP for a subdomain
func (p *AliyunProvider) GetRecord(ctx context.Context, subdomain string) (string, error) {
	rr, domainName := p.parseSubdomain(subdomain)

	record, err := p.getRecordByRR(ctx, domainName, rr)
	if err != nil {
		return "", err
	}
	if record == nil {
		return "", nil
	}
	return record.Value, nil
}

// UpdateRecord updates an A record
func (p *AliyunProvider) UpdateRecord(ctx context.Context, subdomain, ip string) error {
	rr, domainName := p.parseSubdomain(subdomain)

	record, err := p.getRecordByRR(ctx, domainName, rr)
	if err != nil {
		return err
	}
	if record == nil {
		return p.CreateRecord(ctx, subdomain, ip)
	}

	return p.updateRecordByID(ctx, record.RecordID, rr, ip)
}

// getRecordByRR finds a record by its RR (subdomain part)
func (p *AliyunProvider) getRecordByRR(ctx context.Context, domainName, rr string) (*aliyunRecord, error) {
	params := map[string]string{
		"Action":      "DescribeDomainRecords",
		"DomainName":  domainName,
		"RRKeyWord":   rr,
		"TypeKeyWord": "A",
	}

	resp, err := p.doRequest(ctx, params)
	if err != nil {
		return nil, err
	}

	if resp.Code != "" {
		return nil, fmt.Errorf("aliyun API error: %s - %s", resp.Code, resp.Message)
	}

	if resp.DomainRecords == nil || len(resp.DomainRecords.Record) == 0 {
		return nil, nil
	}

	// Find exact match
	for _, record := range resp.DomainRecords.Record {
		if record.RR == rr && record.Type == "A" {
			return &record, nil
		}
	}

	return nil, nil
}

// updateRecordByID updates a record by its ID
func (p *AliyunProvider) updateRecordByID(ctx context.Context, recordID, rr, ip string) error {
	params := map[string]string{
		"Action":   "UpdateDomainRecord",
		"RecordId": recordID,
		"RR":       rr,
		"Type":     "A",
		"Value":    ip,
		"TTL":      "600", // Aliyun minimum TTL is 600
	}

	resp, err := p.doRequest(ctx, params)
	if err != nil {
		return err
	}

	if resp.Code != "" {
		return fmt.Errorf("aliyun API error: %s - %s", resp.Code, resp.Message)
	}

	return nil
}
