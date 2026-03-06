package api

import "fmt"

type APIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error %d: %s", e.Code, e.Message)
}

type DeviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURL string `json:"verification_url"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	Email        string `json:"email"`
	TokenType    string `json:"token_type"`
}

type EnvVars map[string]string

type EnvPullResponse struct {
	Environment string  `json:"environment"`
	Variables   EnvVars `json:"variables"`
	Version     string  `json:"version"`
}

type EnvPushRequest struct {
	Environment string  `json:"environment"`
	Variables   EnvVars `json:"variables"`
}

type EnvDiffResponse struct {
	Added   EnvVars `json:"added"`
	Removed EnvVars `json:"removed"`
	Changed map[string][2]string `json:"changed"`
}

type ProjectInfo struct {
	Name         string   `json:"name"`
	Environments []string `json:"environments"`
	Role         string   `json:"role"`
}

type UserInfo struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

type AuditEntry struct {
	Timestamp string `json:"timestamp"`
	User      string `json:"user"`
	Action    string `json:"action"`
	Resource  string `json:"resource"`
	Details   string `json:"details,omitempty"`
}

type DeployStatus struct {
	Environment string `json:"environment"`
	Service     string `json:"service"`
	Revision    string `json:"revision"`
	Status      string `json:"status"`
	URL         string `json:"url,omitempty"`
	UpdatedAt   string `json:"updated_at"`
}
