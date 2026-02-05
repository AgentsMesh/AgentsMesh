package dns

import (
	"context"
	"fmt"
)

// CreateTXTRecord creates a TXT record for ACME DNS-01 challenge
func (p *AliyunProvider) CreateTXTRecord(ctx context.Context, fqdn, value string) error {
	// Parse fqdn to get RR and domain
	rr, domainName := p.parseSubdomain(fqdn)

	// Check if record exists
	existing, err := p.getTXTRecordByRR(ctx, domainName, rr)
	if err != nil {
		return err
	}
	if existing != nil {
		// Update existing record
		return p.updateTXTRecordByID(ctx, existing.RecordID, rr, value)
	}

	// Create new record
	params := map[string]string{
		"Action":     "AddDomainRecord",
		"DomainName": domainName,
		"RR":         rr,
		"Type":       "TXT",
		"Value":      value,
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

// DeleteTXTRecord deletes a TXT record
func (p *AliyunProvider) DeleteTXTRecord(ctx context.Context, fqdn string) error {
	rr, domainName := p.parseSubdomain(fqdn)

	record, err := p.getTXTRecordByRR(ctx, domainName, rr)
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

// getTXTRecordByRR finds a TXT record by its RR
func (p *AliyunProvider) getTXTRecordByRR(ctx context.Context, domainName, rr string) (*aliyunRecord, error) {
	params := map[string]string{
		"Action":      "DescribeDomainRecords",
		"DomainName":  domainName,
		"RRKeyWord":   rr,
		"TypeKeyWord": "TXT",
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
		if record.RR == rr && record.Type == "TXT" {
			return &record, nil
		}
	}

	return nil, nil
}

// updateTXTRecordByID updates a TXT record by its ID
func (p *AliyunProvider) updateTXTRecordByID(ctx context.Context, recordID, rr, value string) error {
	params := map[string]string{
		"Action":   "UpdateDomainRecord",
		"RecordId": recordID,
		"RR":       rr,
		"Type":     "TXT",
		"Value":    value,
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
