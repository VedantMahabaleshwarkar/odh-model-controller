package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"

	kservev1beta1 "github.com/kserve/kserve/pkg/apis/serving/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type IsvcDeploymentMode string

var (
	Serverless    IsvcDeploymentMode = "Serverless"
	RawDeployment IsvcDeploymentMode = "RawDeployment"
	ModelMesh     IsvcDeploymentMode = "ModelMesh"
)

const (
	inferenceServiceDeploymentModeAnnotation = "serving.kserve.io/deploymentMode"
	KserveConfigMapName                      = "inferenceservice-config"
)

func IsDeploymentModeForIsvcModelMesh(isvc *kservev1beta1.InferenceService) bool {
	value, exists := isvc.Annotations[inferenceServiceDeploymentModeAnnotation]
	if exists && value == string(ModelMesh) {
		return true
	}
	return false
}

func GetDeploymentModeForIsvc(ctx context.Context, cli client.Client, isvc *kservev1beta1.InferenceService) (IsvcDeploymentMode, error) {

	// If ISVC specifically sets deployment mode using an annotation, return bool depending on value
	value, exists := isvc.Annotations[inferenceServiceDeploymentModeAnnotation]
	if exists {
		switch value {
		case string(ModelMesh):
			return ModelMesh, nil
		case string(Serverless):
			return Serverless, nil
		case string(RawDeployment):
			return RawDeployment, nil
		}
	} else {
		// ISVC does not specifically set deployment mode using an annotation, determine the default from configmap
		controllerNs, err := GetOperatorNamespace()
		if err != nil {
			return "", fmt.Errorf("cannot determine controller namespace to retrive inferenceservice-config. %w", err)
		}
		inferenceServiceConfigMap := &corev1.ConfigMap{}
		err = cli.Get(ctx, client.ObjectKey{
			Namespace: controllerNs,
			Name:      KserveConfigMapName,
		}, inferenceServiceConfigMap)
		if err != nil {
			return "", fmt.Errorf("error getting configmap 'inferenceservice-config'. %w", err)
		}
		var deployData map[string]interface{}
		if err = json.Unmarshal([]byte(inferenceServiceConfigMap.Data["deploy"]), &deployData); err != nil {
			return "", fmt.Errorf("error retrieving value for key 'deploy' from configmap %s. %w", KserveConfigMapName, err)
		}
		defaultDeploymentMode := deployData["defaultDeploymentMode"]
		if defaultDeploymentMode == Serverless {
			return Serverless, nil
		} else {
			return RawDeployment, nil
		}
	}
	return "", nil
}

func GetOperatorNamespace() (string, error) {
	data, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err == nil {
		if ns := strings.TrimSpace(string(data)); len(ns) > 0 {
			return ns, nil
		}
	}

	return "", err
}

func IsNil(i any) bool {
	return reflect.ValueOf(i).IsNil()
}

func IsNotNil(i any) bool {
	return !IsNil(i)
}
