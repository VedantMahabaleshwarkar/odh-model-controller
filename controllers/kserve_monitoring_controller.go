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
	"os"
	"reflect"

	"github.com/go-logr/logr"
	kservev1beta1 "github.com/kserve/kserve/pkg/apis/serving/v1beta1"
	"istio.io/api/security/v1beta1"
	istiotypes "istio.io/api/type/v1beta1"
	istiosecurityv1beta1 "istio.io/client-go/pkg/apis/security/v1beta1"
	istioclient "istio.io/client-go/pkg/clientset/versioned"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/networking/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type KserveMonitoringReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

const (
	ServiceMeshLabelKey      = "opendatahub.io/service-mesh"
	peerAuthenticationName   = "kserve-metrics"
	networkPolicyName        = "allow-from-openshift-monitoring-ns"
	userWorkloadMonitoringNS = "openshift-user-workload-monitoring"
)

// modelMeshEnabled return true if this Namespace is modelmesh enabled

// monitoringThisNameSpace return true if this Namespace should be monitored by monitoring stack
// returns true if this Namespace is kserve stack enabled
func (r *KserveMonitoringReconciler) monitoringThisNameSpace(ns string, annotations map[string]string) bool {
	enabled, ok := annotations["opendatahub.io/service-mesh"]
	if !ok || enabled != "enabled" {
		return false
	}
	return true
}

func (r *KserveMonitoringReconciler) foundPeerAuth(ctx context.Context, supportedFormat string, ns string) (bool, *istiosecurityv1beta1.PeerAuthentication, error) {

	peerAuth := &istiosecurityv1beta1.PeerAuthentication{}
	namespacedName := types.NamespacedName{
		Name:      supportedFormat + "-metrics",
		Namespace: ns,
	}
	err := r.Client.Get(ctx, namespacedName, peerAuth)
	if apierrs.IsNotFound(err) {
		return false, nil, nil
	} else if err != nil {
		r.Log.Error(err, "Failed to get Peer Authentication for supported model format "+supportedFormat)
		return false, nil, err
	}
	return true, peerAuth, nil
}

func buildDesiredPeerAuth(ctx context.Context, supportedFormat string, ns string, isvcName string) *istiosecurityv1beta1.PeerAuthentication {
	desiredPeerAuth := &istiosecurityv1beta1.PeerAuthentication{
		ObjectMeta: metav1.ObjectMeta{
			Name:      supportedFormat + "-metrics",
			Namespace: ns,
			Labels:    map[string]string{"opendatahub.io/managed": "true"},
		},
		Spec: v1beta1.PeerAuthentication{
			Selector: &istiotypes.WorkloadSelector{
				MatchLabels: map[string]string{
					"serving.knative.dev/service": isvcName + "predictor-default",
				},
			},
			Mtls: &v1beta1.PeerAuthentication_MutualTLS{Mode: 3},
		},
	}

	switch supportedFormat {
	case "caikit":
		desiredPeerAuth.Spec.PortLevelMtls = map[uint32]*v1beta1.PeerAuthentication_MutualTLS{
			8086: {Mode: 2},
		}
	default:
		//r.Log.Info("Failed to get supportedFormat for Inference Service: " + inferenceService.Name)
	}
	return desiredPeerAuth
}

func arePeerAuthsEqual(desiredPeerAuth *istiosecurityv1beta1.PeerAuthentication, foundPeerAuth *istiosecurityv1beta1.PeerAuthentication) bool {
	areEqual :=
		reflect.DeepEqual(desiredPeerAuth.ObjectMeta, desiredPeerAuth.ObjectMeta.Labels) &&
			reflect.DeepEqual(desiredPeerAuth.Spec.PortLevelMtls, foundPeerAuth.Spec.PortLevelMtls)
	return areEqual
}

func areNetworkPoliciesEqual(networkPolicy *v1.NetworkPolicy, desiredNetWorkPolicy *v1.NetworkPolicy) bool {
	areEqual :=
		reflect.DeepEqual(networkPolicy.ObjectMeta, desiredNetWorkPolicy.ObjectMeta.Labels) &&
			reflect.DeepEqual(networkPolicy.Spec.Ingress, desiredNetWorkPolicy.Spec.Ingress)
	return areEqual
}

// createRBIfDNE will attempt to create desiredRB if it does not exist, or is different from actualRB
func (r *KserveMonitoringReconciler) createPeerAuth(ctx context.Context, desiredPeerAuth *istiosecurityv1beta1.PeerAuthentication, ns string) error {
	//setup go client
	kubeconfig := os.Getenv("KUBECONFIG")
	restConfig, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		r.Log.Error(err, "Failed to create k8s rest client: %s", err)
		return err
	}

	ic, err := istioclient.NewForConfig(restConfig)
	if err != nil {
		r.Log.Error(err, "Failed to create istio client: %s", err)
		return err
	}
	_, err = ic.SecurityV1beta1().PeerAuthentications(ns).Create(ctx, desiredPeerAuth, metav1.CreateOptions{})
	if err != nil {
		r.Log.Error(err, "Failed to create PeerAuthentication", desiredPeerAuth.Name)
		return err
	}
	return nil
}

func (r *KserveMonitoringReconciler) reconcilePeerAuthentication(ctx context.Context, req ctrl.Request, inferenceService *kservev1beta1.InferenceService) error {

	supportedFormat := inferenceService.Spec.Predictor.Model.ModelFormat.Name
	switch supportedFormat {
	case "caikit":
		desiredPeerAuth := buildDesiredPeerAuth(ctx, "caikit", req.Namespace, inferenceService.Name)
		peerAuthExists, foundPeerAuth, err := r.foundPeerAuth(ctx, "caikit", req.Namespace)
		if err != nil {
			return err
		}
		if !peerAuthExists && (!arePeerAuthsEqual(desiredPeerAuth, foundPeerAuth)) {
			err = r.createPeerAuth(ctx, desiredPeerAuth, req.Namespace)
		}
	default:
		r.Log.Info("Failed to get supportedFormat for Inference Service: " + inferenceService.Name)
	}
	return nil

}

func (r *KserveMonitoringReconciler) reconcileNetWorkPolicy(ctx context.Context, req ctrl.Request, inferenceService *kservev1beta1.InferenceService) error {

	foundNetWorkPolicy := false
	networkPolicy := &v1.NetworkPolicy{}
	namespacednetworkPolicy := types.NamespacedName{
		Name:      networkPolicyName,
		Namespace: req.Namespace,
	}
	err := r.Client.Get(ctx, namespacednetworkPolicy, networkPolicy)
	if apierrs.IsNotFound(err) {
		foundNetWorkPolicy = false
	} else if err != nil {
		foundNetWorkPolicy = false
	}
	desiredNetWorkPolicy := &v1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      networkPolicyName,
			Namespace: req.Namespace,
		},
		Spec: v1.NetworkPolicySpec{
			Ingress: []v1.NetworkPolicyIngressRule{
				{
					From: []v1.NetworkPolicyPeer{
						{
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"name": userWorkloadMonitoringNS,
								},
							},
						},
					},
				},
			},
		},
	}
	if !foundNetWorkPolicy && (!areNetworkPoliciesEqual(networkPolicy, desiredNetWorkPolicy)) {
		err = r.Create(ctx, desiredNetWorkPolicy)
		if err != nil {
			return err
		}
	}
	return nil
}

// Reconcile will manage the creation, update and deletion of the namespace level monitoring resources
func (r *KserveMonitoringReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Initialize logger format
	log := r.Log.WithValues("ResourceName", req.Name, "Namespace", req.Namespace)

	ns := &corev1.Namespace{}
	namespacedName := types.NamespacedName{
		Name: req.Namespace,
	}
	err := r.Client.Get(ctx, namespacedName, ns)
	if err != nil {
		return ctrl.Result{}, err
	}
	inferenceService := &kservev1beta1.InferenceService{}

	err = r.Client.Get(ctx, req.NamespacedName, inferenceService)
	if apierrs.IsNotFound(err) {
		return ctrl.Result{}, err
	} else if err != nil {
		r.Log.Error(err, "Failed to get Inference Service "+inferenceService.Name)
		return ctrl.Result{}, err
	}

	log.Info("Kserve Monitoring Controller reconciling.")
	err = r.reconcilePeerAuthentication(ctx, req, inferenceService)
	if err != nil {
		return ctrl.Result{}, err
	}
	err = r.reconcileNetWorkPolicy(ctx, req, inferenceService)
	if err != nil {
		return ctrl.Result{}, err
	}
	log.Info("Kserve Monitoring Controller reconciled successfully.")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *KserveMonitoringReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kservev1beta1.InferenceService{}).
		Owns(&istiosecurityv1beta1.PeerAuthentication{}).
		Owns(&v1.NetworkPolicy{}).
		Complete(r)
}
