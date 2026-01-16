package email

import (
	"context"
	"fmt"

	"github.com/resend/resend-go/v2"
)

// Service defines the email service interface
type Service interface {
	// SendVerificationEmail sends an email verification link
	SendVerificationEmail(ctx context.Context, to, token string) error

	// SendPasswordResetEmail sends a password reset link
	SendPasswordResetEmail(ctx context.Context, to, token string) error

	// SendOrgInvitationEmail sends an organization invitation
	SendOrgInvitationEmail(ctx context.Context, to, orgName, inviterName, token string) error
}

// Config holds email service configuration
type Config struct {
	Provider    string // "resend" or "console" (for development)
	ResendKey   string
	FromAddress string // e.g., "AgentsMesh <noreply@agentsmesh.dev>"
	BaseURL     string // Frontend base URL for links, e.g., "https://agentsmesh.dev"
}

// NewService creates a new email service based on configuration
func NewService(cfg Config) Service {
	if cfg.Provider == "console" || cfg.ResendKey == "" {
		return &ConsoleService{
			baseURL: cfg.BaseURL,
		}
	}
	return &ResendService{
		client:      resend.NewClient(cfg.ResendKey),
		fromAddress: cfg.FromAddress,
		baseURL:     cfg.BaseURL,
	}
}

// ResendService implements email sending via Resend
type ResendService struct {
	client      *resend.Client
	fromAddress string
	baseURL     string
}

// SendVerificationEmail sends email verification via Resend
func (s *ResendService) SendVerificationEmail(ctx context.Context, to, token string) error {
	verifyURL := fmt.Sprintf("%s/verify-email/callback?token=%s", s.baseURL, token)

	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <title>Verify your email</title>
</head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">
    <h1 style="color: #333;">Welcome to AgentsMesh!</h1>
    <p style="color: #666; font-size: 16px;">Please verify your email address by clicking the button below:</p>
    <p style="margin: 30px 0;">
        <a href="%s" style="background-color: #0070f3; color: white; padding: 12px 24px; text-decoration: none; border-radius: 6px; font-weight: 500;">Verify Email</a>
    </p>
    <p style="color: #999; font-size: 14px;">Or copy this link: <a href="%s" style="color: #0070f3;">%s</a></p>
    <p style="color: #999; font-size: 14px;">This link will expire in 24 hours.</p>
    <hr style="border: none; border-top: 1px solid #eee; margin: 30px 0;">
    <p style="color: #999; font-size: 12px;">If you didn't create an account, you can safely ignore this email.</p>
</body>
</html>
`, verifyURL, verifyURL, verifyURL)

	_, err := s.client.Emails.SendWithContext(ctx, &resend.SendEmailRequest{
		From:    s.fromAddress,
		To:      []string{to},
		Subject: "Verify your email - AgentsMesh",
		Html:    html,
	})
	return err
}

// SendPasswordResetEmail sends password reset email via Resend
func (s *ResendService) SendPasswordResetEmail(ctx context.Context, to, token string) error {
	resetURL := fmt.Sprintf("%s/reset-password?token=%s", s.baseURL, token)

	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <title>Reset your password</title>
</head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">
    <h1 style="color: #333;">Reset your password</h1>
    <p style="color: #666; font-size: 16px;">We received a request to reset your password. Click the button below to proceed:</p>
    <p style="margin: 30px 0;">
        <a href="%s" style="background-color: #0070f3; color: white; padding: 12px 24px; text-decoration: none; border-radius: 6px; font-weight: 500;">Reset Password</a>
    </p>
    <p style="color: #999; font-size: 14px;">Or copy this link: <a href="%s" style="color: #0070f3;">%s</a></p>
    <p style="color: #999; font-size: 14px;">This link will expire in 1 hour.</p>
    <hr style="border: none; border-top: 1px solid #eee; margin: 30px 0;">
    <p style="color: #999; font-size: 12px;">If you didn't request a password reset, you can safely ignore this email.</p>
</body>
</html>
`, resetURL, resetURL, resetURL)

	_, err := s.client.Emails.SendWithContext(ctx, &resend.SendEmailRequest{
		From:    s.fromAddress,
		To:      []string{to},
		Subject: "Reset your password - AgentsMesh",
		Html:    html,
	})
	return err
}

// SendOrgInvitationEmail sends organization invitation via Resend
func (s *ResendService) SendOrgInvitationEmail(ctx context.Context, to, orgName, inviterName, token string) error {
	inviteURL := fmt.Sprintf("%s/invite/%s", s.baseURL, token)

	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <title>You've been invited to join %s</title>
</head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">
    <h1 style="color: #333;">You're invited!</h1>
    <p style="color: #666; font-size: 16px;"><strong>%s</strong> has invited you to join <strong>%s</strong> on AgentsMesh.</p>
    <p style="margin: 30px 0;">
        <a href="%s" style="background-color: #0070f3; color: white; padding: 12px 24px; text-decoration: none; border-radius: 6px; font-weight: 500;">Accept Invitation</a>
    </p>
    <p style="color: #999; font-size: 14px;">Or copy this link: <a href="%s" style="color: #0070f3;">%s</a></p>
    <p style="color: #999; font-size: 14px;">This invitation will expire in 7 days.</p>
    <hr style="border: none; border-top: 1px solid #eee; margin: 30px 0;">
    <p style="color: #999; font-size: 12px;">If you don't want to join, you can safely ignore this email.</p>
</body>
</html>
`, orgName, inviterName, orgName, inviteURL, inviteURL, inviteURL)

	_, err := s.client.Emails.SendWithContext(ctx, &resend.SendEmailRequest{
		From:    s.fromAddress,
		To:      []string{to},
		Subject: fmt.Sprintf("You've been invited to join %s - AgentsMesh", orgName),
		Html:    html,
	})
	return err
}

// ConsoleService implements email service for development (prints to console)
type ConsoleService struct {
	baseURL string
}

// SendVerificationEmail prints verification email to console
func (s *ConsoleService) SendVerificationEmail(ctx context.Context, to, token string) error {
	verifyURL := fmt.Sprintf("%s/verify-email/callback?token=%s", s.baseURL, token)
	fmt.Printf("[EMAIL] Verification email to: %s\n", to)
	fmt.Printf("[EMAIL] Verify URL: %s\n", verifyURL)
	return nil
}

// SendPasswordResetEmail prints password reset email to console
func (s *ConsoleService) SendPasswordResetEmail(ctx context.Context, to, token string) error {
	resetURL := fmt.Sprintf("%s/reset-password?token=%s", s.baseURL, token)
	fmt.Printf("[EMAIL] Password reset email to: %s\n", to)
	fmt.Printf("[EMAIL] Reset URL: %s\n", resetURL)
	return nil
}

// SendOrgInvitationEmail prints organization invitation to console
func (s *ConsoleService) SendOrgInvitationEmail(ctx context.Context, to, orgName, inviterName, token string) error {
	inviteURL := fmt.Sprintf("%s/invite/%s", s.baseURL, token)
	fmt.Printf("[EMAIL] Organization invitation to: %s\n", to)
	fmt.Printf("[EMAIL] Org: %s, Inviter: %s\n", orgName, inviterName)
	fmt.Printf("[EMAIL] Invite URL: %s\n", inviteURL)
	return nil
}
