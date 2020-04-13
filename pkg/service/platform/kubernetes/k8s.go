package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/epmd-edp/admin-console-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epmd-edp/admin-console-operator/v2/pkg/client/admin_console"
	adminConsoleSpec "github.com/epmd-edp/admin-console-operator/v2/pkg/service/admin_console/spec"
	platformHelper "github.com/epmd-edp/admin-console-operator/v2/pkg/service/platform/helper"
	edpCompApi "github.com/epmd-edp/edp-component-operator/pkg/apis/v1/v1alpha1"
	edpCompClient "github.com/epmd-edp/edp-component-operator/pkg/client"
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
	edpCompClient         edpCompClient.EDPComponentV1Client
}

func (service K8SService) CreateDeployConf(ac v1alpha1.AdminConsole, url string) error {

	k := "false"
	t := true
	f := false
	var rc int32 = 1
	var id int64 = 1001

	basePath := ""
	if len(ac.Spec.BasePath) != 0 {
		basePath = fmt.Sprintf("/%v", ac.Spec.BasePath)
	}

	l := platformHelper.GenerateLabels(ac.Name)
	do := &appsV1Api.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ac.Name,
			Namespace: ac.Namespace,
			Labels:    l,
		},

		Spec: appsV1Api.DeploymentSpec{
			Replicas: &rc,
			Selector: &metav1.LabelSelector{
				MatchLabels: l,
			},
			Template: coreV1Api.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: l,
				},
				Spec: coreV1Api.PodSpec{
					ImagePullSecrets: ac.Spec.ImagePullSecrets,
					Containers: []coreV1Api.Container{
						{
							SecurityContext: &coreV1Api.SecurityContext{
								Privileged:               &f,
								ReadOnlyRootFilesystem:   &t,
								AllowPrivilegeEscalation: &f,
							},
							Name:            ac.Name,
							Image:           fmt.Sprintf("%s:%s", ac.Spec.Image, ac.Spec.Version),
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
									Name:  "BASE_PATH",
									Value: basePath,
								},
								{
									Name:  "EDP_ADMIN_CONSOLE_VERSION",
									Value: ac.Spec.Version,
								},
								{
									Name: "EDP_VERSION",
									ValueFrom: &coreV1Api.EnvVarSource{
										ConfigMapKeyRef: &coreV1Api.ConfigMapKeySelector{
											LocalObjectReference: coreV1Api.LocalObjectReference{
												Name: "edp-config",
											},
											Key: "edp_version",
										},
									},
								},
								{
									Name:  "AUTH_KEYCLOAK_ENABLED",
									Value: k,
								},
								{
									Name: "DNS_WILDCARD",
									ValueFrom: &coreV1Api.EnvVarSource{
										ConfigMapKeyRef: &coreV1Api.ConfigMapKeySelector{
											LocalObjectReference: coreV1Api.LocalObjectReference{
												Name: "edp-config",
											},
											Key: "dns_wildcard",
										},
									},
								},
								{
									Name: "PG_USER",
									ValueFrom: &coreV1Api.EnvVarSource{
										SecretKeyRef: &coreV1Api.SecretKeySelector{
											LocalObjectReference: coreV1Api.LocalObjectReference{
												Name: "db-admin-console",
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
												Name: "db-admin-console",
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
								{
									Name:  "DEPLOYMENT_SCRIPT",
									Value: "helm-chart,openshift-template",
								},
								{
									Name:  "PLATFORM_TYPE",
									Value: "kubernetes",
								},
								{
									Name:  "VERSIONING_TYPES",
									Value: "default,edp",
								},
								{
									Name: "VCS_INTEGRATION_ENABLED",
									ValueFrom: &coreV1Api.EnvVarSource{
										ConfigMapKeyRef: &coreV1Api.ConfigMapKeySelector{
											LocalObjectReference: coreV1Api.LocalObjectReference{
												Name: "edp-config",
											},
											Key: "vcs_integration_enabled",
										},
									},
								},
								{
									Name: "EDP_NAME",
									ValueFrom: &coreV1Api.EnvVarSource{
										ConfigMapKeyRef: &coreV1Api.ConfigMapKeySelector{
											LocalObjectReference: coreV1Api.LocalObjectReference{
												Name: "edp-config",
											},
											Key: "edp_name",
										},
									},
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
					SecurityContext: &coreV1Api.PodSecurityContext{
						RunAsUser:    &id,
						RunAsGroup:   &id,
						RunAsNonRoot: &t,
						FSGroup:      &id,
					},
				},
			},
			Strategy: appsV1Api.DeploymentStrategy{
				Type: appsV1Api.RollingUpdateDeploymentStrategyType,
			},
		},
	}

	if err := controllerutil.SetControllerReference(&ac, do, service.Scheme); err != nil {
		return err
	}

	d, err := service.AppsClient.Deployments(do.Namespace).Get(do.Name, metav1.GetOptions{})
	if !k8serrors.IsNotFound(err) {
		return err
	}

	dbEnvVars, err := service.GenerateDbSettings(ac)
	if err != nil {
		return errors.Wrap(err, "Failed to generate environment variables for shared database!")
	}
	do.Spec.Template.Spec.Containers[0].Env = append(do.Spec.Template.Spec.Containers[0].Env, dbEnvVars...)

	d, err = service.AppsClient.Deployments(do.Namespace).Create(do)
	if err != nil {
		return err
	}
	log.Info("Deployment has been created",
		"Namespace", d.Name, "Name", d.Name)

	return nil
}

func (service K8SService) CreateSecurityContext(ac v1alpha1.AdminConsole) error {
	return nil
}

func (service K8SService) CreateRole(ac v1alpha1.AdminConsole) error {
	ro := &authV1Api.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "edp-resources-admin",
			Namespace: ac.Namespace,
		},
		Rules: []authV1Api.PolicyRule{
			{
				APIGroups: []string{"*"},
				Resources: []string{"codebases", "codebasebranches", "cdpipelines", "stages",
					"codebases/finalizers", "codebasebranches/finalizers", "cdpipelines/finalizers", "stages/finalizers"},
				Verbs: []string{"get", "create", "update", "delete", "patch"},
			},
		},
	}

	if err := controllerutil.SetControllerReference(&ac, ro, service.Scheme); err != nil {
		return err
	}

	r, err := service.AuthClient.Roles(ro.Namespace).Get(ro.Name, metav1.GetOptions{})
	if err == nil {
		return nil
	}
	if !k8serrors.IsNotFound(err) {
		return err
	}
	log.V(1).Info("Creating Role for Admin Console",
		"Namespace", ro.Namespace, "Name", ro.Name)

	r, err = service.AuthClient.Roles(ro.Namespace).Create(ro)
	if err != nil {
		return err
	}
	log.Info("Role for Admin Console created", "Namespace", r.Namespace, "Name", r.Name)
	return nil
}

func (service K8SService) CreateClusterRole(ac v1alpha1.AdminConsole) error {
	cro := &authV1Api.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "admin-console-sc-access",
		},
		Rules: []authV1Api.PolicyRule{
			{
				APIGroups: []string{"storage.k8s.io"},
				Resources: []string{"storageclasses"},
				Verbs:     []string{"get", "list"},
			},
		},
	}

	if err := controllerutil.SetControllerReference(&ac, cro, service.Scheme); err != nil {
		return err
	}

	cr, err := service.AuthClient.ClusterRoles().Get(cro.Name, metav1.GetOptions{})
	if err == nil {
		return nil
	}
	if !k8serrors.IsNotFound(err) {
		return err
	}
	log.V(1).Info("Creating Role for Admin Console", "Name", cro.Name, "ClusterRoleName", cro.Name)

	cr, err = service.AuthClient.ClusterRoles().Create(cro)
	if err != nil {
		return err
	}
	log.Info("Role for Admin Console created", "Name", cr.Name, "ClusterRoleName", cro.Name)
	return nil

}

func (service K8SService) CreateClusterRoleBinding(ac v1alpha1.AdminConsole, binding string) error {
	crbo := &authV1Api.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-%s", ac.Name, ac.Namespace),
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

	if err := controllerutil.SetControllerReference(&ac, crbo, service.Scheme); err != nil {
		return err
	}

	crb, err := service.AuthClient.ClusterRoleBindings().Get(crbo.Name, metav1.GetOptions{})
	if err == nil {
		return nil
	}
	if !k8serrors.IsNotFound(err) {
		return err
	}
	log.V(1).Info("Creating a new ClusterRoleBinding for Admin Console",
		"Namespace", ac.Namespace, "Name", ac.Name)
	crb, err = service.AuthClient.ClusterRoleBindings().Create(crbo)
	if err != nil {
		return err
	}
	log.Info("ClusterRoleBinding has been created",
		"Namespace", crb.Namespace, "Name", crb.Name)
	return nil
}

func (service K8SService) CreateRoleBinding(ac v1alpha1.AdminConsole, name string, binding string, kind string) error {
	rbo := &authV1Api.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ac.Namespace,
		},
		RoleRef: authV1Api.RoleRef{
			Kind: kind,
			Name: binding,
		},
		Subjects: []authV1Api.Subject{
			{
				Kind: "ServiceAccount",
				Name: ac.Name,
			},
		},
	}

	if err := controllerutil.SetControllerReference(&ac, rbo, service.Scheme); err != nil {
		return err
	}

	rb, err := service.AuthClient.RoleBindings(rbo.Namespace).Get(rbo.Name, metav1.GetOptions{})

	if err == nil {
		return nil
	}
	if !k8serrors.IsNotFound(err) {
		return err
	}
	log.V(1).Info("Creating a new RoleBinding for Admin Console",
		"Namespace", ac.Namespace, "Name", ac.Name)
	rb, err = service.AuthClient.RoleBindings(rbo.Namespace).Create(rbo)
	if err != nil {
		return err
	}
	log.Info("RoleBinding has been created", "Namespace", rb.Namespace, "Name", rb.Name)
	return nil

}

func (service K8SService) GetDisplayName(ac v1alpha1.AdminConsole) (string, error) {
	return "", nil
}

func (service K8SService) GenerateDbSettings(ac v1alpha1.AdminConsole) ([]coreV1Api.EnvVar, error) {
	if !ac.Spec.DbSpec.Enabled {
		return []coreV1Api.EnvVar{
			{
				Name:  "DB_ENABLED",
				Value: "false",
			},
		}, nil
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

func (service K8SService) GetExternalUrl(namespace string, name string) (*string, error) {
	ingress, err := service.ExtensionsV1Client.Ingresses(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info("Ingress not found", "Namespace", namespace, "Name", name)
			return nil, nil
		}
		return nil, err
	}

	routeScheme := "https"
	u := fmt.Sprintf("%s://%s%s", routeScheme, ingress.Spec.Rules[0].Host, strings.TrimRight(ingress.Spec.Rules[0].HTTP.Paths[0].Path, "/"))

	return &u, nil
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

	c, err := service.GetConfigmapData(ac.Namespace, "edp-config")
	if err != nil {
		return err
	}

	log.V(1).Info("Creating Admin Console external endpoint.",
		"Namespace", ac.Namespace, "Name", ac.Name)

	l := platformHelper.GenerateLabels(ac.Name)

	so, err := service.CoreClient.Services(ac.Namespace).Get(ac.Name, metav1.GetOptions{})
	if err != nil {
		log.Info("Console Service has not been found", "Namespace", ac.Namespace, "Name", ac.Name)
		return err
	}

	basePath := "/"
	if len(ac.Spec.BasePath) != 0 {
		basePath = fmt.Sprintf("/%v", ac.Spec.BasePath)
	}

	io := &v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ac.Name,
			Namespace: ac.Namespace,
			Labels:    l,
		},
		Spec: v1beta1.IngressSpec{
			Rules: []v1beta1.IngressRule{
				{
					Host: fmt.Sprintf("%s-%s.%s", ac.Name, ac.Namespace, c["dns_wildcard"]),
					IngressRuleValue: v1beta1.IngressRuleValue{
						HTTP: &v1beta1.HTTPIngressRuleValue{
							Paths: []v1beta1.HTTPIngressPath{
								{
									Path: basePath,
									Backend: v1beta1.IngressBackend{
										ServiceName: ac.Name,
										ServicePort: intstr.IntOrString{
											IntVal: so.Spec.Ports[0].TargetPort.IntVal,
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
	if err := controllerutil.SetControllerReference(&ac, io, service.Scheme); err != nil {
		return err
	}

	i, err := service.ExtensionsV1Client.Ingresses(io.Namespace).Get(io.Name, metav1.GetOptions{})

	if !k8serrors.IsNotFound(err) {
		return err
	}

	i, err = service.ExtensionsV1Client.Ingresses(io.Namespace).Create(io)
	if err != nil {
		return err
	}

	log.Info("Ingress has been created",
		"Namespace", i.Namespace, "Name", i.Name)

	return nil
}

func (service K8SService) GetConfigmapData(namespace string, name string) (map[string]string, error) {
	c, err := service.CoreClient.ConfigMaps(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info("Config Map not found", "Namespace", namespace, "Name", name)
			return map[string]string{}, nil
		}
		return map[string]string{}, err
	}
	return c.Data, nil
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

	compCl, err := edpCompClient.NewForConfig(config)
	if err != nil {
		return errors.Wrap(err, "failed to init edp component client")
	}

	service.EdpClient = *edpClient
	service.CoreClient = *coreClient
	service.Scheme = scheme
	service.k8sUnstructuredClient = *k8sClient
	service.AppsClient = *appsClient
	service.ExtensionsV1Client = *extensionsClient
	service.AuthClient = *rbacClient
	service.edpCompClient = *compCl
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

func (service K8SService) CreateEDPComponentIfNotExist(ac v1alpha1.AdminConsole, url string, icon string) error {
	comp, err := service.edpCompClient.
		EDPComponents(ac.Namespace).
		Get(ac.Name, metav1.GetOptions{})
	if err == nil {
		log.Info("edp component already exists", "name", comp.Name)
		return nil
	}
	if k8serrors.IsNotFound(err) {
		return service.createEDPComponent(ac, url, icon)
	}
	return errors.Wrapf(err, "failed to get edp component: %v", ac.Name)
}

func (service K8SService) createEDPComponent(ac v1alpha1.AdminConsole, url string, icon string) error {
	obj := &edpCompApi.EDPComponent{
		ObjectMeta: metav1.ObjectMeta{
			Name: ac.Name,
		},
		Spec: edpCompApi.EDPComponentSpec{
			Type: "admin-console",
			Url:  url,
			Icon: icon,
		},
	}
	if err := controllerutil.SetControllerReference(&ac, obj, service.Scheme); err != nil {
		return err
	}
	_, err := service.edpCompClient.
		EDPComponents(ac.Namespace).
		Create(obj)
	return err
}
