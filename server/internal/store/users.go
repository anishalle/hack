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

type UserStore struct {
	client     *firestore.Client
	collection string
}

func NewUserStore(client *firestore.Client) *UserStore {
	return &UserStore{client: client, collection: "users"}
}

func (s *UserStore) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	iter := s.client.Collection(s.collection).Where("email", "==", email).Limit(1).Documents(ctx)
	doc, err := iter.Next()
	if err != nil {
		return nil, fmt.Errorf("user not found: %s", email)
	}

	var user models.User
	if err := doc.DataTo(&user); err != nil {
		return nil, fmt.Errorf("failed to parse user: %w", err)
	}
	user.ID = doc.Ref.ID
	return &user, nil
}

func (s *UserStore) GetByID(ctx context.Context, id string) (*models.User, error) {
	doc, err := s.client.Collection(s.collection).Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, fmt.Errorf("user not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	var user models.User
	if err := doc.DataTo(&user); err != nil {
		return nil, fmt.Errorf("failed to parse user: %w", err)
	}
	user.ID = doc.Ref.ID
	return &user, nil
}

func (s *UserStore) Create(ctx context.Context, user *models.User) error {
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	if user.Projects == nil {
		user.Projects = make(map[string]string)
	}

	ref, _, err := s.client.Collection(s.collection).Add(ctx, user)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	user.ID = ref.ID
	return nil
}

func (s *UserStore) Update(ctx context.Context, user *models.User) error {
	if user.ID == "" {
		return fmt.Errorf("user ID is required for update")
	}

	user.UpdatedAt = time.Now()
	_, err := s.client.Collection(s.collection).Doc(user.ID).Set(ctx, user)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	return nil
}

func (s *UserStore) ListByProject(ctx context.Context, project string) ([]*models.User, error) {
	key := fmt.Sprintf("projects.%s", project)
	iter := s.client.Collection(s.collection).Where(key, "!=", "").Documents(ctx)
	docs, err := iter.GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	users := make([]*models.User, 0, len(docs))
	for _, doc := range docs {
		var user models.User
		if err := doc.DataTo(&user); err != nil {
			continue
		}
		user.ID = doc.Ref.ID
		users = append(users, &user)
	}
	return users, nil
}

func (s *UserStore) Delete(ctx context.Context, id string) error {
	_, err := s.client.Collection(s.collection).Doc(id).Delete(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	return nil
}
