package deploy

import (
	"context"
	"fmt"
	"time"

	run "google.golang.org/api/run/v2"

	"github.com/anishalle/hack/server/internal/provider"
)

type CloudRunProvider struct {
	service *run.ProjectsLocationsServicesService
}

func NewCloudRunProvider(ctx context.Context) (*CloudRunProvider, error) {
	runService, err := run.NewService(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create Cloud Run client: %w", err)
	}

	return &CloudRunProvider{
		service: run.NewProjectsLocationsServicesService(runService),
	}, nil
}

func (p *CloudRunProvider) Deploy(ctx context.Context, opts provider.DeployOptions) (*provider.DeployResult, error) {
	serviceName := fmt.Sprintf("projects/%s/locations/%s/services/%s",
		opts.Project, opts.Region, opts.Service)

	svc, err := p.service.Get(serviceName).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get service %s: %w", opts.Service, err)
	}

	image := opts.Image
	if image == "" && opts.Tag != "" {
		image = opts.Tag
	}

	if image != "" && svc.Template != nil && len(svc.Template.Containers) > 0 {
		svc.Template.Containers[0].Image = image
	}

	if opts.EnvVars != nil && svc.Template != nil && len(svc.Template.Containers) > 0 {
		envVars := make([]*run.GoogleCloudRunV2EnvVar, 0, len(opts.EnvVars))
		for k, v := range opts.EnvVars {
			envVars = append(envVars, &run.GoogleCloudRunV2EnvVar{
				Name:  k,
				Value: v,
			})
		}
		svc.Template.Containers[0].Env = envVars
	}

	op, err := p.service.Patch(serviceName, svc).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to deploy: %w", err)
	}

	return &provider.DeployResult{
		Service:  opts.Service,
		Revision: op.Name,
		URL:      svc.Uri,
		Status:   "deploying",
	}, nil
}

func (p *CloudRunProvider) Status(ctx context.Context, opts provider.StatusOptions) (*provider.ServiceStatus, error) {
	serviceName := fmt.Sprintf("projects/%s/locations/%s/services/%s",
		opts.Project, opts.Region, opts.Service)

	svc, err := p.service.Get(serviceName).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get service status: %w", err)
	}

	status := "unknown"
	for _, cond := range svc.Conditions {
		if cond.Type == "Ready" {
			if cond.State == "CONDITION_SUCCEEDED" {
				status = "ready"
			} else {
				status = "not_ready"
			}
		}
	}

	revision := ""
	if svc.LatestReadyRevision != "" {
		revision = svc.LatestReadyRevision
	}

	return &provider.ServiceStatus{
		Service:        opts.Service,
		Status:         status,
		URL:            svc.Uri,
		LatestRevision: revision,
		UpdatedAt:      svc.UpdateTime,
	}, nil
}

func (p *CloudRunProvider) Logs(ctx context.Context, opts provider.LogsOptions) ([]provider.LogEntry, error) {
	// Cloud Run logs are fetched through Cloud Logging API, not Cloud Run API.
	// This would need a separate Cloud Logging client.
	return []provider.LogEntry{
		{
			Timestamp: time.Now().Format(time.RFC3339),
			Severity:  "INFO",
			Message:   "Use 'gcloud logging read' for full log access. Cloud Logging integration coming soon.",
		},
	}, nil
}

func (p *CloudRunProvider) Rollback(ctx context.Context, opts provider.RollbackOptions) (*provider.DeployResult, error) {
	serviceName := fmt.Sprintf("projects/%s/locations/%s/services/%s",
		opts.Project, opts.Region, opts.Service)

	svc, err := p.service.Get(serviceName).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get service: %w", err)
	}

	// Rollback by setting traffic to the specified revision
	if opts.Revision != "" && svc.Traffic != nil {
		svc.Traffic = []*run.GoogleCloudRunV2TrafficTarget{
			{
				Revision: opts.Revision,
				Percent:  100,
				Type:     "TRAFFIC_TARGET_ALLOCATION_TYPE_REVISION",
			},
		}
	}

	_, err = p.service.Patch(serviceName, svc).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to rollback: %w", err)
	}

	return &provider.DeployResult{
		Service:  opts.Service,
		Revision: opts.Revision,
		Status:   "rolling_back",
	}, nil
}

func (p *CloudRunProvider) Restart(ctx context.Context, opts provider.RestartOptions) error {
	serviceName := fmt.Sprintf("projects/%s/locations/%s/services/%s",
		opts.Project, opts.Region, opts.Service)

	svc, err := p.service.Get(serviceName).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to get service: %w", err)
	}

	// Force a new revision by updating an annotation
	if svc.Template == nil {
		return fmt.Errorf("service has no template")
	}
	if svc.Template.Annotations == nil {
		svc.Template.Annotations = make(map[string]string)
	}
	svc.Template.Annotations["hack/restarted-at"] = time.Now().Format(time.RFC3339)

	_, err = p.service.Patch(serviceName, svc).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to restart: %w", err)
	}

	return nil
}
