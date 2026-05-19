package dns

import (
	"context"
	"fmt"
)

func (p *AliyunProvider) CreateRecord(ctx context.Context, subdomain, ip string) error {
	rr, domainName := p.parseSubdomain(subdomain)

	existing, err := p.getRecordByRR(ctx, domainName, rr)
	if err != nil {
		return err
	}
	if existing != nil {
		return p.updateRecordByID(ctx, existing.RecordID, rr, ip)
	}

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

func (p *AliyunProvider) DeleteRecord(ctx context.Context, subdomain string) error {
	rr, domainName := p.parseSubdomain(subdomain)

	record, err := p.getRecordByRR(ctx, domainName, rr)
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

	for _, record := range resp.DomainRecords.Record {
		if record.RR == rr && record.Type == "A" {
			return &record, nil
		}
	}

	return nil, nil
}

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
