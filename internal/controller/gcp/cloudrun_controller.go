/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"errors"
	"fmt"
	"time"

	"cloud.google.com/go/iam/apiv1/iampb"
	gcprun "cloud.google.com/go/run/apiv2"
	"cloud.google.com/go/run/apiv2/runpb"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	gcpv1 "github.com/tjololo/stilas/api/gcp/v1"
)

// CloudRunReconciler reconciles a CloudRun object
type CloudRunReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	NewClient     newCloudRunServiceClient
	ClientOptions []option.ClientOption
}

//+kubebuilder:rbac:groups=gcp.stilas.418.cloud,resources=cloudruns,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=gcp.stilas.418.cloud,resources=cloudruns/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=gcp.stilas.418.cloud,resources=cloudruns/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the CloudRun object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.2/pkg/reconcile
func (r *CloudRunReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var run gcpv1.CloudRun
	if err := r.Client.Get(ctx, req.NamespacedName, &run); err != nil {
		logger.Error(err, "unable to fetch CloudRun")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	logger.Info(fmt.Sprintf("Reconciling CloudRun: %+v", run.Spec))
	if run.Status.OperationsName == "" {
		srv, err := r.getRunService(ctx, run)
		if err != nil {
			if isRunServiceNotFoundError(err) {
				cr, err := r.createRunService(ctx, run)
				if err != nil {
					if isRunServiceAlreadyExistsError(err) {
						logger.Info("Cloud Run Service already exists")
						return ctrl.Result{RequeueAfter: time.Minute}, nil
					}
					logger.Error(err, "unable to create cloud run service")
					return ctrl.Result{}, err
				}
				logger.Info(fmt.Sprintf("Created Cloud Run Service: %+v", cr))
				run.Status = gcpv1.CloudRunStatus{
					OperationsName: cr.Name(),
					Done:           cr.Done(),
				}
				if err := r.Client.Status().Update(ctx, &run); err != nil {
					logger.Error(err, "unable to update cloud run status")
					return ctrl.Result{}, err
				}
			}
		} else {
			if srv.Template.Containers[0].Image != run.Spec.Containers[0].Image {
				srv.Template.Containers[0].Image = run.Spec.Containers[0].Image
				cr, err := r.updateRunService(ctx, srv)
				if err != nil {
					if isRunServiceAlreadyExistsError(err) {
						logger.Info("Cloud Run Service already exists")
						return ctrl.Result{RequeueAfter: time.Minute}, nil
					}
					logger.Error(err, "unable to create cloud run service")
					return ctrl.Result{}, err
				}
				logger.Info(fmt.Sprintf("Created Cloud Run Service: %+v", cr))
				run.Status.OperationsName = cr.Name()
				run.Status.Done = cr.Done()
				if err := r.Client.Status().Update(ctx, &run); err != nil {
					logger.Error(err, "unable to update cloud run status")
					return ctrl.Result{}, err
				}
			}
		}
	} else if !run.Status.Done {
		logger.Info(fmt.Sprintf("Cloud Run Service already created: %+v", run.Status.OperationsName))
		srv, err := r.checkRunOperationStatus(ctx, run)
		if err != nil {
			logger.Error(err, "unable to check cloud run service")
			return ctrl.Result{}, err
		}
		if srv == nil {
			logger.Info("Operation not done, reque after 1 second")
			return ctrl.Result{RequeueAfter: time.Second}, nil
		} else {
			run.Status.Done = true
			run.Status.OperationsName = ""
			run.Status.Uri = srv.Uri
			run.Status.LatestReadyRevision = srv.LatestReadyRevision
			if run.Status.Revisions == nil {
				run.Status.Revisions = make([]string, 0)
			}
			run.Status.Revisions = append(run.Status.Revisions, srv.LatestReadyRevision)
			err = r.setIamPolicy(ctx, run)
			if err != nil {
				logger.Error(err, "unable to set iam policy")
				return ctrl.Result{}, err
			}
			if err := r.Client.Status().Update(ctx, &run); err != nil {
				logger.Error(err, "unable to update cloud run status")
				return ctrl.Result{}, err
			}
		}
	}

	return ctrl.Result{RequeueAfter: time.Minute}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CloudRunReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gcpv1.CloudRun{}).
		Complete(r)
}

func (r *CloudRunReconciler) createRunService(ctx context.Context, cloudRun gcpv1.CloudRun) (*gcprun.CreateServiceOperation, error) {
	c, err := r.getClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create cloud run client: %w", err)
	}
	defer func(c *gcprun.ServicesClient) {
		_ = c.Close()
	}(c)
	runService := runpb.CreateServiceRequest{
		Parent: fmt.Sprintf("projects/%s/locations/%s", cloudRun.Spec.ProjectID, cloudRun.Spec.Location),
		Service: &runpb.Service{
			Ingress: cloudRun.Spec.TrafficMode,
			Template: &runpb.RevisionTemplate{
				Containers: []*runpb.Container{
					{
						Image: cloudRun.Spec.Containers[0].Image,
						Name:  cloudRun.Spec.Containers[0].Name,
						Ports: []*runpb.ContainerPort{
							{
								ContainerPort: cloudRun.Spec.Containers[0].Port,
							},
						},
					},
				},
			},
			Traffic: []*runpb.TrafficTarget{
				{
					Percent: 100,
					Type:    runpb.TrafficTargetAllocationType_TRAFFIC_TARGET_ALLOCATION_TYPE_LATEST,
				},
			},
		},
		ServiceId: cloudRun.Name,
	}
	crs, err := c.CreateService(ctx, &runService)
	if err != nil {
		return nil, fmt.Errorf("CreateService: failed to create cloud run service: %w", err)
	}
	return crs, nil
}

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

func (r *CloudRunReconciler) checkRunOperationStatus(ctx context.Context, cloudRun gcpv1.CloudRun) (*runpb.Service, error) {
	c, err := r.getClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create cloud run client: %w", err)
	}
	defer func(c *gcprun.ServicesClient) {
		_ = c.Close()
	}(c)
	crs := c.CreateServiceOperation(cloudRun.Status.OperationsName)
	srv, err := crs.Poll(ctx)
	if err != nil {
		return nil, fmt.Errorf("Poll: failed to poll cloud run operation: %w", err)
	}
	return srv, nil
}

func (r *CloudRunReconciler) getRunService(ctx context.Context, cloudRun gcpv1.CloudRun) (*runpb.Service, error) {
	c, err := r.getClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create cloud run client: %w", err)
	}
	defer func(c *gcprun.ServicesClient) {
		_ = c.Close()
	}(c)
	srv, err := c.GetService(ctx, &runpb.GetServiceRequest{
		Name: fmt.Sprintf("projects/%s/locations/%s/services/%s", cloudRun.Spec.ProjectID, cloudRun.Spec.Location, cloudRun.Name),
	})
	if err != nil {
		return nil, fmt.Errorf("GetService: failed to get cloud run service: %w", err)
	}
	return srv, nil
}

func (r *CloudRunReconciler) setIamPolicy(ctx context.Context, cloudRun gcpv1.CloudRun) error {
	c, err := r.getClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create cloud run client: %w", err)
	}
	defer func(c *gcprun.ServicesClient) {
		_ = c.Close()
	}(c)

	policyRequest := &iampb.SetIamPolicyRequest{
		Resource: fmt.Sprintf("projects/%s/locations/%s/services/%s", cloudRun.Spec.ProjectID, cloudRun.Spec.Location, cloudRun.Name),
		Policy: &iampb.Policy{
			Bindings: []*iampb.Binding{
				{
					Role:    "roles/run.invoker",
					Members: cloudRun.Spec.InvokeMembers,
				},
			},
		},
	}

	_, err = c.SetIamPolicy(ctx, policyRequest)
	if err != nil {
		return fmt.Errorf("SetIamPolicy: failed to set iam policy: %w", err)
	}
	return nil
}

func (r *CloudRunReconciler) getClient(ctx context.Context) (*gcprun.ServicesClient, error) {
	return r.NewClient(ctx, r.ClientOptions...)
}

func isRunServiceAlreadyExistsError(err error) bool {
	if gs, ok := statusFromError(err); ok {
		return gs.Code() == codes.AlreadyExists
	}
	return false
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