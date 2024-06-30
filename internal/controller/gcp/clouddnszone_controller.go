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

package gcp

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/api/dns/v2"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	gcpv1 "github.com/tjololo/stilas/api/gcp/v1"
	"github.com/tjololo/stilas/internal/services/gcp"
)

// CloudDnsZoneReconciler reconciles a CloudDnsZone object
type CloudDnsZoneReconciler struct {
	client.Client
	CloudDnsService gcp.CloudDnsService
	Scheme          *runtime.Scheme
	ClientOptions   []option.ClientOption
}

// +kubebuilder:rbac:groups=gcp.stilas.418.cloud,resources=clouddnszones,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=gcp.stilas.418.cloud,resources=clouddnszones/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=gcp.stilas.418.cloud,resources=clouddnszones/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.

// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.18.2/pkg/reconcile
func (r *CloudDnsZoneReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var dnsZone gcpv1.CloudDnsZone
	if err := r.Client.Get(ctx, req.NamespacedName, &dnsZone); err != nil {
		if !errors.IsNotFound(err) {
			logger.Error(err, "unable to fetch CloudDnsZone")
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !controllerutil.ContainsFinalizer(&dnsZone, finalizerName) {
		return ctrl.Result{}, r.addFinalizer(ctx, &dnsZone)
	}

	if dnsZone.DeletionTimestamp != nil {
		if dnsZone.Spec.CleanupOnDelete {
			err := r.CloudDnsService.DeleteZone(ctx, dnsZone.Spec.ProjectID, dnsZone.GetCloudDnsZoneFullName())
			if err != nil {
				logger.Error(err, "unable to delete ManagedZone")
			} else {
				controllerutil.RemoveFinalizer(&dnsZone, finalizerName)
				if err := r.Client.Update(ctx, &dnsZone); err != nil {
					return ctrl.Result{}, err
				}
			}
			return ctrl.Result{}, err
		}
		controllerutil.RemoveFinalizer(&dnsZone, finalizerName)
		if err := r.Client.Update(ctx, &dnsZone); err != nil {
			return ctrl.Result{}, err
		}
	}

	logger.Info(fmt.Sprintf("Reconciling CloudDnsZone: %+v", dnsZone.Spec))
	if dnsZone.Status.Operation != "" {
		op, err := r.CloudDnsService.GetOperation(ctx, dnsZone.Spec.ProjectID, dnsZone.GetCloudDnsZoneFullName(), dnsZone.Status.Operation)
		if err != nil {
			logger.Error(err, "unable to get Operation")
			return ctrl.Result{}, err
		}
		if op.Status == "DONE" {
			dnsZone.Status.Operation = ""
			return ctrl.Result{}, r.Client.Status().Update(ctx, &dnsZone)
		}
		return ctrl.Result{}, nil
	}
	zone, err := r.CloudDnsService.GetZone(ctx, dnsZone.Spec.ProjectID, dnsZone.GetCloudDnsZoneFullName())
	if err == nil && dnsZoneUpdated(&dnsZone, zone) {
		logger.Info("ManagedZone updated, updating.")
		zone.DnssecConfig.State = dnsZone.Spec.DnsSecSpec.State
		op, err := r.CloudDnsService.UpdateZone(ctx, dnsZone.Spec.ProjectID, dnsZone.GetCloudDnsZoneFullName(), zone)
		if err != nil {
			return ctrl.Result{}, err
		}
		if op.Status == "DONE" {
			dnsZone.Status.Nameservers = zone.NameServers
			return ctrl.Result{}, r.Client.Status().Update(ctx, &dnsZone)
		} else {
			dnsZone.Status.Operation = op.Id
			return ctrl.Result{}, r.Client.Status().Update(ctx, &dnsZone)
		}
	}
	if err != nil && !googleapi.IsNotModified(err) {
		apiErr := gcp.ApiErrorFromErr(err)
		if apiErr != nil && apiErr.HTTPCode() == 404 {
			logger.Info("ManagedZone not found, creating.")
			zone := dns.ManagedZone{
				Name:        dnsZone.GetCloudDnsZoneFullName(),
				Description: "DnsZone created by Stilas",
				DnsName:     fmt.Sprintf("%s.", dnsZone.Spec.DnsName),
				Visibility:  "PRIVATE",
				DnssecConfig: &dns.ManagedZoneDnsSecConfig{
					NonExistence: "NSEC3",
					State:        dnsZone.Spec.DnsSecSpec.State,
				},
			}
			if !dnsZone.Spec.PrivateZone {
				zone.Visibility = "PUBLIC"
			}
			mz, err := r.CloudDnsService.CreateZone(ctx, dnsZone.Spec.ProjectID, &zone)
			if err != nil {
				return ctrl.Result{}, err
			}
			dnsZone.Status.Nameservers = mz.NameServers
			if err := r.Client.Status().Update(ctx, &dnsZone); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}
		logger.Error(err, "unable to get ManagedZone")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CloudDnsZoneReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gcpv1.CloudDnsZone{}).
		Complete(r)
}

func (r *CloudDnsZoneReconciler) addFinalizer(ctx context.Context, dnsZone *gcpv1.CloudDnsZone) error {
	controllerutil.AddFinalizer(dnsZone, finalizerName)
	if err := r.Client.Update(ctx, dnsZone); err != nil {
		return err
	}
	return nil
}

func dnsZoneUpdated(new *gcpv1.CloudDnsZone, current *dns.ManagedZone) bool {
	return !strings.EqualFold(new.Spec.DnsSecSpec.State, current.DnssecConfig.State)
}
