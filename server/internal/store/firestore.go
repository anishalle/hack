package store

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
)

func NewFirestoreClient(ctx context.Context, projectID string) (*firestore.Client, error) {
	var client *firestore.Client
	var err error

	if projectID != "" {
		client, err = firestore.NewClient(ctx, projectID)
	} else {
		client, err = firestore.NewClient(ctx, firestore.DetectProjectID)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create Firestore client: %w", err)
	}

	return client, nil
}
