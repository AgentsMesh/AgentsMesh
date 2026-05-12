package authconnect

import (
	domainUser "github.com/anthropics/agentsmesh/backend/internal/domain/user"
	authv1 "github.com/anthropics/agentsmesh/proto/gen/go/auth/v1"
)

// toProtoUser converts the GORM-backed User entity to the protobuf wire
// shape. Field mapping mirrors the gin.H block REST handlers emit for
// login / register / verify-email — same five public fields plus
// is_email_verified, sourced from the same domain object.
//
// Passwords and other sensitive fields stay server-side; the proto User
// message has no field for them.
func toProtoUser(u *domainUser.User) *authv1.User {
	if u == nil {
		return nil
	}
	out := &authv1.User{
		Id:       u.ID,
		Email:    u.Email,
		Username: u.Username,
	}
	if u.Name != "" {
		n := u.Name
		out.Name = &n
	}
	if u.AvatarURL != "" {
		a := u.AvatarURL
		out.AvatarUrl = &a
	}
	verified := u.IsEmailVerified
	out.IsEmailVerified = &verified
	return out
}
