package models

type Permission string

const (
	PermEnvRead  Permission = "env:read"
	PermEnvWrite Permission = "env:write"

	PermDeployRead     Permission = "deploy:read"
	PermDeployWrite    Permission = "deploy:write"
	PermDeployRestart  Permission = "deploy:restart"
	PermDeployRollback Permission = "deploy:rollback"

	PermDBRead    Permission = "db:read"
	PermDBWrite   Permission = "db:write"
	PermDBConnect Permission = "db:connect"

	PermAuthRead  Permission = "auth:read"
	PermAuthWrite Permission = "auth:write"

	PermAdminUsers Permission = "admin:users"
	PermAdminRoles Permission = "admin:roles"
	PermAdminAudit Permission = "admin:audit"
)

type Role struct {
	Name        string       `json:"name" firestore:"name"`
	Permissions []Permission `json:"permissions" firestore:"permissions"`
	BuiltIn     bool         `json:"built_in" firestore:"built_in"`
}

var DefaultRoles = map[string]Role{
	"viewer": {
		Name:    "viewer",
		BuiltIn: true,
		Permissions: []Permission{
			PermEnvRead, PermDeployRead, PermDBRead, PermAuthRead,
		},
	},
	"developer": {
		Name:    "developer",
		BuiltIn: true,
		Permissions: []Permission{
			PermEnvRead, PermDeployRead, PermDBRead, PermDBConnect, PermAuthRead,
		},
	},
	"deployer": {
		Name:    "deployer",
		BuiltIn: true,
		Permissions: []Permission{
			PermEnvRead, PermDeployRead, PermDeployWrite, PermDeployRestart,
			PermDeployRollback, PermDBRead, PermDBConnect, PermAuthRead,
		},
	},
	"admin": {
		Name:    "admin",
		BuiltIn: true,
		Permissions: []Permission{
			PermEnvRead, PermEnvWrite,
			PermDeployRead, PermDeployWrite, PermDeployRestart, PermDeployRollback,
			PermDBRead, PermDBWrite, PermDBConnect,
			PermAuthRead, PermAuthWrite,
			PermAdminUsers, PermAdminRoles, PermAdminAudit,
		},
	},
	"owner": {
		Name:    "owner",
		BuiltIn: true,
		Permissions: []Permission{
			PermEnvRead, PermEnvWrite,
			PermDeployRead, PermDeployWrite, PermDeployRestart, PermDeployRollback,
			PermDBRead, PermDBWrite, PermDBConnect,
			PermAuthRead, PermAuthWrite,
			PermAdminUsers, PermAdminRoles, PermAdminAudit,
		},
	},
}

func HasPermission(role string, perm Permission) bool {
	r, ok := DefaultRoles[role]
	if !ok {
		return false
	}
	for _, p := range r.Permissions {
		if p == perm {
			return true
		}
	}
	return false
}
