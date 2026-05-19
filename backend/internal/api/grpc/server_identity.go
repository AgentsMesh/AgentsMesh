package grpc

import (
	"context"
	"crypto/x509"
	"fmt"
	"strings"

	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

const (
	MetadataKeyClientCertDN = "x-client-cert-dn"

	MetadataKeyClientCertSerial = "x-client-cert-serial"

	MetadataKeyClientCertFingerprint = "x-client-cert-fingerprint"

	MetadataKeyOrgSlug = "x-org-slug"

	MetadataKeyRealIP = "x-real-ip"
)

type ClientIdentity struct {
	NodeID           string
	OrgSlug          string
	CertSerialNumber string
	CertFingerprint  string
	RealIP           string
}

// Prefers TLS peer certificate (direct mTLS); falls back to metadata (TLS-terminating proxy compat).
func ExtractClientIdentity(ctx context.Context) (*ClientIdentity, error) {
	if identity, err := extractFromTLSPeer(ctx); err == nil {
		return identity, nil
	}

	return extractFromMetadata(ctx)
}

func extractFromTLSPeer(ctx context.Context) (*ClientIdentity, error) {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("no peer in context")
	}

	tlsInfo, ok := p.AuthInfo.(credentials.TLSInfo)
	if !ok {
		return nil, fmt.Errorf("no TLS info in peer")
	}

	if len(tlsInfo.State.VerifiedChains) == 0 || len(tlsInfo.State.VerifiedChains[0]) == 0 {
		return nil, fmt.Errorf("no verified client certificate")
	}

	clientCert := tlsInfo.State.VerifiedChains[0][0]
	return extractFromCertificate(ctx, clientCert)
}

func extractFromCertificate(ctx context.Context, cert *x509.Certificate) (*ClientIdentity, error) {
	identity := &ClientIdentity{
		NodeID:           cert.Subject.CommonName,
		CertSerialNumber: cert.SerialNumber.String(),
	}

	if len(cert.Subject.Organization) > 0 {
		identity.OrgSlug = cert.Subject.Organization[0]
	}

	if identity.OrgSlug == "" {
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if values := md.Get(MetadataKeyOrgSlug); len(values) > 0 {
				identity.OrgSlug = values[0]
			}
		}
	}

	if identity.NodeID == "" {
		return nil, fmt.Errorf("missing client certificate CN (node_id)")
	}
	if identity.OrgSlug == "" {
		return nil, fmt.Errorf("missing org slug")
	}

	if p, ok := peer.FromContext(ctx); ok && p.Addr != nil {
		identity.RealIP = p.Addr.String()
	}

	return identity, nil
}

func extractFromMetadata(ctx context.Context) (*ClientIdentity, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, fmt.Errorf("no metadata in context")
	}

	identity := &ClientIdentity{}

	if values := md.Get(MetadataKeyClientCertDN); len(values) > 0 && values[0] != "" {
		identity.NodeID = extractCNFromDN(values[0])
	}
	if identity.NodeID == "" {
		return nil, fmt.Errorf("missing client certificate CN (node_id)")
	}

	if values := md.Get(MetadataKeyOrgSlug); len(values) > 0 {
		identity.OrgSlug = values[0]
	}
	if identity.OrgSlug == "" {
		return nil, fmt.Errorf("missing org slug")
	}

	if values := md.Get(MetadataKeyClientCertSerial); len(values) > 0 {
		identity.CertSerialNumber = values[0]
	}
	if values := md.Get(MetadataKeyClientCertFingerprint); len(values) > 0 {
		identity.CertFingerprint = values[0]
	}
	if values := md.Get(MetadataKeyRealIP); len(values) > 0 {
		identity.RealIP = values[0]
	}

	return identity, nil
}

func extractCNFromDN(dn string) string {
	if dn == "" {
		return ""
	}

	for _, part := range splitDN(dn) {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(strings.ToUpper(part), "CN=") {
			return strings.TrimPrefix(part, part[:3])
		}
	}

	return ""
}

func splitDN(dn string) []string {
	if strings.Contains(dn, "/") && !strings.Contains(dn, ",") {
		parts := strings.Split(dn, "/")
		result := make([]string, 0, len(parts))
		for _, p := range parts {
			if p != "" {
				result = append(result, p)
			}
		}
		return result
	}
	return strings.Split(dn, ",")
}
