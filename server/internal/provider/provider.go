package provider

import "context"

type DeployProvider interface {
	Deploy(ctx context.Context, opts DeployOptions) (*DeployResult, error)
	Status(ctx context.Context, opts StatusOptions) (*ServiceStatus, error)
	Logs(ctx context.Context, opts LogsOptions) ([]LogEntry, error)
	Rollback(ctx context.Context, opts RollbackOptions) (*DeployResult, error)
	Restart(ctx context.Context, opts RestartOptions) error
}

type DeployOptions struct {
	Project    string
	Region     string
	Service    string
	Image      string
	Tag        string
	EnvVars    map[string]string
	Dockerfile string
}

type StatusOptions struct {
	Project string
	Region  string
	Service string
}

type LogsOptions struct {
	Project string
	Region  string
	Service string
	Limit   int
	Follow  bool
}

type RollbackOptions struct {
	Project  string
	Region   string
	Service  string
	Revision string
}

type RestartOptions struct {
	Project string
	Region  string
	Service string
}

type DeployResult struct {
	Service  string `json:"service"`
	Revision string `json:"revision"`
	URL      string `json:"url"`
	Status   string `json:"status"`
}

type ServiceStatus struct {
	Service       string `json:"service"`
	Status        string `json:"status"`
	URL           string `json:"url"`
	LatestRevision string `json:"latest_revision"`
	UpdatedAt     string `json:"updated_at"`
}

type LogEntry struct {
	Timestamp string `json:"timestamp"`
	Severity  string `json:"severity"`
	Message   string `json:"message"`
}

type DBProvider interface {
	Connect(ctx context.Context, opts DBConnectOptions) (*DBConnection, error)
	Status(ctx context.Context, opts DBStatusOptions) (*DBStatus, error)
	Branches(ctx context.Context, opts DBBranchOptions) ([]DBBranch, error)
}

type DBConnectOptions struct {
	ProjectID string
	Branch    string
	Database  string
}

type DBStatusOptions struct {
	ProjectID string
}

type DBBranchOptions struct {
	ProjectID string
}

type DBConnection struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Database string `json:"database"`
	User     string `json:"user"`
	Password string `json:"password"`
	URI      string `json:"uri"`
}

type DBStatus struct {
	Provider string `json:"provider"`
	Status   string `json:"status"`
	Region   string `json:"region"`
	Size     string `json:"size"`
}

type DBBranch struct {
	Name      string `json:"name"`
	Primary   bool   `json:"primary"`
	CreatedAt string `json:"created_at"`
}

type AuthProvider interface {
	ListUsers(ctx context.Context, opts AuthListOptions) ([]AuthUser, error)
	GetConfig(ctx context.Context) (map[string]any, error)
}

type AuthListOptions struct {
	Limit  int
	Search string
}

type AuthUser struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	CreatedAt string `json:"created_at"`
	LastLogin string `json:"last_login,omitempty"`
}
