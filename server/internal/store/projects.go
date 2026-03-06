package store

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/anishalle/hack/internal/models"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ProjectStore struct {
	client     *firestore.Client
	collection string
}

func NewProjectStore(client *firestore.Client) *ProjectStore {
	return &ProjectStore{client: client, collection: "projects"}
}

func (s *ProjectStore) Get(ctx context.Context, name string) (*models.Project, error) {
	doc, err := s.client.Collection(s.collection).Doc(name).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, fmt.Errorf("project not found: %s", name)
		}
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	var project models.Project
	if err := doc.DataTo(&project); err != nil {
		return nil, fmt.Errorf("failed to parse project: %w", err)
	}
	return &project, nil
}

func (s *ProjectStore) Create(ctx context.Context, project *models.Project) error {
	project.CreatedAt = time.Now()
	project.UpdatedAt = time.Now()

	_, err := s.client.Collection(s.collection).Doc(project.Name).Set(ctx, project)
	if err != nil {
		return fmt.Errorf("failed to create project: %w", err)
	}
	return nil
}

func (s *ProjectStore) Update(ctx context.Context, project *models.Project) error {
	project.UpdatedAt = time.Now()
	_, err := s.client.Collection(s.collection).Doc(project.Name).Set(ctx, project)
	if err != nil {
		return fmt.Errorf("failed to update project: %w", err)
	}
	return nil
}

func (s *ProjectStore) List(ctx context.Context) ([]*models.Project, error) {
	docs, err := s.client.Collection(s.collection).Documents(ctx).GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	projects := make([]*models.Project, 0, len(docs))
	for _, doc := range docs {
		var project models.Project
		if err := doc.DataTo(&project); err != nil {
			continue
		}
		projects = append(projects, &project)
	}
	return projects, nil
}

func (s *ProjectStore) Delete(ctx context.Context, name string) error {
	_, err := s.client.Collection(s.collection).Doc(name).Delete(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
	}
	return nil
}
