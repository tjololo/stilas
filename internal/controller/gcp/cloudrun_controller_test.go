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
	"net"

	"cloud.google.com/go/longrunning/autogen/longrunningpb"
	gcprun "cloud.google.com/go/run/apiv2"
	"cloud.google.com/go/run/apiv2/runpb"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	gcpv1 "github.com/tjololo/stilas/api/gcp/v1"
)

type fakeCloudRunServiceClient struct {
	runpb.UnimplementedServicesServer
}

func (f *fakeCloudRunServiceClient) CreateService(_ context.Context, _ *runpb.CreateServiceRequest) (*longrunningpb.Operation, error) {
	return &longrunningpb.Operation{Name: "test-operation", Done: true}, nil
}

var _ = Describe("CloudRun Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default", // TODO(user):Modify as needed
		}
		cloudrun := &gcpv1.CloudRun{}
		var fakeServerAddr string
		BeforeEach(func() {
			By("creating the custom resource for the Kind CloudRun")
			err := k8sClient.Get(ctx, typeNamespacedName, cloudrun)
			if err != nil && errors.IsNotFound(err) {
				resource := &gcpv1.CloudRun{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: gcpv1.CloudRunSpec{
						Name:      "test",
						Location:  "us-central1",
						ProjectID: "test-project",
						Containers: []gcpv1.CloudRunContainer{
							{
								Image: "gcr.io/test-project/test-image",
								Name:  "test-container",
							},
						},
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
			By("Creating the fakeCloudRunServiceClient")
			fakeCloudRunServiceClient := &fakeCloudRunServiceClient{}
			l, err := net.Listen("tcp", "localhost:0")
			if err != nil {
				Fail("failed to listen")
			}
			gsrv := grpc.NewServer()
			runpb.RegisterServicesServer(gsrv, fakeCloudRunServiceClient)
			fakeServerAddr = l.Addr().String()
			go func() {
				if err := gsrv.Serve(l); err != nil {
					panic(err)
				}
			}()
		})

		AfterEach(func() {
			// TODO(user): Cleanup logic after each test, like removing the resource instance.
			resource := &gcpv1.CloudRun{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance CloudRun")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &CloudRunReconciler{
				Client:    k8sClient,
				Scheme:    k8sClient.Scheme(),
				NewClient: gcprun.NewServicesClient,
				ClientOptions: []option.ClientOption{
					option.WithEndpoint(fakeServerAddr),
					option.WithoutAuthentication(),
					option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
				},
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
