// Package client provides communication with AgentsMesh server.
package client

// Error codes for pod operations
const (
	ErrCodeUnknown         = "UNKNOWN"
	ErrCodeSandboxCreate   = "SANDBOX_CREATE_FAILED"
	ErrCodeGitClone        = "GIT_CLONE_FAILED"
	ErrCodeGitWorktree     = "GIT_WORKTREE_FAILED"
	ErrCodeGitAuth         = "GIT_AUTH_FAILED"
	ErrCodeFileCreate      = "FILE_CREATE_FAILED"
	ErrCodeFilePermission  = "FILE_PERMISSION_DENIED"
	ErrCodeCommandNotFound = "COMMAND_NOT_FOUND"
	ErrCodeCommandStart    = "COMMAND_START_FAILED"
	ErrCodeWorkDirNotExist = "WORK_DIR_NOT_EXIST"
	ErrCodeDiskFull        = "DISK_FULL"
	ErrCodePrepareScript   = "PREPARE_SCRIPT_FAILED"
	ErrCodePTYError        = "PTY_READ_ERROR"
)

// PodError represents an error that occurred during pod operations.
type PodError struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
}

// Error implements the error interface.
func (e *PodError) Error() string {
	return e.Message
}

// NewPodError creates a new PodError with the given code and message.
func NewPodError(code, message string) *PodError {
	return &PodError{
		Code:    code,
		Message: message,
	}
}

// NewPodErrorWithDetails creates a new PodError with details.
func NewPodErrorWithDetails(code, message string, details map[string]string) *PodError {
	return &PodError{
		Code:    code,
		Message: message,
		Details: details,
	}
}

// CreatePodResponse represents the response to a create pod request.
type CreatePodResponse struct {
	Success bool      `json:"success"`
	PodKey  string    `json:"pod_key,omitempty"`
	Error   *PodError `json:"error,omitempty"`
}
