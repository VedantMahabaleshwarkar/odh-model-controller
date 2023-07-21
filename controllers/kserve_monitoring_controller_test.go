/*

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

package controllers

import (
	"context"

	kservev1beta1 "github.com/kserve/kserve/pkg/apis/serving/v1beta1"
	inferenceservicev1 "github.com/kserve/modelmesh-serving/apis/serving/v1beta1"
	routev1 "github.com/openshift/api/route/v1"
	istiosecurityv1beta1 "istio.io/client-go/pkg/apis/security/v1beta1"
	"k8s.io/apimachinery/pkg/types"

	mfc "github.com/manifestival/controller-runtime-client"
	mf "github.com/manifestival/manifestival"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("The Openshift model controller", func() {

	When("creating a ServiceRuntime & InferenceService with 'enable-route' enabled", func() {
		var opts mf.Option

		BeforeEach(func() {
			client := mfc.NewClient(cli)
			opts = mf.UseClient(client)
			ctx := context.Background()

			caikitIsvc := &kservev1beta1.InferenceService{}
			err := convertToStructuredResource(caikitIsvcPath, caikitIsvc, opts)
			Expect(err).NotTo(HaveOccurred())
			Expect(cli.Create(ctx, caikitIsvc)).Should(Succeed())
		})

		It("when InferenceService specifies a caikit runtime, controller should create a PeerAuthentication and NetworkPolicy to scrape metrics", func() {

			By("By checking that the controller has created the Route")
			PeerAuth := &istiosecurityv1beta1.PeerAuthentication{}
			Eventually(func() error {
				key := types.NamespacedName{Name: inferenceService.Name, Namespace: inferenceService.Namespace}
				return cli.Get(ctx, key, route)
			}, timeout, interval).ShouldNot(HaveOccurred())

			expectedPeerAuth := &istiosecurityv1beta1.PeerAuthentication{}
			err = convertToStructuredResource(ExpectedPeerAuthPath, expectedPeerAuth, opts)
			Expect(err).NotTo(HaveOccurred())

			Expect(CompareInferenceServiceRoutes(*route, *expectedRoute)).Should(BeTrue())
		})

		It("when InferenceService does not specifies a runtime, should automatically pick a runtime and create a Route", func() {
			inferenceService := &inferenceservicev1.InferenceService{}
			err := convertToStructuredResource(InferenceServiceNoRuntime, inferenceService, opts)
			Expect(err).NotTo(HaveOccurred())
			Expect(cli.Create(ctx, inferenceService)).Should(Succeed())

			route := &routev1.Route{}
			Eventually(func() error {
				key := types.NamespacedName{Name: inferenceService.Name, Namespace: inferenceService.Namespace}
				return cli.Get(ctx, key, route)
			}, timeout, interval).ShouldNot(HaveOccurred())

			expectedRoute := &routev1.Route{}
			err = convertToStructuredResource(ExpectedRouteNoRuntimePath, expectedRoute, opts)
			Expect(err).NotTo(HaveOccurred())

			Expect(CompareInferenceServiceRoutes(*route, *expectedRoute)).Should(BeTrue())
		})
	})
})
