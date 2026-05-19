package dns

import (
	"context"
	"fmt"
)

func (p *AliyunProvider) CreateTXTRecord(ctx context.Context, fqdn, value string) error {
	rr, domainName := p.parseSubdomain(fqdn)

	existing, err := p.getTXTRecordByRR(ctx, domainName, rr)
	if err != nil {
		return err
	}
	if existing != nil {
		return p.updateTXTRecordByID(ctx, existing.RecordID, rr, value)
	}

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

func (p *AliyunProvider) DeleteTXTRecord(ctx context.Context, fqdn string) error {
	rr, domainName := p.parseSubdomain(fqdn)

	record, err := p.getTXTRecordByRR(ctx, domainName, rr)
	if err != nil {
		return err
	}
	if record == nil {
		return nil
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

	for _, record := range resp.DomainRecords.Record {
		if record.RR == rr && record.Type == "TXT" {
			return &record, nil
		}
	}

	return nil, nil
}

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
