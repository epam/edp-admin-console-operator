package main

import (
	"flag"
	"os"

	adminConsoleApiV1 "github.com/epam/edp-admin-console-operator/v2/pkg/apis/edp/v1"
	adminConsoleApiV1aplpha1 "github.com/epam/edp-admin-console-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-admin-console-operator/v2/pkg/controller/adminconsole"
	"github.com/epam/edp-admin-console-operator/v2/pkg/controller/helper"
	buildInfo "github.com/epam/edp-common/pkg/config"
	edpCompApi "github.com/epam/edp-component-operator/pkg/apis/v1/v1alpha1"
	keycloakV1Api "github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	//+kubebuilder:scaffold:imports
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

const adminConsoleOperatorLock = "edp-admin-console-operator-lock"

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(adminConsoleApiV1.AddToScheme(scheme))

	utilruntime.Must(adminConsoleApiV1aplpha1.AddToScheme(scheme))

	utilruntime.Must(keycloakV1Api.AddToScheme(scheme))

	utilruntime.Must(edpCompApi.AddToScheme(scheme))
}

func main() {
	var (
		metricsAddr          string
		enableLeaderElection bool
		probeAddr            string
	)

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", helper.RunningInCluster(),
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")

	mode, err := helper.GetDebugMode()
	if err != nil {
		setupLog.Error(err, "unable to get debug mode value")
		os.Exit(1)
	}

	opts := zap.Options{
		Development: mode,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	v := buildInfo.Get()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	setupLog.Info("Starting the Admin Console Operator",
		"version", v.Version,
		"git-commit", v.GitCommit,
		"git-tag", v.GitTag,
		"build-date", v.BuildDate,
		"go-version", v.Go,
		"go-client", v.KubectlVersion,
		"platform", v.Platform,
	)

	ns, err := helper.GetWatchNamespace()
	if err != nil {
		setupLog.Error(err, "unable to get watch namespace")
		os.Exit(1)
	}

	cfg := ctrl.GetConfigOrDie()
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		HealthProbeBindAddress: probeAddr,
		Port:                   9443,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       adminConsoleOperatorLock,
		MapperProvider: func(c *rest.Config) (meta.RESTMapper, error) {
			return apiutil.NewDynamicRESTMapper(cfg)
		},
		Namespace: ns,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	cl, err := client.New(mgr.GetConfig(), client.Options{
		Scheme: mgr.GetScheme(),
		Mapper: mgr.GetRESTMapper(),
	})
	if err != nil {
		setupLog.Error(err, "unable to create uncached client")
		os.Exit(1)
	}

	acCtrl, err := adminconsole.NewReconcileAdminConsole(cl, mgr.GetScheme(), ctrl.Log.WithName("controllers"))
	if err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "admin-console")
		os.Exit(1)
	}

	if err := acCtrl.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "admin-console")
		os.Exit(1)
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}

	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
