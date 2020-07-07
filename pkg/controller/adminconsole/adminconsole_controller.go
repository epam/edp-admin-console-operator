package adminconsole

import (
	"context"
	"fmt"
	"github.com/epmd-edp/admin-console-operator/v2/pkg/controller/helper"
	"github.com/epmd-edp/admin-console-operator/v2/pkg/service/admin_console"
	"github.com/epmd-edp/admin-console-operator/v2/pkg/service/platform"
	"os"
	"time"

	edpv1alpha1 "github.com/epmd-edp/admin-console-operator/v2/pkg/apis/edp/v1alpha1"

	errorsf "github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
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

var log = logf.Log.WithName("controller_adminconsole")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new AdminConsole Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	scheme := mgr.GetScheme()
	client := mgr.GetClient()
	platformType := helper.GetPlatformTypeEnv()
	platformService, err := platform.NewPlatformService(platformType, scheme, &client)
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	adminConsoleService := admin_console.NewAdminConsoleService(platformService, client)

	return &ReconcileAdminConsole{
		client:  client,
		scheme:  scheme,
		service: adminConsoleService,
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("adminconsole-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource AdminConsole
	err = c.Watch(&source.Kind{Type: &edpv1alpha1.AdminConsole{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner AdminConsole
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &edpv1alpha1.AdminConsole{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileAdminConsole implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileAdminConsole{}

// ReconcileAdminConsole reconciles a AdminConsole object
type ReconcileAdminConsole struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client  client.Client
	scheme  *runtime.Scheme
	service admin_console.AdminConsoleService
}

// Reconcile reads that state of the cluster for a AdminConsole object and makes changes based on the state read
// and what is in the AdminConsole.Spec
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileAdminConsole) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling AdminConsole")

	// Fetch the AdminConsole instance
	instance := &edpv1alpha1.AdminConsole{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	if instance.Status.Status == "" || instance.Status.Status == StatusFailed {
		reqLogger.Info("Installation has been started")
		err = r.updateStatus(instance, StatusInstall)
		if err != nil {
			return reconcile.Result{RequeueAfter: DefaultRequeueTime * time.Second}, err
		}
	}

	instance, err = r.service.Install(*instance)
	if err != nil {
		err = r.updateStatus(instance, StatusFailed)
		if err != nil {
			return reconcile.Result{RequeueAfter: DefaultRequeueTime * time.Second}, err
		}
		return reconcile.Result{RequeueAfter: DefaultRequeueTime * time.Second}, errorsf.Wrapf(err, "Installation has failed")
	}

	if instance.Status.Status == StatusInstall {
		log.Info("Installation has finished")
		err = r.updateStatus(instance, StatusCreated)
		if err != nil {
			return reconcile.Result{RequeueAfter: DefaultRequeueTime * time.Second}, err
		}
	}

	if dcIsReady, err := r.service.IsDeploymentReady(*instance); err != nil {
		return reconcile.Result{RequeueAfter: DefaultRequeueTime * time.Second}, errorsf.Wrapf(err, "Checking if Deployment configs is ready has been failed")
	} else if !dcIsReady {
		reqLogger.Info("Deployment config is not ready for exposing configuration yet")
		return reconcile.Result{RequeueAfter: DefaultRequeueTime * time.Second}, nil
	}

	if instance.Status.Status == StatusCreated {
		reqLogger.Info("Exposing configuration has started")
		err = r.updateStatus(instance, StatusExposeStart)
		if err != nil {
			return reconcile.Result{RequeueAfter: DefaultRequeueTime * time.Second}, err
		}
	}

	instance, err = r.service.ExposeConfiguration(*instance)
	if err != nil {
		err = r.updateStatus(instance, StatusFailed)
		if err != nil {
			return reconcile.Result{RequeueAfter: DefaultRequeueTime * time.Second}, err
		}
		return reconcile.Result{RequeueAfter: DefaultRequeueTime * time.Second}, errorsf.Wrapf(err, "Exposing configuration failed")
	}

	if instance.Status.Status == StatusExposeStart {
		reqLogger.Info("Exposing configuration has finished")
		err = r.updateStatus(instance, StatusExposeFinish)
		if err != nil {
			return reconcile.Result{RequeueAfter: DefaultRequeueTime * time.Second}, err
		}
	}

	if instance.Status.Status == StatusExposeFinish {
		reqLogger.Info("Integration has started")
		err = r.updateStatus(instance, StatusIntegrationStart)
		if err != nil {
			return reconcile.Result{RequeueAfter: DefaultRequeueTime * time.Second}, err
		}
	}

	instance, err = r.service.Integrate(*instance)
	if err != nil {
		err = r.updateStatus(instance, StatusFailed)
		if err != nil {
			return reconcile.Result{RequeueAfter: DefaultRequeueTime * time.Second}, err
		}
		return reconcile.Result{RequeueAfter: DefaultRequeueTime * time.Second}, errorsf.Wrapf(err, "Integration failed")
	}

	if instance.Status.Status == StatusIntegrationStart {
		reqLogger.Info("Exposing configuration has started")
		err = r.updateStatus(instance, StatusReady)
		if err != nil {
			return reconcile.Result{RequeueAfter: DefaultRequeueTime * time.Second}, err
		}
	}

	err = r.updateAvailableStatus(instance, true)
	if err != nil {
		reqLogger.Info("Failed to update availability status")
		return reconcile.Result{RequeueAfter: DefaultRequeueTime * time.Second}, err
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileAdminConsole) updateStatus(instance *edpv1alpha1.AdminConsole, newStatus string) error {
	reqLogger := log.WithValues("Request.Namespace", instance.Namespace, "Request.Name", instance.Name).WithName("status_update")
	currentStatus := instance.Status.Status
	instance.Status.Status = newStatus
	instance.Status.LastTimeUpdated = time.Now()
	err := r.client.Status().Update(context.TODO(), instance)
	if err != nil {
		err = r.client.Update(context.TODO(), instance)
		if err != nil {
			return errorsf.Wrapf(err, "Couldn't update status from '%v' to '%v'", currentStatus, newStatus)
		}
	}

	reqLogger.Info(fmt.Sprintf("Status has been updated to '%v'", newStatus))
	return nil
}

func (r *ReconcileAdminConsole) resourceActionFailed(instance *edpv1alpha1.AdminConsole, err error) error {
	if r.updateStatus(instance, StatusFailed) != nil {
		return err
	}
	return err
}

func (r ReconcileAdminConsole) updateAvailableStatus(instance *edpv1alpha1.AdminConsole, value bool) error {
	if instance.Status.Available != value {
		instance.Status.Available = value
		instance.Status.LastTimeUpdated = time.Now()
		err := r.client.Status().Update(context.TODO(), instance)
		if err != nil {
			err := r.client.Update(context.TODO(), instance)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
