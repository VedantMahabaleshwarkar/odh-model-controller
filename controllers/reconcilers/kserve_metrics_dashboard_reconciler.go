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

package reconcilers

import (
	"context"
	"encoding/json"
	"os"
	"regexp"
	"strings"

	"github.com/go-logr/logr"
	kservev1alpha1 "github.com/kserve/kserve/pkg/apis/serving/v1alpha1"
	kservev1beta1 "github.com/kserve/kserve/pkg/apis/serving/v1beta1"
	"github.com/opendatahub-io/odh-model-controller/controllers/comparators"
	"github.com/opendatahub-io/odh-model-controller/controllers/constants"
	"github.com/opendatahub-io/odh-model-controller/controllers/processors"
	"github.com/opendatahub-io/odh-model-controller/controllers/resources"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Graph struct {
	Title string `json:"Title"`
	Query string `json:"Query"`
}

type Panels struct {
	Graphs []Graph `json:"Graphs"`
}

type MetricsDashboardConfigMapData struct {
	Data map[string]Panels `json:""`
}

var _ SubResourceReconciler = (*KserveMetricsDashboardReconciler)(nil)
var ovmsData []byte
var tgisData []byte
var vllmData []byte

type KserveMetricsDashboardReconciler struct {
	NoResourceRemoval
	client           client.Client
	telemetryHandler resources.ConfigMapHandler
	deltaProcessor   processors.DeltaProcessor
}

func NewKserveMetricsDashboardReconciler(client client.Client) *KserveMetricsDashboardReconciler {
	return &KserveMetricsDashboardReconciler{
		client:           client,
		telemetryHandler: resources.NewConfigMapHandler(client),
		deltaProcessor:   processors.NewDeltaProcessor(),
	}
}

func (r *KserveMetricsDashboardReconciler) Reconcile(ctx context.Context, log logr.Logger, isvc *kservev1beta1.InferenceService) error {

	// Create Desired resource
	desiredResource, err := r.createDesiredResource(ctx, log, isvc)
	if err != nil {
		return err
	}

	// Get Existing resource
	existingResource, err := r.getExistingResource(ctx, log, isvc)
	if err != nil {
		return err
	}

	// Process Delta
	if err = r.processDelta(ctx, log, desiredResource, existingResource); err != nil {
		return err
	}
	return nil
}

func (r *KserveMetricsDashboardReconciler) createDesiredResource(ctx context.Context, log logr.Logger, isvc *kservev1beta1.InferenceService) (*corev1.ConfigMap, error) {

	isvcRuntime := isvc.Spec.Predictor.Model.Runtime
	runtime := &kservev1alpha1.ServingRuntime{}
	if err := r.client.Get(ctx, types.NamespacedName{Name: *isvcRuntime, Namespace: isvc.Namespace}, runtime); err != nil {
		log.Error(err, "Could not determine servingruntime for isvc")
	}

	runtimeImage := runtime.Spec.Containers[0].Image
	runtimeImageName := extractImageName(runtimeImage)

	var templatedData []byte
	switch runtimeImageName {
	case constants.Tgis:
		if tgisData == nil {
			data, err := os.ReadFile("tgis-metrics.json")
			if err != nil {
				log.Error(err, "Unable to load metrics dashboard template file")
			}
			tgisData = data
		}
		templatedData = tgisData
	case constants.Ovms:
		if ovmsData == nil {
			data, err := os.ReadFile("ovms-metrics.json")
			if err != nil {
				log.Error(err, "Unable to load metrics dashboard template file")
			}
			ovmsData = data
		}
		templatedData = ovmsData
	case constants.Vllm:
		if vllmData == nil {
			data, err := os.ReadFile("vllm-metrics.json")
			if err != nil {
				log.Error(err, "Unable to load metrics dashboard template file")
			}
			vllmData = data
		}
		templatedData = vllmData
	}

	var configMapData MetricsDashboardConfigMapData
	data := substituteVariablesInQueries(templatedData, isvc.Namespace, isvc.Name)
	err := json.Unmarshal(data, &configMapData)
	if err != nil {
		log.Error(err, "Unable to load metrics dashboard templates")
	}
	log.V(1).Info("data", "value", string(data))
	// Create ConfigMap object
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      isvc.Name + "-metrics-dashboard",
			Namespace: isvc.Namespace,
		},
		Data: map[string]string{
			"Data": string(data),
		},
	}
	if err := ctrl.SetControllerReference(isvc, configMap, r.client.Scheme()); err != nil {
		log.Error(err, "Unable to add OwnerReference to the Metrics Dashboard Configmap")
		return nil, err
	}

	return configMap, nil
}

func (r *KserveMetricsDashboardReconciler) getExistingResource(ctx context.Context, log logr.Logger, isvc *kservev1beta1.InferenceService) (*corev1.ConfigMap, error) {
	configMap := &corev1.ConfigMap{}
	err := r.client.Get(ctx, types.NamespacedName{Name: isvc.Name + "-metrics-dashboard", Namespace: isvc.Namespace}, configMap)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, nil // ConfigMap doesn't exist
		}
		log.Error(err, "Failed to get existing ConfigMap")
		return nil, err
	}
	return configMap, nil
}

func (r *KserveMetricsDashboardReconciler) processDelta(ctx context.Context, log logr.Logger, desiredResource *corev1.ConfigMap, existingResource *corev1.ConfigMap) (err error) {

	comparator := comparators.GetConfigMapComparator()
	delta := r.deltaProcessor.ComputeDelta(comparator, desiredResource, existingResource)
	if !delta.HasChanges() {
		log.V(1).Info("No delta found in metrics configmap")
		return nil
	}

	if delta.IsAdded() {
		log.V(1).Info("Delta found", "create", desiredResource.GetName())
		if err = r.client.Create(ctx, desiredResource); err != nil {
			return
		}
	}
	if delta.IsUpdated() {
		log.V(1).Info("Delta found", "update", existingResource.GetName())
		rp := existingResource.DeepCopy()
		rp.Labels = desiredResource.Labels
		rp.Data = desiredResource.Data

		if err = r.client.Update(ctx, rp); err != nil {
			return
		}
	}
	if delta.IsRemoved() {
		log.V(1).Info("Delta found", "delete", existingResource.GetName())
		if err = r.client.Delete(ctx, existingResource); err != nil {
			return
		}
	}
	return nil
}

func extractImageName(image string) string {
	r := regexp.MustCompile(`.*/(.+?)(:|@).*`)
	matches := r.FindStringSubmatch(image)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func substituteVariablesInQueries(data []byte, namespace string, name string) []byte {
	stringData := string(data)
	stringDatawithNS := strings.Replace(stringData, "${namespace}", namespace, -1)
	stringDataComplete := strings.Replace(stringDatawithNS, "${model_name}", name, -1)
	return []byte(stringDataComplete)
}
