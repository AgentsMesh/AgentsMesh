package interfaces

import (
	"github.com/anthropics/agentsmesh/backend/internal/infra/pki"
)

type PKICertificateIssuer interface {
	IssueRunnerCertificate(nodeID, orgSlug string) (*pki.CertificateInfo, error)

	CACertPEM() []byte
}
