// Package interfaces defines shared interfaces for dependency injection.
// This package follows the Interface Segregation Principle (ISP) by providing
// focused interfaces that can be used across different packages.
package interfaces

import (
	"github.com/anthropics/agentsmesh/backend/internal/infra/pki"
)

// PKICertificateIssuer defines the interface for issuing Runner certificates.
// This is the minimal interface needed by runner.Service for certificate operations.
//
// Design decision: We use pki.CertificateInfo directly instead of duplicating
// the struct to avoid maintenance burden and ensure type compatibility.
type PKICertificateIssuer interface {
	// IssueRunnerCertificate issues a client certificate for a Runner.
	// The certificate CN contains the node_id and Organization contains the org_slug.
	IssueRunnerCertificate(nodeID, orgSlug string) (*pki.CertificateInfo, error)

	// CACertPEM returns the CA certificate in PEM format.
	// This is returned to Runners during registration for them to verify the server.
	CACertPEM() []byte
}
