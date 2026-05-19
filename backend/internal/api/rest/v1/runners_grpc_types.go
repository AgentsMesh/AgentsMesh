package v1

type RequestAuthURLRequest struct {
	MachineKey string            `json:"machine_key" binding:"required"`
	NodeID     string            `json:"node_id"`
	Labels     map[string]string `json:"labels"`
}

type AuthorizeRunnerRequest struct {
	AuthKey string `json:"auth_key" binding:"required"`
	NodeID  string `json:"node_id"`
}

type GenerateGRPCTokenRequest struct {
	Name      string            `json:"name"`
	Labels    map[string]string `json:"labels"`
	SingleUse bool              `json:"single_use"`
	MaxUses   int               `json:"max_uses"`
	ExpiresIn int               `json:"expires_in"` // seconds
}

type RegisterWithTokenRequest struct {
	Token  string `json:"token" binding:"required"`
	NodeID string `json:"node_id"`
}

type ReactivateRequest struct {
	Token string `json:"token" binding:"required"`
}
