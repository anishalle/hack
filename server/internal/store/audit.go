package store

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

type AuditEntry struct {
	ID        string    `firestore:"-"`
	Project   string    `firestore:"project"`
	User      string    `firestore:"user"`
	Action    string    `firestore:"action"`
	Resource  string    `firestore:"resource"`
	Details   string    `firestore:"details,omitempty"`
	Timestamp time.Time `firestore:"timestamp"`
}

type AuditStore struct {
	client     *firestore.Client
	collection string
}

func NewAuditStore(client *firestore.Client) *AuditStore {
	return &AuditStore{client: client, collection: "audit_log"}
}

func (s *AuditStore) Log(ctx context.Context, entry *AuditEntry) error {
	entry.Timestamp = time.Now()

	_, _, err := s.client.Collection(s.collection).Add(ctx, entry)
	if err != nil {
		return fmt.Errorf("failed to write audit log: %w", err)
	}
	return nil
}

func (s *AuditStore) Query(ctx context.Context, project string, limit int, filters ...AuditFilter) ([]*AuditEntry, error) {
	q := s.client.Collection(s.collection).
		Where("project", "==", project).
		OrderBy("timestamp", firestore.Desc)

	for _, f := range filters {
		q = f(q)
	}

	if limit > 0 {
		q = q.Limit(limit)
	}

	iter := q.Documents(ctx)

	var entries []*AuditEntry

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read audit log: %w", err)
		}

		var entry AuditEntry
		if err := doc.DataTo(&entry); err != nil {
			continue
		}
		entry.ID = doc.Ref.ID
		entries = append(entries, &entry)
	}

	return entries, nil
}

type AuditFilter func(firestore.Query) firestore.Query

func FilterByUser(email string) AuditFilter {
	return func(q firestore.Query) firestore.Query {
		return q.Where("user", "==", email)
	}
}

func FilterByAction(action string) AuditFilter {
	return func(q firestore.Query) firestore.Query {
		return q.Where("action", "==", action)
	}
}
