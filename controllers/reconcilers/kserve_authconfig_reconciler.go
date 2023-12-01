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

	"github.com/go-logr/logr"
	kservev1beta1 "github.com/kserve/kserve/pkg/apis/serving/v1beta1"
	authorinov1beta2 "github.com/kuadrant/authorino/api/v1beta2"
	"github.com/opendatahub-io/odh-model-controller/controllers/comparators"
	"github.com/opendatahub-io/odh-model-controller/controllers/processors"
	"github.com/opendatahub-io/odh-model-controller/controllers/resources"
	"github.com/pkg/errors"
	k8serror "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type KserveAuthConfigReconciler struct {
	client         client.Client
	scheme         *runtime.Scheme
	deltaProcessor processors.DeltaProcessor
	detector       resources.AuthTypeDetector
	store          resources.AuthConfigStore
	templateLoader resources.AuthConfigTemplateLoader
	hostExtractor  resources.InferenceServiceHostExtractor
}

func NewKserveAuthConfigReconciler(client client.Client, scheme *runtime.Scheme) *KserveAuthConfigReconciler {
	return &KserveAuthConfigReconciler{
		client:         client,
		scheme:         scheme,
		deltaProcessor: processors.NewDeltaProcessor(),
		detector:       resources.NewKServeAuthTypeDetector(client),
		store:          resources.NewClientAuthConfigStore(client),
		templateLoader: resources.NewConfigMapTemplateLoader(client, resources.NewStaticTemplateLoader()),
		hostExtractor:  resources.NewKServeInferenceServiceHostExtractor(),
	}
}

func (r *KserveAuthConfigReconciler) Reconcile(ctx context.Context, log logr.Logger, isvc *kservev1beta1.InferenceService) error {

	if isvc.Status.URL == nil {
		log.V(1).Info("Inference Service not ready yet, waiting for URL")
		return nil
	}

	log.V(1).Info("create desired state")
	desiredState, err := r.createDesiredResource(ctx, isvc)
	if err != nil {
		return err
	}

	log.V(1).Info("get existing state")
	existingState, err := r.getExistingResource(ctx, isvc)
	if err != nil && !k8serror.IsNotFound(err) {
		return err
	}

	// Process Delta
	log.V(1).Info("process delta")
	if err = r.processDelta(ctx, log, desiredState, existingState); err != nil {
		return err
	}
	return nil

}

func (r *KserveAuthConfigReconciler) createDesiredResource(ctx context.Context, isvc *kservev1beta1.InferenceService) (*authorinov1beta2.AuthConfig, error) {
	typeName := types.NamespacedName{
		Name:      isvc.GetName(),
		Namespace: isvc.GetNamespace(),
	}

	authType, err := r.detector.Detect(ctx, isvc)
	if err != nil {
		return nil, errors.Wrapf(err, "could not detect AuthType for InferenceService %s", typeName)
	}
	template, err := r.templateLoader.Load(ctx, authType, typeName)
	if err != nil {
		return nil, errors.Wrapf(err, "could not load template for AuthType %s for InferenceService %s", authType, typeName)
	}

	template.Name = typeName.Name
	template.Namespace = typeName.Namespace
	template.Spec.Hosts = r.hostExtractor.Extract(isvc)
	if template.Labels == nil {
		template.Labels = map[string]string{}
	}
	template.Labels["security.opendatahub.io/authorization-group"] = "default"

	ctrl.SetControllerReference(isvc, &template, r.scheme)

	return &template, nil
}

func (r *KserveAuthConfigReconciler) getExistingResource(ctx context.Context, isvc *kservev1beta1.InferenceService) (*authorinov1beta2.AuthConfig, error) {
	typeName := types.NamespacedName{
		Name:      isvc.GetName(),
		Namespace: isvc.GetNamespace(),
	}
	return r.store.Get(ctx, typeName)
}

func (r *KserveAuthConfigReconciler) processDelta(ctx context.Context, log logr.Logger, desiredState *authorinov1beta2.AuthConfig, existingState *authorinov1beta2.AuthConfig) (err error) {
	comparator := comparators.GetAuthConfigComparator()
	delta := r.deltaProcessor.ComputeDelta(comparator, desiredState, existingState)

	if !delta.HasChanges() {
		log.V(1).Info("No delta found")
		return nil
	}

	if delta.IsAdded() {
		log.V(1).Info("Delta found", "create", desiredState.GetName())
		return errors.Wrapf(
			r.store.Create(ctx, desiredState),
			"could not store AuthConfig %s for InferenceService %s", desiredState.Name, desiredState.Name)
	}
	if delta.IsUpdated() {
		log.V(1).Info("Delta found", "update", desiredState.GetName())
		rp := existingState.DeepCopy()
		rp.Spec = desiredState.Spec
		rp.Labels = desiredState.Labels
		return errors.Wrapf(
			r.store.Update(ctx, rp),
			"could not store AuthConfig %s for InferenceService %s", desiredState.Name, desiredState.Name)
	}
	if delta.IsRemoved() {
		log.V(1).Info("Delta found", "delete", existingState.GetName())
		return errors.Wrapf(
			r.store.Remove(ctx, types.NamespacedName{Namespace: existingState.Namespace, Name: existingState.Name}),
			"could not remove AuthConfig %s for InferenceService %s", existingState.Name, existingState.Name)
	}
	return nil
}

func (r *KserveAuthConfigReconciler) Remove(ctx context.Context, log logr.Logger, isvc *kservev1beta1.InferenceService) error {
	typeName := types.NamespacedName{
		Name:      isvc.GetName(),
		Namespace: isvc.GetNamespace(),
	}
	return r.store.Remove(ctx, typeName)
}
