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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gcpdns "google.golang.org/api/dns/v2"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	gcpv1 "github.com/tjololo/stilas/api/gcp/v1"
)

type mockCloudDnsService struct {
}

func (m *mockCloudDnsService) GetZone(_ context.Context, _ string, _ string) (*gcpdns.ManagedZone, error) {
	return &gcpdns.ManagedZone{
		Name:    "test-zone",
		DnsName: "test-dns-name",
	}, nil
}

func (m *mockCloudDnsService) GetOperation(_ context.Context, _ string, _ string, _ string) (*gcpdns.Operation, error) {
	return &gcpdns.Operation{Status: "DONE"}, nil
}

func (m *mockCloudDnsService) CreateZone(_ context.Context, _ string, _ *gcpdns.ManagedZone) (*gcpdns.ManagedZone, error) {
	return &gcpdns.ManagedZone{
		Name:    "test-zone",
		DnsName: "test-dns-name",
	}, nil
}

func (m *mockCloudDnsService) UpdateZone(_ context.Context, _ string, _ string, _ *gcpdns.ManagedZone) (*gcpdns.Operation, error) {
	return &gcpdns.Operation{Status: "DONE"}, nil
}

func (m *mockCloudDnsService) DeleteZone(_ context.Context, _ string, _ string) error {
	return nil
}

func (m *mockCloudDnsService) GetRecord(_ context.Context, _ string, _ string, _ string, _ string) (*gcpdns.ResourceRecordSet, error) {
	return &gcpdns.ResourceRecordSet{
		Name: "test-record",
		Type: "test-type",
	}, nil
}

var _ = Describe("CloudDnsZone Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default", // TODO(user):Modify as needed
		}
		clouddnszone := &gcpv1.CloudDnsZone{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind CloudDnsZone")
			err := k8sClient.Get(ctx, typeNamespacedName, clouddnszone)
			if err != nil && errors.IsNotFound(err) {
				resource := &gcpv1.CloudDnsZone{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					// TODO(user): Specify other spec details if needed.
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
			By("Creating the fakeCloudRunServiceClient")
		})

		AfterEach(func() {
			// TODO(user): Cleanup logic after each test, like removing the resource instance.
			resource := &gcpv1.CloudDnsZone{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance CloudDnsZone")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &CloudDnsZoneReconciler{
				Client:          k8sClient,
				Scheme:          k8sClient.Scheme(),
				CloudDnsService: &mockCloudDnsService{},
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			// TODO(user): Add more specific assertions depending on your controller's reconciliation logic.
			// Example: If you expect a certain status condition after reconciliation, verify it here.
		})
	})
})
