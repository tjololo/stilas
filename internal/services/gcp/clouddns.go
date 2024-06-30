package gcp

import (
	"context"

	"github.com/googleapis/gax-go/v2/apierror"
	"google.golang.org/api/dns/v2"
	"google.golang.org/api/option"
)

// CloudDnsService is an interface for interacting with Google Cloud DNS
type CloudDnsService interface {
	// GetZone returns a ManagedZone object for the given project and zone
	GetZone(ctx context.Context, project string, zone string) (*dns.ManagedZone, error)
	// GetOperation returns an Operation object for the given project and zone
	GetOperation(ctx context.Context, project string, zone string, operation string) (*dns.Operation, error)
	// CreateZone creates a new ManagedZone object for the given project
	CreateZone(ctx context.Context, project string, zone *dns.ManagedZone) (*dns.ManagedZone, error)
	// UpdateZone updates an existing ManagedZone object for the given project and zone
	UpdateZone(ctx context.Context, project string, zone string, mz *dns.ManagedZone) (*dns.Operation, error)
	// DeleteZone deletes a ManagedZone object for the given project and zone
	DeleteZone(ctx context.Context, project string, zone string) error
	// GetRecord returns a RecordSet object for the given project, zone and record
	GetRecord(ctx context.Context, project string, zone string, record string, type_ string) (*dns.ResourceRecordSet, error)
}

type newCloudDnsService func(ctx context.Context, opts ...option.ClientOption) (*dns.Service, error)

type GcpCloudDnsService struct {
	NewService    newCloudDnsService
	ClientOptions []option.ClientOption
}

func (g *GcpCloudDnsService) GetZone(ctx context.Context, project string, zone string) (*dns.ManagedZone, error) {
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

func (g *GcpCloudDnsService) GetOperation(ctx context.Context, project string, zone string, operation string) (*dns.Operation, error) {
	svc, err := g.NewService(ctx, g.ClientOptions...)
	if err != nil {
		return nil, err
	}
	op, err := svc.ManagedZoneOperations.Get(project, "global", zone, operation).Do()
	if err != nil {
		return nil, err
	}
	return op, nil
}

func (g *GcpCloudDnsService) CreateZone(ctx context.Context, project string, zone *dns.ManagedZone) (*dns.ManagedZone, error) {
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

func (g *GcpCloudDnsService) UpdateZone(ctx context.Context, project string, zoneName string, zone *dns.ManagedZone) (*dns.Operation, error) {
	svc, err := g.NewService(ctx, g.ClientOptions...)
	if err != nil {
		return nil, err
	}
	op, err := svc.ManagedZones.Update(project, "global", zoneName, zone).Do()
	if err != nil {
		return nil, err
	}
	return op, nil
}

func (g *GcpCloudDnsService) DeleteZone(ctx context.Context, project string, zone string) error {
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

func (g *GcpCloudDnsService) GetRecord(ctx context.Context, project string, zone string, record string, type_ string) (*dns.ResourceRecordSet, error) {
	svc, err := g.NewService(ctx, g.ClientOptions...)
	if err != nil {
		return nil, err
	}
	rs, err := svc.ResourceRecordSets.Get(project, "global", zone, record, type_).Do()
	if err != nil {
		return nil, err
	}
	return rs, nil

}

func ApiErrorFromErr(err error) *apierror.APIError {
	ae, ok := apierror.FromError(err)
	if ok {
		return ae
	}
	return nil
}
