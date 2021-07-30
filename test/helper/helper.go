package helper

import (
	"github.com/epam/edp-admin-console-operator/v2/pkg/controller/helper"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

func GetBoolPointer(val bool) *bool {
	return &val
}

func InitClient(initScheme func(scheme *runtime.Scheme)) (client.Client, error) {
	scheme := runtime.NewScheme()
	initScheme(scheme)

	ns, err := helper.GetWatchNamespace()
	if err != nil {
		return nil, err
	}

	cfg := ctrl.GetConfigOrDie()
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme,
		MapperProvider: func(c *rest.Config) (meta.RESTMapper, error) {
			return apiutil.NewDynamicRESTMapper(cfg)
		},
		Namespace: ns,
	})
	if err != nil {
		return nil, err
	}

	cl, err := client.New(mgr.GetConfig(), client.Options{
		Scheme: mgr.GetScheme(),
		Mapper: mgr.GetRESTMapper(),
	})
	if err != nil {
		return nil, err
	}

	return cl, nil
}
