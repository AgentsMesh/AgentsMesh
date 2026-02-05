// Package dns provides DNS management functionality.
// This file is deprecated - use aliyun_request.go, aliyun_a_record.go, and aliyun_txt_record.go instead.
// This file is kept for reference only and will be removed in a future version.
package dns

// Aliyun DNS provider functionality has been split into:
// - aliyun_request.go: Provider struct, request signing, API communication
// - aliyun_a_record.go: A record CRUD operations
// - aliyun_txt_record.go: TXT record CRUD operations (for ACME DNS-01 challenge)
