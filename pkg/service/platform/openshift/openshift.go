package openshift

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/epmd-edp/admin-console-operator/v2/pkg/apis/edp/v1alpha1"
	adminConsoleSpec "github.com/epmd-edp/admin-console-operator/v2/pkg/service/admin_console/spec"
	platformHelper "github.com/epmd-edp/admin-console-operator/v2/pkg/service/platform/helper"
	"github.com/epmd-edp/admin-console-operator/v2/pkg/service/platform/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"

	appsV1Api "github.com/openshift/api/apps/v1"
	authV1Api "github.com/openshift/api/authorization/v1"
	routeV1Api "github.com/openshift/api/route/v1"
	securityV1Api "github.com/openshift/api/security/v1"
	appsV1client "github.com/openshift/client-go/apps/clientset/versioned/typed/apps/v1"
	authV1Client "github.com/openshift/client-go/authorization/clientset/versioned/typed/authorization/v1"
	projectV1Client "github.com/openshift/client-go/project/clientset/versioned/typed/project/v1"
	routeV1Client "github.com/openshift/client-go/route/clientset/versioned/typed/route/v1"
	securityV1Client "github.com/openshift/client-go/security/clientset/versioned/typed/security/v1"
	templateV1Client "github.com/openshift/client-go/template/clientset/versioned/typed/template/v1"
	"github.com/pkg/errors"

	coreV1Api "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var log = logf.Log.WithName("platform")

type OpenshiftService struct {
	kubernetes.K8SService

	authClient     authV1Client.AuthorizationV1Client
	templateClient templateV1Client.TemplateV1Client
	projectClient  projectV1Client.ProjectV1Client
	securityClient securityV1Client.SecurityV1Client
	appClient      appsV1client.AppsV1Client
	routeClient    routeV1Client.RouteV1Client
}

func (service OpenshiftService) CreateClusterRole(instance v1alpha1.AdminConsole) error {
	return nil
}

func (service OpenshiftService) CreateClusterRoleBinding(ac v1alpha1.AdminConsole, binding string) error {
	return nil
}

func (service OpenshiftService) CreateDeployConf(ac v1alpha1.AdminConsole, url string) error {
	openshiftClusterURL, err := service.getClusterURL()
	if err != nil {
		return errors.Wrap(err, "Unable to build an OpenshiftClusterURL value")
	}

	keycloakEnabled := "false"

	basePath := ""
	if len(ac.Spec.BasePath) != 0 {
		basePath = fmt.Sprintf("/%v", ac.Spec.BasePath)
	}

	labels := platformHelper.GenerateLabels(ac.Name)
	consoleDcObject := &appsV1Api.DeploymentConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ac.Name,
			Namespace: ac.Namespace,
			Labels:    labels,
		},
		Spec: appsV1Api.DeploymentConfigSpec{
			Replicas: 1,
			Triggers: []appsV1Api.DeploymentTriggerPolicy{
				{
					Type: appsV1Api.DeploymentTriggerOnConfigChange,
				},
			},
			Strategy: appsV1Api.DeploymentStrategy{
				Type: appsV1Api.DeploymentStrategyTypeRolling,
			},
			Selector: labels,
			Template: &coreV1Api.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: coreV1Api.PodSpec{
					ImagePullSecrets: ac.Spec.ImagePullSecrets,
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
									Value: keycloakEnabled,
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
									Name:  "OPENSHIFT_CLUSTER_URL",
									Value: openshiftClusterURL,
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
									Value: "openshift",
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
				},
			},
		},
	}
	if err := controllerutil.SetControllerReference(&ac, consoleDcObject, service.Scheme); err != nil {
		return err
	}

	consoleDc, err := service.appClient.DeploymentConfigs(consoleDcObject.Namespace).Get(consoleDcObject.Name, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			msg := fmt.Sprintf("Creating DeploymentConfig %s/%s for Admin Console %s", consoleDcObject.Namespace, consoleDcObject.Name, ac.Name)
			log.V(1).Info(msg)
			dbEnvVars, err := service.GenerateDbSettings(ac)
			if err != nil {
				return errors.Wrap(err, "Failed to generate environment variables for shared database!")
			}
			consoleDcObject.Spec.Template.Spec.Containers[0].Env = append(consoleDcObject.Spec.Template.Spec.Containers[0].Env, dbEnvVars...)
			consoleDc, err = service.appClient.DeploymentConfigs(consoleDcObject.Namespace).Create(consoleDcObject)
			if err != nil {
				return err
			}
			log.Info(fmt.Sprintf("DeploymentConfig %s/%s has been created", consoleDc.Namespace, consoleDc.Name))
			return nil
		}
		return err
	}

	return nil
}

func (service OpenshiftService) CreateExternalEndpoint(ac v1alpha1.AdminConsole) error {

	labels := platformHelper.GenerateLabels(ac.Name)

	basePath := "/"
	if len(ac.Spec.BasePath) != 0 {
		basePath = fmt.Sprintf("/%v", ac.Spec.BasePath)
	}

	consoleRouteObject := &routeV1Api.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ac.Name,
			Namespace: ac.Namespace,
			Labels:    labels,
		},
		Spec: routeV1Api.RouteSpec{
			TLS: &routeV1Api.TLSConfig{
				Termination:                   routeV1Api.TLSTerminationEdge,
				InsecureEdgeTerminationPolicy: routeV1Api.InsecureEdgeTerminationPolicyRedirect,
			},
			To: routeV1Api.RouteTargetReference{
				Name: ac.Name,
				Kind: "Service",
			},
			Path: basePath,
		},
	}

	if err := controllerutil.SetControllerReference(&ac, consoleRouteObject, service.Scheme); err != nil {
		return err
	}

	consoleRoute, err := service.routeClient.Routes(consoleRouteObject.Namespace).Get(consoleRouteObject.Name, metav1.GetOptions{})

	if err != nil {
		if k8serrors.IsNotFound(err) {
			msg := fmt.Sprintf("Creating a new Route %s/%s for Admin Console %s", consoleRouteObject.Namespace, consoleRouteObject.Name, ac.Name)
			log.V(1).Info(msg)
			consoleRoute, err = service.routeClient.Routes(consoleRouteObject.Namespace).Create(consoleRouteObject)
			if err != nil {
				return err
			}
			log.Info(fmt.Sprintf("Route %s/%s has been created", consoleRoute.Namespace, consoleRoute.Name))
			return nil
		}
		return err
	}

	return nil
}

func (service OpenshiftService) CreateSecurityContext(ac v1alpha1.AdminConsole) error {

	labels := platformHelper.GenerateLabels(ac.Name)
	priority := int32(1)
	id := int64(1001)

	displayName, err := service.GetDisplayName(ac)
	if err != nil {
		return err
	}

	consoleSCCObject := &securityV1Api.SecurityContextConstraints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", ac.Name, displayName),
			Namespace: ac.Namespace,
			Labels:    labels,
		},
		Volumes: []securityV1Api.FSType{
			securityV1Api.FSTypeSecret,
			securityV1Api.FSTypeDownwardAPI,
			securityV1Api.FSTypeEmptyDir,
			securityV1Api.FSTypePersistentVolumeClaim,
			securityV1Api.FSProjected,
			securityV1Api.FSTypeConfigMap,
		},
		AllowHostDirVolumePlugin: false,
		AllowHostIPC:             false,
		AllowHostNetwork:         false,
		AllowHostPID:             false,
		AllowHostPorts:           false,
		AllowPrivilegedContainer: false,
		AllowedCapabilities:      []coreV1Api.Capability{},
		AllowedFlexVolumes:       []securityV1Api.AllowedFlexVolume{},
		DefaultAddCapabilities:   []coreV1Api.Capability{},
		FSGroup: securityV1Api.FSGroupStrategyOptions{
			Type: securityV1Api.FSGroupStrategyMustRunAs,
		},
		Groups:                 []string{},
		Priority:               &priority,
		ReadOnlyRootFilesystem: false,
		RunAsUser: securityV1Api.RunAsUserStrategyOptions{
			Type: securityV1Api.RunAsUserStrategyMustRunAs,
			UID:  &id,
		},
		SELinuxContext: securityV1Api.SELinuxContextStrategyOptions{
			Type:           securityV1Api.SELinuxStrategyMustRunAs,
			SELinuxOptions: nil,
		},
		SupplementalGroups: securityV1Api.SupplementalGroupsStrategyOptions{
			Type:   securityV1Api.SupplementalGroupsStrategyRunAsAny,
			Ranges: nil,
		},
		Users: []string{
			fmt.Sprintf("system:serviceaccount:%s:%s", ac.Namespace, ac.Name),
		},
	}

	if err := controllerutil.SetControllerReference(&ac, consoleSCCObject, service.Scheme); err != nil {
		return err
	}

	consoleSCC, err := service.securityClient.SecurityContextConstraints().Get(consoleSCCObject.Name, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			msg := fmt.Sprintf("Creating a new Security Context Constraint %s for Admin Console %s", consoleSCCObject.Name, ac.Name)
			log.V(1).Info(msg)
			consoleSCC, err = service.securityClient.SecurityContextConstraints().Create(consoleSCCObject)
			if err != nil {
				return err
			}
			log.Info(fmt.Sprintf("Security Context Constraint %s has been created", consoleSCC.Name))
			return nil
		}
	} else {
		if !reflect.DeepEqual(consoleSCC.Users, consoleSCCObject.Users) {
			consoleSCC, err = service.securityClient.SecurityContextConstraints().Update(consoleSCCObject)
			if err != nil {
				return err
			}

			log.Info(fmt.Sprintf("Security Context Constraint %s has been updated", consoleSCC.Name))
		}
	}

	return nil
}

func (service OpenshiftService) CreateRole(ac v1alpha1.AdminConsole) error {
	consoleRoleObject := &authV1Api.Role{
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

	if err := controllerutil.SetControllerReference(&ac, consoleRoleObject, service.Scheme); err != nil {
		return err
	}

	consoleRole, err := service.authClient.Roles(consoleRoleObject.Namespace).Get(consoleRoleObject.Name, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			msg := fmt.Sprintf("Creating Role %s for Admin Console %s", consoleRoleObject.Name, consoleRoleObject.Name)
			log.V(1).Info(msg)

			consoleRole, err = service.authClient.Roles(consoleRoleObject.Namespace).Create(consoleRoleObject)
			if err != nil {
				return err
			}
			log.Info(fmt.Sprintf("Role %s for Admin Console %s created", consoleRole.Name, ac.Name))
			return nil
		}
		return err
	}
	return nil
}

func (service OpenshiftService) CreateRoleBinding(ac v1alpha1.AdminConsole, name string, binding string, kind string) error {

	rbo := getRoleBindingObjectByKind(ac, name, binding, kind)

	if err := controllerutil.SetControllerReference(&ac, rbo, service.Scheme); err != nil {
		return err
	}

	rb, err := service.authClient.RoleBindings(rbo.Namespace).Get(rbo.Name, metav1.GetOptions{})

	if err != nil {
		if k8serrors.IsNotFound(err) {
			log.V(1).Info("Creating a new RoleBinding for Admin Console",
				"Namespace", ac.Namespace, "Name", ac.Name)
			rb, err = service.authClient.RoleBindings(rbo.Namespace).Create(rbo)
			if err != nil {
				return err
			}
			log.Info("RoleBinding has been created", "Namespace", rb.Namespace, "Name", rb.Name)
			return nil
		}
		return err
	}
	return err
}

func (service OpenshiftService) GetDisplayName(ac v1alpha1.AdminConsole) (string, error) {
	project, err := service.projectClient.Projects().Get(ac.Namespace, metav1.GetOptions{})
	if err != nil && k8serrors.IsNotFound(err) {
		return "", errors.New(fmt.Sprintf("Unable to retrieve project %s", ac.Namespace))
	}

	displayName := project.GetObjectMeta().GetAnnotations()["openshift.io/display-name"]
	if displayName == "" {
		return "", errors.New(fmt.Sprintf("Project display name does not set"))
	}
	return displayName, nil
}

func (service OpenshiftService) GenerateDbSettings(ac v1alpha1.AdminConsole) ([]coreV1Api.EnvVar, error) {
	if !ac.Spec.DbSpec.Enabled {
		msg := fmt.Sprintf("DB_ENABLED flag in %s spec is false.", ac.Name)
		log.V(1).Info(msg)

		return []coreV1Api.EnvVar{
			{
				Name:  "DB_ENABLED",
				Value: "false",
			},
		}, nil
	}

	log.V(1).Info(fmt.Sprintf("Generating DB settings for Admin Console %s", ac.Name))
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

func (service OpenshiftService) GenerateKeycloakSettings(ac v1alpha1.AdminConsole, keycloakUrl string) ([]coreV1Api.EnvVar, error) {
	var out []coreV1Api.EnvVar

	log.V(1).Info(fmt.Sprintf("Generating Keycloak settings for Admin Console %s", ac.Name))

	out = []coreV1Api.EnvVar{
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
	}

	return out, nil
}

func (service OpenshiftService) PatchDeploymentEnv(ac v1alpha1.AdminConsole, env []coreV1Api.EnvVar) error {

	if len(env) == 0 {
		return nil
	}

	dc, err := service.appClient.DeploymentConfigs(ac.Namespace).Get(ac.Name, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info(fmt.Sprintf("Deployment %s not found!", ac.Name))
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

	_, err = service.appClient.DeploymentConfigs(dc.Namespace).Patch(dc.Name, types.StrategicMergePatchType, jsonDc)
	if err != nil {
		return err
	}

	return nil
}

func (service *OpenshiftService) Init(config *rest.Config, scheme *runtime.Scheme, k8sClient *client.Client) error {

	err := service.K8SService.Init(config, scheme, k8sClient)
	if err != nil {
		return err
	}

	templateClient, err := templateV1Client.NewForConfig(config)
	if err != nil {
		return err
	}

	service.templateClient = *templateClient
	projectClient, err := projectV1Client.NewForConfig(config)
	if err != nil {
		return err
	}

	service.projectClient = *projectClient
	securityClient, err := securityV1Client.NewForConfig(config)
	if err != nil {
		return err
	}

	service.securityClient = *securityClient
	appClient, err := appsV1client.NewForConfig(config)
	if err != nil {
		return err
	}

	service.appClient = *appClient
	routeClient, err := routeV1Client.NewForConfig(config)
	if err != nil {
		return err
	}
	service.routeClient = *routeClient

	authClient, err := authV1Client.NewForConfig(config)
	if err != nil {
		return err
	}
	service.authClient = *authClient

	return nil
}

// getClusterURL extracts Openshift's Cluster URL from openshift-web-console/webconsole-config ConfigMap object
func (service OpenshiftService) getClusterURL() (string, error) {
	namespace := "openshift-web-console"
	name := "webconsole-config"
	filename := "webconsole-config.yaml"
	cm, err := service.CoreClient.ConfigMaps(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return "", errors.Wrapf(err, "Unable to get a ConfigMap - %s/%s", namespace, name)
	}
	data, ok := cm.Data[filename]
	if !ok {
		return "", errors.Wrapf(err, "ConfigMap %s/%s has no required data, % is missing", namespace, name, filename)
	}
	config, err := platformHelper.ParseWebConsoleConfig(data)
	if err != nil {
		return "", errors.Wrap(err, "Unable to parse WebConsoleConfiguration")
	}
	clusterURL, err := platformHelper.StripClusterURL(config.ClusterInfo.ConsolePublicURL)
	if err != nil {
		return "", errors.Wrap(err, "Unable to strip a relative path from a URL")
	}
	// Success
	return clusterURL, nil
}

// GetExternalUrl returns Route object from Openshift
func (service OpenshiftService) GetExternalUrl(namespace string, name string) (*string, error) {
	route, err := service.routeClient.Routes(namespace).Get(name, metav1.GetOptions{})
	if err != nil && k8serrors.IsNotFound(err) {
		log.Info(fmt.Sprintf("Route %v in namespace %v not found", name, namespace))
		return nil, err
	} else if err != nil {
		return nil, err
	}

	var routeScheme = "http"
	if route.Spec.TLS.Termination != "" {
		routeScheme = "https"
	}

	u := fmt.Sprintf("%s://%s%s", routeScheme, route.Spec.Host, strings.TrimRight(route.Spec.Path, "/"))
	return &u, nil
}

// IsDeploymentReady gets Deployment Config from Openshift, based on data from Admin Console
func (service OpenshiftService) IsDeploymentReady(instance v1alpha1.AdminConsole) (bool, error) {

	deploymentConfig, err := service.appClient.DeploymentConfigs(instance.Namespace).Get(instance.Name, metav1.GetOptions{})
	if err != nil {
		return false, err
	}

	if deploymentConfig.Status.UpdatedReplicas == 1 && deploymentConfig.Status.AvailableReplicas == 1 {
		return true, nil
	}

	return false, nil
}

func getRoleBindingObjectByKind(ac v1alpha1.AdminConsole, name string, binding string, kind string) *authV1Api.RoleBinding {
	switch kind {
	case "ClusterRole":
		return &authV1Api.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: ac.Namespace,
			},
			RoleRef: coreV1Api.ObjectReference{
				APIVersion: "rbac.authorization.k8s.io",
				Kind:       kind,
				Name:       binding,
			},
			Subjects: []coreV1Api.ObjectReference{
				{
					Kind: "ServiceAccount",
					Name: ac.Name,
				},
			},
		}
	case "Role":
		return &authV1Api.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: ac.Namespace,
			},
			RoleRef: coreV1Api.ObjectReference{
				APIVersion: "rbac.authorization.k8s.io",
				Kind:       kind,
				Name:       binding,
				Namespace:  ac.Namespace,
			},
			Subjects: []coreV1Api.ObjectReference{
				{
					Kind: "ServiceAccount",
					Name: ac.Name,
				},
			},
		}
	}

	return &authV1Api.RoleBinding{}
}
