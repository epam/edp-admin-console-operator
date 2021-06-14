package adminconsole

import (
	"context"
	"fmt"
	"github.com/epam/edp-admin-console-operator/v2/pkg/controller/helper"
	"github.com/epam/edp-admin-console-operator/v2/pkg/service/admin_console"
	"github.com/epam/edp-admin-console-operator/v2/pkg/service/platform"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"time"

	adminConsoleApi "github.com/epam/edp-admin-console-operator/v2/pkg/apis/edp/v1alpha1"

	"github.com/pkg/errors"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	StatusInstall          = "installing"
	StatusFailed           = "failed"
	StatusCreated          = "created"
	StatusExposeStart      = "exposing config"
	StatusExposeFinish     = "config exposed"
	StatusIntegrationStart = "integration started"
	StatusReady            = "ready"
	DefaultRequeueTime     = 30
)

func NewReconcileAdminConsole(client client.Client, scheme *runtime.Scheme, log logr.Logger) (*ReconcileAdminConsole, error) {
	ps, err := platform.NewPlatformService(helper.GetPlatformTypeEnv(), scheme, &client)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create platform service")
	}

	return &ReconcileAdminConsole{
		client:  client,
		scheme:  scheme,
		service: admin_console.NewAdminConsoleService(ps, client, scheme),
		log:     log.WithName("admin-console"),
	}, nil
}

type ReconcileAdminConsole struct {
	client  client.Client
	scheme  *runtime.Scheme
	service admin_console.AdminConsoleService
	log     logr.Logger
}

func (r *ReconcileAdminConsole) SetupWithManager(mgr ctrl.Manager) error {
	c, err := controller.New("adminconsole-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource AdminConsole
	err = c.Watch(&source.Kind{Type: &adminConsoleApi.AdminConsole{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner AdminConsole
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &adminConsoleApi.AdminConsole{},
	})
	if err != nil {
		return err
	}

	return nil
}

func (r *ReconcileAdminConsole) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	log.Info("Reconciling AdminConsole")

	instance := &adminConsoleApi.AdminConsole{}
	if err := r.client.Get(ctx, request.NamespacedName, instance); err != nil {
		if k8sErrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if instance.Status.Status == "" || instance.Status.Status == StatusFailed {
		log.Info("Installation has been started")
		if err := r.updateStatus(ctx, instance, StatusInstall); err != nil {
			return reconcile.Result{RequeueAfter: DefaultRequeueTime * time.Second}, err
		}
	}

	if instance.Status.Status == StatusInstall {
		log.Info("Installation has finished")
		if err := r.updateStatus(ctx, instance, StatusCreated); err != nil {
			return reconcile.Result{RequeueAfter: DefaultRequeueTime * time.Second}, err
		}
	}

	if dcIsReady, err := r.service.IsDeploymentReady(*instance); err != nil {
		return reconcile.Result{RequeueAfter: DefaultRequeueTime * time.Second}, errors.Wrapf(err, "Checking if Deployment configs is ready has been failed")
	} else if !dcIsReady {
		log.Info("Deployment config is not ready for exposing configuration yet")
		return reconcile.Result{RequeueAfter: DefaultRequeueTime * time.Second}, nil
	}

	if instance.Status.Status == StatusCreated {
		log.Info("Exposing configuration has started")
		if err := r.updateStatus(ctx, instance, StatusExposeStart); err != nil {
			return reconcile.Result{RequeueAfter: DefaultRequeueTime * time.Second}, err
		}
	}

	instance, err := r.service.ExposeConfiguration(*instance)
	if err != nil {
		if err := r.updateStatus(ctx, instance, StatusFailed); err != nil {
			return reconcile.Result{RequeueAfter: DefaultRequeueTime * time.Second}, err
		}
		return reconcile.Result{RequeueAfter: DefaultRequeueTime * time.Second}, errors.Wrapf(err, "Exposing configuration failed")
	}

	if instance.Status.Status == StatusExposeStart {
		log.Info("Exposing configuration has finished")
		err = r.updateStatus(ctx, instance, StatusExposeFinish)
		if err != nil {
			return reconcile.Result{RequeueAfter: DefaultRequeueTime * time.Second}, err
		}
	}

	if instance.Status.Status == StatusExposeFinish {
		log.Info("Integration has started")
		err = r.updateStatus(ctx, instance, StatusIntegrationStart)
		if err != nil {
			return reconcile.Result{RequeueAfter: DefaultRequeueTime * time.Second}, err
		}
	}

	instance, err = r.service.Integrate(*instance)
	if err != nil {
		log.Error(err, "couldn't finish integrating")
		if err = r.updateStatus(ctx, instance, StatusFailed); err != nil {
			return reconcile.Result{RequeueAfter: DefaultRequeueTime * time.Second}, nil
		}
		return reconcile.Result{RequeueAfter: DefaultRequeueTime * time.Second}, nil
	}

	if instance.Status.Status == StatusIntegrationStart {
		log.Info("Exposing configuration has started")
		err = r.updateStatus(ctx, instance, StatusReady)
		if err != nil {
			return reconcile.Result{RequeueAfter: DefaultRequeueTime * time.Second}, err
		}
	}

	err = r.updateAvailableStatus(ctx, instance, true)
	if err != nil {
		log.Info("Failed to update availability status")
		return reconcile.Result{RequeueAfter: DefaultRequeueTime * time.Second}, err
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileAdminConsole) updateStatus(ctx context.Context, instance *adminConsoleApi.AdminConsole, newStatus string) error {
	log := r.log.WithValues("Request.Namespace", instance.Namespace, "Request.Name", instance.Name).WithName("status_update")
	currentStatus := instance.Status.Status
	instance.Status.Status = newStatus
	instance.Status.LastTimeUpdated = time.Now()

	if err := r.client.Status().Update(ctx, instance); err != nil {
		if err := r.client.Update(ctx, instance); err != nil {
			return errors.Wrapf(err, "Couldn't update status from '%v' to '%v'", currentStatus, newStatus)
		}
	}

	log.Info(fmt.Sprintf("Status has been updated to '%v'", newStatus))
	return nil
}

func (r *ReconcileAdminConsole) resourceActionFailed(ctx context.Context, instance *adminConsoleApi.AdminConsole, err error) error {
	if r.updateStatus(ctx, instance, StatusFailed) != nil {
		return err
	}
	return err
}

func (r ReconcileAdminConsole) updateAvailableStatus(ctx context.Context, instance *adminConsoleApi.AdminConsole, value bool) error {
	if instance.Status.Available != value {
		instance.Status.Available = value
		instance.Status.LastTimeUpdated = time.Now()
		if err := r.client.Status().Update(ctx, instance); err != nil {
			if err := r.client.Update(ctx, instance); err != nil {
				return err
			}
		}
	}
	return nil
}
