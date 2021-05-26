package helper

import (
	"context"
	openshiftApi "github.com/openshift/api/apps/v1"
	openshiftClient "github.com/openshift/client-go/apps/clientset/versioned/typed/apps/v1"
	k8sApi "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sCLient "k8s.io/client-go/kubernetes/typed/apps/v1"
)

func GetDeployment(client k8sCLient.AppsV1Client, name, namespace string) (*k8sApi.Deployment, error) {
	d, err := client.
		Deployments(namespace).
		Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return d, nil
}

func IsDeploymentReady(client k8sCLient.AppsV1Client, name, namespace string) (bool, error) {
	d, err := GetDeployment(client, name, namespace)
	if err != nil {
		return false, err
	}

	if d.Status.UpdatedReplicas == 1 && d.Status.AvailableReplicas == 1 {
		return true, nil
	}

	return false, nil
}

func GetDeploymentConfig(client openshiftClient.AppsV1Client, name, namespace string) (*openshiftApi.DeploymentConfig, error) {
	dc, err := client.
		DeploymentConfigs(namespace).
		Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return dc, nil
}

func IsDeploymentConfigReady(client openshiftClient.AppsV1Client, name, namespace string) (bool, error) {
	dc, err := GetDeploymentConfig(client, name, namespace)
	if err != nil {
		return false, err
	}

	if dc.Status.UpdatedReplicas == 1 && dc.Status.AvailableReplicas == 1 {
		return true, nil
	}

	return false, nil
}
