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
	"fmt"
	"time"

	"cloud.google.com/go/iam/apiv1/iampb"
	gcprun "cloud.google.com/go/run/apiv2"
	"google.golang.org/api/option"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	gcpv1 "github.com/tjololo/stilas/api/gcp/v1"
)

const finalizerName = "cloudrun.gcp.stilas.418.cloud/finalizer"

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
		if !errors.IsNotFound(err) {
			logger.Error(err, "unable to fetch CloudRun")
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	logger.Info(fmt.Sprintf("Reconciling CloudRun: %+v", run.Spec))
	if !controllerutil.ContainsFinalizer(&run, finalizerName) {
		controllerutil.AddFinalizer(&run, finalizerName)
		if err := r.Client.Update(ctx, &run); err != nil {
			logger.Error(err, "unable to add finalizer")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	if run.DeletionTimestamp != nil {
		return ctrl.Result{RequeueAfter: time.Second}, r.handleDeletion(ctx, run)
	}

	runningOperations := getOngoingOperations(run.Status.Operations)
	if runningOperations != nil {
		allDone := true
		for _, operation := range *runningOperations {
			done, err := r.checkRunOperationStatus(ctx, operation.Name)
			if err != nil {
				logger.Error(err, "unable to check cloud run operation")
				return ctrl.Result{}, err
			}
			if !done {
				allDone = false
			}
			updateOperationStatusByName(&run, operation.Name, done)
		}
		if err := r.Client.Status().Update(ctx, &run); err != nil {
			logger.Error(err, "unable to update cloud run status")
			return ctrl.Result{}, err
		}
		if !allDone {
			return ctrl.Result{RequeueAfter: time.Second}, nil
		}
	} else {
		srv, err := r.getRunService(ctx, run)
		if err == nil {
			if srv.Template.Containers[0].Image != run.Spec.Containers[0].Image {
				srv.Template.Containers[0].Image = run.Spec.Containers[0].Image
				logger.Info(fmt.Sprintf("Image has changed, current image: %s, new image: %s", srv.Template.Containers[0].Image, run.Spec.Containers[0].Image))
				cr, err := r.updateRunService(ctx, srv)
				if err != nil {
					logger.Error(err, "unable to update cloud run service")
					return ctrl.Result{}, err
				}
				run.Status.Operations = append(run.Status.Operations, &gcpv1.CloudRunOperation{
					Name:          cr.Name(),
					Done:          cr.Done(),
					OperationType: gcpv1.CloudRunOperationType_Update,
				})
				if err := r.Client.Status().Update(ctx, &run); err != nil {
					logger.Error(err, "unable to update cloud run status")
					return ctrl.Result{}, err
				}
			} else {
				run.Status.Uri = srv.Uri
				run.Status.LatestReadyRevision = srv.LatestReadyRevision
				run.Status.Reconciling = srv.Reconciling
				if err := r.Client.Status().Update(ctx, &run); err != nil {
					logger.Error(err, "unable to update cloud run status")
					return ctrl.Result{}, err
				}
				err = r.setIamPolicy(ctx, run)
				if err != nil {
					logger.Error(err, "unable to set iam policy")
					return ctrl.Result{}, err
				}
			}
		} else {
			if isRunServiceNotFoundError(err) {
				cr, err := r.createRunService(ctx, run)
				if err != nil {
					logger.Error(err, "unable to create cloud run service")
					return ctrl.Result{}, err
				}
				run.Status.Operations = append(run.Status.Operations, &gcpv1.CloudRunOperation{
					Name:          cr.Name(),
					Done:          cr.Done(),
					OperationType: gcpv1.CloudRunOperationType_Create,
				})
				if err := r.Client.Status().Update(ctx, &run); err != nil {
					logger.Error(err, "unable to update cloud run status")
					return ctrl.Result{}, err
				}
				return ctrl.Result{}, nil
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

func (r *CloudRunReconciler) handleDeletion(ctx context.Context, cloudRun gcpv1.CloudRun) error {
	logger := log.FromContext(ctx)
	deleteOperations := getOperationsByType(cloudRun.Status.Operations, gcpv1.CloudRunOperationType_Delete)
	if deleteOperations == nil {
		dso, err := r.deleteRunService(ctx, cloudRun)
		if err != nil {
			logger.Error(err, "unable to delete cloud run service")
			return err
		}
		cloudRun.Status.Operations = append(cloudRun.Status.Operations, &gcpv1.CloudRunOperation{
			Name:          dso.Name(),
			Done:          dso.Done(),
			OperationType: gcpv1.CloudRunOperationType_Delete,
		})
		if err := r.Client.Status().Update(ctx, &cloudRun); err != nil {
			logger.Error(err, "unable to update cloud run status")
			return err
		}
		return nil
	} else {
		allDone := true
		for _, operation := range *deleteOperations {
			done, err := r.checkRunOperationStatus(ctx, operation.Name)
			if err != nil {
				logger.Error(err, "unable to check cloud run operation")
				return err
			}
			if !done {
				allDone = false
			}
			updateOperationStatusByName(&cloudRun, operation.Name, done)
		}
		if err := r.Client.Status().Update(ctx, &cloudRun); err != nil {
			logger.Error(err, "unable to update cloud run status")
			return err
		}
		if !allDone {
			return nil
		}
		controllerutil.RemoveFinalizer(&cloudRun, finalizerName)
		if err := r.Client.Update(ctx, &cloudRun); err != nil {
			logger.Error(err, "unable to remove finalizer")
			return err
		}
		return nil
	}
}

func getOngoingOperations(operations []*gcpv1.CloudRunOperation) *[]gcpv1.CloudRunOperation {
	var ongoing []gcpv1.CloudRunOperation
	for _, operation := range operations {
		if !operation.Done {
			ongoing = append(ongoing, *operation)
		}
	}
	if len(ongoing) == 0 {
		return nil
	}
	return &ongoing
}

func getOperationsByType(operations []*gcpv1.CloudRunOperation, operationType gcpv1.CloudRunOperationType) *[]gcpv1.CloudRunOperation {
	var operationsOfType []gcpv1.CloudRunOperation
	for _, operation := range operations {
		if operation.OperationType == operationType {
			operationsOfType = append(operationsOfType, *operation)
		}
	}
	if len(operationsOfType) == 0 {
		return nil
	}
	return &operationsOfType
}

func updateOperationStatusByName(cloudRun *gcpv1.CloudRun, operationName string, done bool) {
	for _, operation := range cloudRun.Status.Operations {
		if operation.Name == operationName {
			operation.Done = done
		}
	}
}
