package gcp

import (
	"context"

	"github.com/googleapis/gax-go/v2/apierror"
	"google.golang.org/api/dns/v2"
	"google.golang.org/api/option"
)

// / CloudDnsService is an interface for interacting with Google Cloud DNS
type CloudDnsService interface {
	// GetZone returns a ManagedZone object for the given project and zone
	GetZone(ctx context.Context, project string, zone string) (*dns.ManagedZone, error)
	// CreateZone creates a new ManagedZone object for the given project
	CreateZone(ctx context.Context, project string, zone *dns.ManagedZone) (*dns.ManagedZone, error)
	// DeleteZone deletes a ManagedZone object for the given project and zone
	DeleteZone(ctx context.Context, project string, zone string) error
}

type newCloudDnsService func(ctx context.Context, opts ...option.ClientOption) (*dns.Service, error)

type CloudDnsServiceImpl struct {
	NewService    newCloudDnsService
	ClientOptions []option.ClientOption
}

func (g *CloudDnsServiceImpl) GetZone(ctx context.Context, project string, zone string) (*dns.ManagedZone, error) {
	svc, err := g.NewService(ctx, g.ClientOptions...)
	if err != nil {
		return nil, err
	}
	mz, err := svc.ManagedZones.Get(project, "global", zone).Do()
	if err != nil {
		return nil, err
	}
	return mz, nil
}

func (g *CloudDnsServiceImpl) CreateZone(ctx context.Context, project string, zone *dns.ManagedZone) (*dns.ManagedZone, error) {
	svc, err := g.NewService(ctx, g.ClientOptions...)
	if err != nil {
		return nil, err
	}
	mz, err := svc.ManagedZones.Create(project, "global", zone).Do()
	if err != nil {
		return nil, err
	}
	return mz, nil
}

func (g *CloudDnsServiceImpl) DeleteZone(ctx context.Context, project string, zone string) error {
	svc, err := g.NewService(ctx, g.ClientOptions...)
	if err != nil {
		return err
	}
	err = svc.ManagedZones.Delete(project, "global", zone).Do()
	if err != nil {
		return err
	}
	return nil
}

func ApiErrorFromErr(err error) *apierror.APIError {
	ae, ok := apierror.FromError(err)
	if ok {
		return ae
	}
	return nil
}
