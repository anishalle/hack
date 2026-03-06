package models

import "time"

type Project struct {
	Name         string   `json:"name" firestore:"name"`
	Description  string   `json:"description,omitempty" firestore:"description,omitempty"`
	GCPProject   string   `json:"gcp_project" firestore:"gcp_project"`
	Environments []string `json:"environments" firestore:"environments"`
	CreatedBy    string   `json:"created_by" firestore:"created_by"`
	CreatedAt    time.Time `json:"created_at" firestore:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" firestore:"updated_at"`
}
