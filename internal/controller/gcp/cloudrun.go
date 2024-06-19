package gcp

import (
	"context"
	"errors"
	"fmt"

	gcprun "cloud.google.com/go/run/apiv2"
	"cloud.google.com/go/run/apiv2/runpb"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	gcpv1 "github.com/tjololo/stilas/api/gcp/v1"
)

type newCloudRunServiceClient func(ctx context.Context, opts ...option.ClientOption) (*gcprun.ServicesClient, error)

func (r *CloudRunReconciler) updateRunService(ctx context.Context, updatedService *runpb.Service) (*gcprun.UpdateServiceOperation, error) {
	c, err := r.getClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create cloud run client: %w", err)
	}
	defer func(c *gcprun.ServicesClient) {
		_ = c.Close()
	}(c)
	crs, err := c.UpdateService(ctx, &runpb.UpdateServiceRequest{
		Service: updatedService,
	})
	if err != nil {
		return nil, fmt.Errorf("UpdateService: failed to update cloud run service: %w", err)
	}
	return crs, nil
}

func (r *CloudRunReconciler) getRunService(ctx context.Context, run gcpv1.CloudRun) (*runpb.Service, error) {
	c, err := r.getClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create cloud run client: %w", err)
	}
	defer func(c *gcprun.ServicesClient) {
		_ = c.Close()
	}(c)
	srv, err := c.GetService(ctx, &runpb.GetServiceRequest{
		Name: run.GetGcpCloudRunServiceFullName(),
	})
	if err != nil {
		return nil, fmt.Errorf("GetService: failed to get cloud run service: %w", err)
	}
	return srv, nil
}

func (r *CloudRunReconciler) checkRunOperationStatus(ctx context.Context, operationName string) (bool, error) {
	c, err := r.getClient(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to create cloud run client: %w", err)
	}
	defer func(c *gcprun.ServicesClient) {
		_ = c.Close()
	}(c)
	crs := c.CreateServiceOperation(operationName)
	_, err = crs.Poll(ctx)
	if err != nil {
		return crs.Done(), fmt.Errorf("Poll: failed to poll cloud run operation: %w", err)
	}
	return crs.Done(), nil
}

func (r *CloudRunReconciler) createRunService(ctx context.Context, cloudRun gcpv1.CloudRun) (*gcprun.CreateServiceOperation, error) {
	c, err := r.getClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create cloud run client: %w", err)
	}
	defer func(c *gcprun.ServicesClient) {
		_ = c.Close()
	}(c)
	runService := cloudRun.ConvertToCreateServiceRequest()
	crs, err := c.CreateService(ctx, runService)
	if err != nil {
		return nil, fmt.Errorf("CreateService: failed to create cloud run service: %w", err)
	}
	return crs, nil
}

func (r *CloudRunReconciler) deleteRunService(ctx context.Context, run gcpv1.CloudRun) (*gcprun.DeleteServiceOperation, error) {
	c, err := r.getClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create cloud run client: %w", err)
	}
	defer func(c *gcprun.ServicesClient) {
		_ = c.Close()
	}(c)
	dso, err := c.DeleteService(ctx, &runpb.DeleteServiceRequest{
		Name: run.GetGcpCloudRunServiceFullName(),
	})
	if err != nil {
		return nil, fmt.Errorf("DeleteService: failed to delete cloud run service: %w", err)
	}
	return dso, nil
}

func (r *CloudRunReconciler) getClient(ctx context.Context) (*gcprun.ServicesClient, error) {
	return r.NewClient(ctx, r.ClientOptions...)
}

func isRunServiceNotFoundError(err error) bool {
	if gs, ok := statusFromError(err); ok {
		return gs.Code() == codes.NotFound
	}
	return false
}

func statusFromError(err error) (*status.Status, bool) {
	type gRPCError interface {
		GRPCStatus() *status.Status
	}

	var se gRPCError
	if errors.As(err, &se) {
		return se.GRPCStatus(), true
	}

	return nil, false
}
