package controllers

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	kservev1alpha1 "github.com/kserve/kserve/pkg/apis/serving/v1alpha1"
	kservev1beta1 "github.com/kserve/kserve/pkg/apis/serving/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

const (
	KserveOvmsInferenceServiceName    = "mnist"
	KserveSklearnInferenceServiceName = "sklearn-v2-iris"
	ConfigMapNameSuffix               = "-metrics-dashboard"
	ExpectedOvmsConfigMapPath         = "./testdata/results/mnist-ovms-metrics-dashboard.yaml"
	SklearnServerInferenceServicePath = "./testdata/deploy/kserve-sklearnserver-inference-service.yaml"
	SklearnServerServingRuntimePath   = "./testdata/deploy/kserve-sklearnserver-serving-runtime.yaml"
)

var _ = Describe("The KServe mesh reconciler", func() {
	var testNs string

	createServingRuntime := func(namespace, path string) *kservev1alpha1.ServingRuntime {
		servingRuntime := &kservev1alpha1.ServingRuntime{}
		err := convertToStructuredResource(path, servingRuntime)
		Expect(err).NotTo(HaveOccurred())
		servingRuntime.SetNamespace(namespace)
		if err := cli.Create(ctx, servingRuntime); err != nil && !errors.IsAlreadyExists(err) {
			Fail(err.Error())
		}
		return servingRuntime
	}

	createInferenceService := func(namespace, name string, path string) *kservev1beta1.InferenceService {
		inferenceService := &kservev1beta1.InferenceService{}
		err := convertToStructuredResource(path, inferenceService)
		Expect(err).NotTo(HaveOccurred())
		inferenceService.SetNamespace(namespace)
		if len(name) != 0 {
			inferenceService.Name = name
		}
		if err := cli.Create(ctx, inferenceService); err != nil && !errors.IsAlreadyExists(err) {
			Fail(err.Error())
		}
		return inferenceService
	}

	BeforeEach(func() {
		testNamespace := Namespaces.Create(cli)
		testNs = testNamespace.Name

		inferenceServiceConfig := &corev1.ConfigMap{}
		Expect(convertToStructuredResource(InferenceServiceConfigPath1, inferenceServiceConfig)).To(Succeed())
		if err := cli.Create(ctx, inferenceServiceConfig); err != nil && !errors.IsAlreadyExists(err) {
			Fail(err.Error())
		}

	})

	When("deploying a Kserve model", func() {
		It("if the runtime is supported for metrics, it should create a configmap with prometheus queries", func() {
			_ = createServingRuntime(testNs, KserveServingRuntimePath1)
			_ = createInferenceService(testNs, KserveOvmsInferenceServiceName, KserveInferenceServicePath1)

			metricsConfigMap, err := waitForConfigMap(cli, WorkingNamespace, KserveOvmsInferenceServiceName+ConfigMapNameSuffix, 30*time.Second)
			Expect(err).NotTo(HaveOccurred())
			Expect(metricsConfigMap).NotTo(BeNil())

			expectedmetricsConfigMap := &corev1.ConfigMap{}
			err = convertToStructuredResource(ExpectedOvmsConfigMapPath, expectedmetricsConfigMap)
			Expect(err).NotTo(HaveOccurred())
			Expect(compareConfigMap(metricsConfigMap, expectedmetricsConfigMap)).Should((BeTrue()))
		})

		It("if the runtime is not supported for metrics, it should create a configmap with the unsupported config", func() {
			_ = createServingRuntime(testNs, SklearnServerServingRuntimePath)
			_ = createInferenceService(testNs, KserveSklearnInferenceServiceName, SklearnServerInferenceServicePath)

			metricsConfigMap, err := waitForConfigMap(cli, WorkingNamespace, KserveSklearnInferenceServiceName+ConfigMapNameSuffix, 30*time.Second)
			Expect(err).NotTo(HaveOccurred())
			Expect(metricsConfigMap).NotTo(BeNil())

			expectedmetricsConfigMap := &corev1.ConfigMap{}
			err = convertToStructuredResource(ExpectedOvmsConfigMapPath, expectedmetricsConfigMap)
			Expect(err).NotTo(HaveOccurred())
			Expect(compareConfigMap(metricsConfigMap, expectedmetricsConfigMap)).Should((BeTrue()))
		})
	})

	When("deleting the deployed models", func() {
		It("it should delete the associated configmap", func() {
			_ = createServingRuntime(testNs, KserveServingRuntimePath1)
			OvmsInferenceService := createInferenceService(testNs, KserveOvmsInferenceServiceName, KserveInferenceServicePath1)

			Expect(cli.Delete(ctx, OvmsInferenceService)).Should(Succeed())
			Eventually(func() error {
				configmap := &corev1.ConfigMap{}
				key := types.NamespacedName{Name: KserveOvmsInferenceServiceName + ConfigMapNameSuffix, Namespace: OvmsInferenceService.Namespace}
				err := cli.Get(ctx, key, configmap)
				return err
			}, timeout, interval).ShouldNot(Succeed())

			_ = createServingRuntime(testNs, SklearnServerServingRuntimePath)
			SklearnInferenceService := createInferenceService(testNs, KserveSklearnInferenceServiceName, SklearnServerInferenceServicePath)

			Expect(cli.Delete(ctx, SklearnInferenceService)).Should(Succeed())
			Eventually(func() error {
				configmap := &corev1.ConfigMap{}
				key := types.NamespacedName{Name: KserveOvmsInferenceServiceName + ConfigMapNameSuffix, Namespace: SklearnInferenceService.Namespace}
				err := cli.Get(ctx, key, configmap)
				return err
			}, timeout, interval).ShouldNot(Succeed())
		})
	})
})
