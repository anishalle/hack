package models

import "time"

type User struct {
	ID        string            `json:"id" firestore:"id"`
	Email     string            `json:"email" firestore:"email"`
	Name      string            `json:"name,omitempty" firestore:"name,omitempty"`
	Projects  map[string]string `json:"projects" firestore:"projects"` // project name -> role
	CreatedAt time.Time         `json:"created_at" firestore:"created_at"`
	UpdatedAt time.Time         `json:"updated_at" firestore:"updated_at"`
}

func (u *User) HasProject(project string) bool {
	_, ok := u.Projects[project]
	return ok
}

func (u *User) RoleIn(project string) string {
	return u.Projects[project]
}
