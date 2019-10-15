package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/epmd-edp/admin-console-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epmd-edp/admin-console-operator/v2/pkg/client/admin_console"
	adminConsoleSpec "github.com/epmd-edp/admin-console-operator/v2/pkg/service/admin_console/spec"
	platformHelper "github.com/epmd-edp/admin-console-operator/v2/pkg/service/platform/helper"
	keycloakV1Api "github.com/epmd-edp/keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/pkg/errors"
	appsV1Api "k8s.io/api/apps/v1"
	coreV1Api "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	authV1Api "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	appsV1Client "k8s.io/client-go/kubernetes/typed/apps/v1"
	coreV1Client "k8s.io/client-go/kubernetes/typed/core/v1"
	extensionsV1Client "k8s.io/client-go/kubernetes/typed/extensions/v1beta1"
	authV1Client "k8s.io/client-go/kubernetes/typed/rbac/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"strconv"
)

var log = logf.Log.WithName("platform")

type K8SService struct {
	Scheme                *runtime.Scheme
	CoreClient            coreV1Client.CoreV1Client
	ExtensionsV1Client    extensionsV1Client.ExtensionsV1beta1Client
	EdpClient             admin_console.EdpV1Client
	k8sUnstructuredClient client.Client
	AppsClient            appsV1Client.AppsV1Client
	AuthClient            authV1Client.RbacV1Client
}

func (service K8SService) AddServiceAccToSecurityContext(scc string, ac v1alpha1.AdminConsole) error {
	return nil
}

func (service K8SService) CreateDeployConf(ac v1alpha1.AdminConsole, url string) error {

	dbEnabled := "false"
	keycloakEnabled := "false"
	var replicaCount int32 = 1

	labels := platformHelper.GenerateLabels(ac.Name)
	consoleDeployObj := &appsV1Api.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ac.Name,
			Namespace: ac.Namespace,
			Labels:    labels,
		},

		Spec: appsV1Api.DeploymentSpec{
			Replicas: &replicaCount,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: coreV1Api.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: coreV1Api.PodSpec{
					Containers: []coreV1Api.Container{
						{
							Name:            ac.Name,
							Image:           ac.Spec.Image + ":" + ac.Spec.Version,
							ImagePullPolicy: coreV1Api.PullAlways,
							Env: []coreV1Api.EnvVar{
								{
									Name: "NAMESPACE",
									ValueFrom: &coreV1Api.EnvVarSource{
										FieldRef: &coreV1Api.ObjectFieldSelector{
											FieldPath: "metadata.namespace",
										},
									},
								},
								{
									Name:  "HOST",
									Value: url,
								},
								{
									Name:  "EDP_ADMIN_CONSOLE_VERSION",
									Value: ac.Spec.Version,
								},
								{
									Name:  "DB_ENABLED",
									Value: dbEnabled,
								},
								{
									Name:  "EDP_VERSION",
									Value: ac.Spec.EdpSpec.Version,
								},
								{
									Name:  "AUTH_KEYCLOAK_ENABLED",
									Value: keycloakEnabled,
								},
								{
									Name:  "DNS_WILDCARD",
									Value: ac.Spec.EdpSpec.DnsWildcard,
								},
								{
									Name: "PG_USER",
									ValueFrom: &coreV1Api.EnvVarSource{
										SecretKeyRef: &coreV1Api.SecretKeySelector{
											LocalObjectReference: coreV1Api.LocalObjectReference{
												Name: "admin-console-db",
											},
											Key: "username",
										},
									},
								},
								{
									Name: "PG_PASSWORD",
									ValueFrom: &coreV1Api.EnvVarSource{
										SecretKeyRef: &coreV1Api.SecretKeySelector{
											LocalObjectReference: coreV1Api.LocalObjectReference{
												Name: "admin-console-db",
											},
											Key: "password",
										},
									},
								},
								{
									Name:  "INTEGRATION_STRATEGIES",
									Value: ac.Spec.EdpSpec.IntegrationStrategies,
								},
								{
									Name:  "BUILD_TOOLS",
									Value: "maven",
								},
							},
							Ports: []coreV1Api.ContainerPort{
								{
									ContainerPort: adminConsoleSpec.AdminConsolePort,
								},
							},
							LivenessProbe: &coreV1Api.Probe{
								FailureThreshold:    5,
								InitialDelaySeconds: 180,
								PeriodSeconds:       20,
								SuccessThreshold:    1,
								Handler: coreV1Api.Handler{
									TCPSocket: &coreV1Api.TCPSocketAction{
										Port: intstr.FromInt(adminConsoleSpec.AdminConsolePort),
									},
								},
							},
							ReadinessProbe: &coreV1Api.Probe{
								FailureThreshold:    5,
								InitialDelaySeconds: 60,
								PeriodSeconds:       20,
								SuccessThreshold:    1,
								Handler: coreV1Api.Handler{
									TCPSocket: &coreV1Api.TCPSocketAction{
										Port: intstr.FromInt(adminConsoleSpec.AdminConsolePort),
									},
								},
							},
							TerminationMessagePath: "/dev/termination-log",
							Resources: coreV1Api.ResourceRequirements{
								Requests: map[coreV1Api.ResourceName]resource.Quantity{
									coreV1Api.ResourceMemory: resource.MustParse(adminConsoleSpec.MemoryRequest),
								},
							},
						},
					},
					ServiceAccountName: ac.Name,
				},
			},
			Strategy: appsV1Api.DeploymentStrategy{
				Type: appsV1Api.RollingUpdateDeploymentStrategyType,
			},
		},
	}

	if err := controllerutil.SetControllerReference(&ac, consoleDeployObj, service.Scheme); err != nil {
		return err
	}

	consoleDeployment, err := service.AppsClient.Deployments(consoleDeployObj.Namespace).Get(consoleDeployObj.Name, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			log.V(1).Info("Creating Deployment for Admin Console",
				"Namespace", consoleDeployObj.Namespace, "Name", consoleDeployObj.Name)

			consoleDeployment, err = service.AppsClient.Deployments(consoleDeployObj.Namespace).Create(consoleDeployObj)
			if err != nil {
				return err
			}
			log.Info("Deployment has been created",
				"Namespace", consoleDeployment.Name, "Name", consoleDeployment.Name)

			return nil
		}
		return err
	}

	return nil
}

func (service K8SService) CreateSecurityContext(ac v1alpha1.AdminConsole) error {
	return nil
}

func (service K8SService) CreateUserRole(ac v1alpha1.AdminConsole) error {
	consoleRoleObject := &authV1Api.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "edp-resources-admin",
			Namespace: ac.Namespace,
		},
		Rules: []authV1Api.PolicyRule{
			{
				APIGroups: []string{"*"},
				Resources: []string{"codebases", "applicationbranches", "codebasebranches", "cdpipelines", "stages"},
				Verbs:     []string{"get", "create", "update"},
			},
		},
	}

	if err := controllerutil.SetControllerReference(&ac, consoleRoleObject, service.Scheme); err != nil {
		return err
	}

	consoleRole, err := service.AuthClient.Roles(consoleRoleObject.Namespace).Get(consoleRoleObject.Name, metav1.GetOptions{})
	if err == nil {
		return nil
	}
	if !k8serrors.IsNotFound(err) {
		return err
	}
	log.V(1).Info("Creating Role for Admin Console",
		"Namespace", consoleRoleObject.Namespace, "Name", consoleRoleObject.Name)

	consoleRole, err = service.AuthClient.Roles(consoleRoleObject.Namespace).Create(consoleRoleObject)
	if err != nil {
		return err
	}
	log.Info("Role for Admin Console created", "Namespace", consoleRole.Namespace, "Name", consoleRole.Name)
	return nil
}

func (service K8SService) CreateClusterRoleBinding(ac v1alpha1.AdminConsole, name string, binding string) error {
	acClusterBindingObject := &authV1Api.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		RoleRef: authV1Api.RoleRef{
			Kind: "ClusterRole",
			Name: binding,
		},
		Subjects: []authV1Api.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      ac.Name,
				Namespace: ac.Namespace,
			},
		},
	}

	if err := controllerutil.SetControllerReference(&ac, acClusterBindingObject, service.Scheme); err != nil {
		return err
	}

	acBinding, err := service.AuthClient.RoleBindings(ac.Namespace).Get(acClusterBindingObject.Name, metav1.GetOptions{})
	if err == nil {
		return nil
	}
	if !k8serrors.IsNotFound(err) {
		return err
	}
	log.V(1).Info("Creating a new ClusterRoleBinding for Admin Console",
		"Namespace", ac.Namespace, "Name", ac.Name)
	acBinding, err = service.AuthClient.RoleBindings(ac.Namespace).Create(acClusterBindingObject)
	if err != nil {
		return err
	}
	log.Info("ClusterRoleBinding has been created",
		"Namespace", acBinding.Namespace, "Name", acBinding.Name)
	return nil
}

func (service K8SService) CreateRoleBinding(ac v1alpha1.AdminConsole, name string, binding string) error {
	acBindingObject := &authV1Api.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ac.Namespace,
		},
		RoleRef: authV1Api.RoleRef{
			Kind: "Role",
			Name: binding,
		},
		Subjects: []authV1Api.Subject{
			{
				Kind: "ServiceAccount",
				Name: ac.Name,
			},
		},
	}

	if err := controllerutil.SetControllerReference(&ac, acBindingObject, service.Scheme); err != nil {
		return err
	}

	acBinding, err := service.AuthClient.RoleBindings(acBindingObject.Namespace).Get(acBindingObject.Name, metav1.GetOptions{})

	if err == nil {
		return nil
	}
	if !k8serrors.IsNotFound(err) {
		return err
	}
	log.V(1).Info("Creating a new RoleBinding for Admin Console",
		"Namespace", ac.Namespace, "Name", ac.Name)
	acBinding, err = service.AuthClient.RoleBindings(acBindingObject.Namespace).Create(acBindingObject)
	if err != nil {
		return err
	}
	log.Info("RoleBinding has been created", "Namespace", acBinding.Namespace, "Name", acBinding.Name)
	return nil

}

func (service K8SService) GetDisplayName(ac v1alpha1.AdminConsole) (string, error) {
	return "", nil
}

func (service K8SService) GenerateDbSettings(ac v1alpha1.AdminConsole) ([]coreV1Api.EnvVar, error) {

	if !ac.Spec.DbSpec.Enabled {
		return []coreV1Api.EnvVar{}, nil
	}

	log.V(1).Info("Generating DB settings for Admin Console ",
		"Namespace", ac.Namespace, "Name", ac.Name)
	if platformHelper.ContainsEmptyString(ac.Spec.DbSpec.Name, ac.Spec.DbSpec.Hostname, ac.Spec.DbSpec.Port) {
		return nil, errors.New("One or many DB settings field are empty!")
	}

	return []coreV1Api.EnvVar{
		{
			Name:  "PG_HOST",
			Value: ac.Spec.DbSpec.Hostname,
		},
		{
			Name:  "PG_PORT",
			Value: ac.Spec.DbSpec.Port,
		},
		{
			Name:  "PG_DATABASE",
			Value: ac.Spec.DbSpec.Name,
		},
		{
			Name:  "DB_ENABLED",
			Value: strconv.FormatBool(ac.Spec.DbSpec.Enabled),
		},
	}, nil

}

func (service K8SService) GenerateKeycloakSettings(ac v1alpha1.AdminConsole, keycloakUrl string) ([]coreV1Api.EnvVar, error) {

	log.V(1).Info("Generating Keycloak settings for Admin Console",
		"Namespace", ac.Namespace, "Name", ac.Name)

	if !ac.Spec.KeycloakSpec.Enabled {
		return []coreV1Api.EnvVar{}, nil
	}

	return []coreV1Api.EnvVar{
		{
			Name: "KEYCLOAK_CLIENT_ID",
			ValueFrom: &coreV1Api.EnvVarSource{
				SecretKeyRef: &coreV1Api.SecretKeySelector{
					LocalObjectReference: coreV1Api.LocalObjectReference{
						Name: "admin-console-client",
					},
					Key: "username",
				},
			},
		},
		{
			Name: "KEYCLOAK_CLIENT_SECRET",
			ValueFrom: &coreV1Api.EnvVarSource{
				SecretKeyRef: &coreV1Api.SecretKeySelector{
					LocalObjectReference: coreV1Api.LocalObjectReference{
						Name: "admin-console-client",
					},
					Key: "password",
				},
			},
		},
		{
			Name:  "KEYCLOAK_URL",
			Value: keycloakUrl,
		},
		{
			Name:  "AUTH_KEYCLOAK_ENABLED",
			Value: strconv.FormatBool(ac.Spec.KeycloakSpec.Enabled),
		},
	}, nil
}

func (service K8SService) PatchDeploymentEnv(ac v1alpha1.AdminConsole, env []coreV1Api.EnvVar) error {
	if len(env) == 0 {
		return nil
	}

	dc, err := service.AppsClient.Deployments(ac.Namespace).Get(ac.Name, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info("Deployment not found!", "Namespace", ac.Namespace, "Name", ac.Name)
			return nil
		}
		return err
	}

	container, err := platformHelper.SelectContainer(dc.Spec.Template.Spec.Containers, ac.Name)
	if err != nil {
		return err
	}

	container.Env = platformHelper.UpdateEnv(container.Env, env)

	dc.Spec.Template.Spec.Containers = append(dc.Spec.Template.Spec.Containers, container)

	jsonDc, err := json.Marshal(dc)
	if err != nil {
		return err
	}

	_, err = service.AppsClient.Deployments(dc.Namespace).Patch(dc.Name, types.StrategicMergePatchType, jsonDc)
	if err != nil {
		return err
	}

	return err
}

func (service K8SService) GetExternalUrl(namespace string, name string) (string, string, error) {
	ingress, err := service.ExtensionsV1Client.Ingresses(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info("Ingress not found", "Namespace", namespace, "Name", name)
			return "", "", nil
		}
		return "", "", err
	}

	host := ingress.Spec.Rules[0].Host
	routeScheme := "https"

	webUrl := fmt.Sprintf("%s://%s", routeScheme, host)
	return webUrl, routeScheme, nil
}

func (service K8SService) IsDeploymentReady(instance v1alpha1.AdminConsole) (bool, error) {
	deploymentConfig, err := service.AppsClient.Deployments(instance.Namespace).Get(instance.Name, metav1.GetOptions{})
	if err != nil {
		return false, err
	}

	if deploymentConfig.Status.UpdatedReplicas == 1 && deploymentConfig.Status.AvailableReplicas == 1 {
		return true, nil
	}

	return false, nil
}

func (service K8SService) CreateSecret(ac v1alpha1.AdminConsole, name string, data map[string][]byte) error {
	labels := platformHelper.GenerateLabels(ac.Name)

	consoleSecretObject := &coreV1Api.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ac.Namespace,
			Labels:    labels,
		},
		Data: data,
		Type: "Opaque",
	}

	if err := controllerutil.SetControllerReference(&ac, consoleSecretObject, service.Scheme); err != nil {
		return err
	}

	consoleSecret, err := service.CoreClient.Secrets(consoleSecretObject.Namespace).Get(consoleSecretObject.Name, metav1.GetOptions{})

	if err != nil {
		if k8serrors.IsNotFound(err) {
			msg := fmt.Sprintf("Creating a new Secret %s/%s for Admin Console", consoleSecretObject.Namespace, consoleSecretObject.Name)
			log.V(1).Info(msg)
			consoleSecret, err = service.CoreClient.Secrets(consoleSecretObject.Namespace).Create(consoleSecretObject)
			if err != nil {
				return err
			}
			log.Info(fmt.Sprintf("Secret %s/%s has been created", consoleSecret.Namespace, consoleSecret.Name))
			// Successfully created
			return nil
		}
		// Some error occurred
		return err
	}
	// Nothing to do
	return nil
}

func (service K8SService) CreateService(ac v1alpha1.AdminConsole) error {

	labels := platformHelper.GenerateLabels(ac.Name)

	consoleServiceObject := &coreV1Api.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ac.Name,
			Namespace: ac.Namespace,
			Labels:    labels,
		},
		Spec: coreV1Api.ServiceSpec{
			Selector: labels,
			Ports: []coreV1Api.ServicePort{
				{
					TargetPort: intstr.IntOrString{StrVal: ac.Name},
					Port:       8080,
				},
			},
		},
	}

	if err := controllerutil.SetControllerReference(&ac, consoleServiceObject, service.Scheme); err != nil {
		return err
	}

	consoleService, err := service.CoreClient.Services(ac.Namespace).Get(ac.Name, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			msg := fmt.Sprintf("Creating a new service %s/%s for Admin Console %s",
				consoleServiceObject.Namespace, consoleServiceObject.Name, ac.Name)
			log.V(1).Info(msg)
			consoleService, err = service.CoreClient.Services(consoleServiceObject.Namespace).Create(consoleServiceObject)
			if err != nil {
				return err
			}
			log.Info(fmt.Sprintf("Service %s/%s has been created", consoleService.Namespace, consoleService.Name))

			return nil
		}
		return err
	}

	return nil
}

func (service K8SService) CreateServiceAccount(ac v1alpha1.AdminConsole) error {

	labels := platformHelper.GenerateLabels(ac.Name)

	consoleServiceAccountObject := &coreV1Api.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ac.Name,
			Namespace: ac.Namespace,
			Labels:    labels,
		},
	}

	if err := controllerutil.SetControllerReference(&ac, consoleServiceAccountObject, service.Scheme); err != nil {
		return err
	}

	consoleServiceAccount, err := service.CoreClient.ServiceAccounts(consoleServiceAccountObject.Namespace).Get(consoleServiceAccountObject.Name, metav1.GetOptions{})

	if err != nil {
		if k8serrors.IsNotFound(err) {
			msg := fmt.Sprintf("Creating ServiceAccount %s/%s for Admin Console %s", consoleServiceAccountObject.Namespace, consoleServiceAccountObject.Name, ac.Name)
			log.V(1).Info(msg)
			consoleServiceAccount, err = service.CoreClient.ServiceAccounts(consoleServiceAccountObject.Namespace).Create(consoleServiceAccountObject)
			if err != nil {
				return err
			}
			msg = fmt.Sprintf("ServiceAccount %s/%s has been created", consoleServiceAccount.Namespace, consoleServiceAccount.Name)
			log.Info(msg)
			// Successfully created
			return nil
		}
		// Some error occurred
		return err
	}

	// Nothing to do
	return nil
}

func (service K8SService) CreateExternalEndpoint(ac v1alpha1.AdminConsole) error {

	log.V(1).Info("Creating Admin Console external endpoint.",
		"Namespace", ac.Namespace, "Name", ac.Name)

	labels := platformHelper.GenerateLabels(ac.Name)

	consoleService, err := service.CoreClient.Services(ac.Namespace).Get(ac.Name, metav1.GetOptions{})
	if err != nil {
		log.Info("Console Service has not been found")
		return err
	}

	consoleIngressObject := &v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ac.Name,
			Namespace: ac.Namespace,
			Labels:    labels,
		},
		Spec: v1beta1.IngressSpec{
			Rules: []v1beta1.IngressRule{
				{
					Host: fmt.Sprintf("%s.%s", ac.Name, ac.Spec.EdpSpec.DnsWildcard),
					IngressRuleValue: v1beta1.IngressRuleValue{
						HTTP: &v1beta1.HTTPIngressRuleValue{
							Paths: []v1beta1.HTTPIngressPath{
								{
									Path: "/",
									Backend: v1beta1.IngressBackend{
										ServiceName: ac.Name,
										ServicePort: intstr.IntOrString{
											IntVal: consoleService.Spec.Ports[0].TargetPort.IntVal,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	if err := controllerutil.SetControllerReference(&ac, consoleIngressObject, service.Scheme); err != nil {
		return err
	}

	consoleIngress, err := service.ExtensionsV1Client.Ingresses(consoleIngressObject.Namespace).Get(consoleIngressObject.Name, metav1.GetOptions{})

	if err != nil {
		if k8serrors.IsNotFound(err) {
			log.V(1).Info("Creating a new ingress for Admin Console", "ingress", consoleIngressObject, "admin console", ac)
			consoleIngress, err = service.ExtensionsV1Client.Ingresses(consoleIngressObject.Namespace).Create(consoleIngressObject)
			if err != nil {
				return err
			}
			log.Info("Ingress has been created",
				"Namespace", consoleIngress.Namespace, "Name", consoleIngress.Name)
			return nil
		}
		return err
	}

	return nil
}

func (service K8SService) GetConfigmap(namespace string, name string) (map[string]string, error) {
	out := map[string]string{}
	configmap, err := service.CoreClient.ConfigMaps(namespace).Get(name, metav1.GetOptions{})

	if err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info(fmt.Sprintf("Config map %v in namespace %v not found", name, namespace))
			return out, nil
		}
		// Some error occurred
		return out, err
	}
	out = configmap.Data
	// Success
	return out, nil
}

func (service K8SService) GetSecret(namespace string, name string) (map[string][]byte, error) {
	out := map[string][]byte{}
	adminDBSecret, err := service.CoreClient.Secrets(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info(fmt.Sprintf("Secret %v in namespace %v not found", name, namespace))
			return nil, nil
		}
		return out, err
	}
	out = adminDBSecret.Data
	return out, nil
}

func (service K8SService) GetAdminConsole(ac v1alpha1.AdminConsole) (*v1alpha1.AdminConsole, error) {

	AdminConsoleInstance, err := service.EdpClient.Get(ac.Name, ac.Namespace, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return AdminConsoleInstance, nil
}

func (service K8SService) GetPods(namespace string) (*coreV1Api.PodList, error) {

	PodList, err := service.CoreClient.Pods(namespace).List(metav1.ListOptions{})
	if err != nil {
		return &coreV1Api.PodList{}, err
	}

	return PodList, nil
}

func (service K8SService) UpdateAdminConsole(ac v1alpha1.AdminConsole) (*v1alpha1.AdminConsole, error) {
	instance, err := service.EdpClient.Update(&ac)
	if err != nil {
		return nil, err
	}

	return instance, nil
}

func (service *K8SService) Init(config *rest.Config, scheme *runtime.Scheme, k8sClient *client.Client) error {
	coreClient, err := coreV1Client.NewForConfig(config)
	if err != nil {
		return errors.Wrap(err, "Core Client initialization failed!")
	}

	edpClient, err := admin_console.NewForConfig(config)
	if err != nil {
		return errors.Wrap(err, "EDP Client initialization failed!")
	}

	appsClient, err := appsV1Client.NewForConfig(config)
	if err != nil {
		return errors.New("appsV1 client initialization failed!")
	}

	extensionsClient, err := extensionsV1Client.NewForConfig(config)
	if err != nil {
		return errors.New("extensionsV1 client initialization failed!")
	}

	rbacClient, err := authV1Client.NewForConfig(config)
	if err != nil {
		return errors.New("extensionsV1 client initialization failed!")
	}

	service.EdpClient = *edpClient
	service.CoreClient = *coreClient
	service.Scheme = scheme
	service.k8sUnstructuredClient = *k8sClient
	service.AppsClient = *appsClient
	service.ExtensionsV1Client = *extensionsClient
	service.AuthClient = *rbacClient
	return nil
}

func (service K8SService) CreateKeycloakClient(kc *keycloakV1Api.KeycloakClient) error {
	nsn := types.NamespacedName{
		Namespace: kc.Namespace,
		Name:      kc.Name,
	}

	err := service.k8sUnstructuredClient.Get(context.TODO(), nsn, kc)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			err := service.k8sUnstructuredClient.Create(context.TODO(), kc)
			if err != nil {
				return errors.Wrapf(err, "Failed to create Keycloak client %s/%s", kc.Namespace, kc.Name)
			}
			log.Info(fmt.Sprintf("Keycloak client %s/%s created", kc.Namespace, kc.Name))
			// Successfully created
			return nil
		}
		// Some error occurred
		return errors.Wrapf(err, "Failed to create Keycloak client %s/%s", kc.Namespace, kc.Name)
	}

	// Nothing to do
	return nil
}

func (service K8SService) GetKeycloakClient(name string, namespace string) (keycloakV1Api.KeycloakClient, error) {
	out := keycloakV1Api.KeycloakClient{}
	nsn := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}

	err := service.k8sUnstructuredClient.Get(context.TODO(), nsn, &out)
	if err != nil {
		return out, err
	}

	// Success
	return out, nil
}
