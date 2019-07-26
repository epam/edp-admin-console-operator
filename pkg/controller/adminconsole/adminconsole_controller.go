package adminconsole

import (
	"admin-console-operator/pkg/service"
	"context"
	"time"

	edpv1alpha1 "admin-console-operator/pkg/apis/edp/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	logPrint "log"
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
	StatusConfiguring      = "configuring"
	StatusConfigured       = "configured"
	StatusExposeStart      = "exposing config"
	StatusExposeFinish     = "config exposed"
	StatusIntegrationStart = "integration started"
	StatusReady            = "ready"
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
	platformService, _ := service.NewPlatformService(scheme)
	adminConsoleService := service.NewAdminConsoleService(platformService, client)

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
	service service.AdminConsoleService
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
		err = r.updateStatus(instance, StatusInstall)
		if err != nil {
			r.resourceActionFailed(instance, err)
			return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
		}
	}

	instance, err = r.service.Install(*instance)
	if err != nil {
		logPrint.Printf("[ERROR] Cannot install Admin Console %s. The reason: %s", instance.Name, err)
		r.resourceActionFailed(instance, err)
		return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
	}

	if instance.Status.Status == StatusInstall {
		logPrint.Printf("Installing Admin Console component has been finished")
		err = r.updateStatus(instance, StatusCreated)
		if err != nil {
			r.resourceActionFailed(instance, err)
			return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
		}
	}

	if instance.Status.Status == StatusCreated || instance.Status.Status == "" {
		logPrint.Println("Admin Console configuration has been started")
		err := r.updateStatus(instance, StatusConfiguring)
		if err != nil {
			return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
		}
	}

	instance, err = r.service.Configure(*instance)
	if err != nil {
		logPrint.Printf("[ERROR] Cannot run Admin Console post-configuration %s %s. The reason: %s", instance.Name, instance.Spec.Version, err)
		return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
	}

	if instance.Status.Status == StatusConfiguring {
		logPrint.Println("Admin Console component configuration has been finished")
		err = r.updateStatus(instance, StatusConfigured)
		if err != nil {
			return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
		}
	}

	if instance.Status.Status == StatusConfigured {
		logPrint.Println("Admin Console component configuration has been finished")
		err = r.updateStatus(instance, StatusExposeStart)
		if err != nil {
			return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
		}
	}

	instance, err = r.service.ExposeConfiguration(*instance)
	if err != nil {
		logPrint.Printf("[ERROR] Cannot expose configuration for Admin Console %s. The reason: %s", instance.Name, err)
		return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
	}

	if instance.Status.Status == StatusExposeStart {
		logPrint.Println("Admin Console component configuration has been finished")
		err = r.updateStatus(instance, StatusExposeFinish)
		if err != nil {
			return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
		}
	}

	if instance.Status.Status == StatusExposeFinish {
		logPrint.Println("Admin Console component configuration has been finished")
		err = r.updateStatus(instance, StatusIntegrationStart)
		if err != nil {
			return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
		}
	}

	instance, err = r.service.Integrate(*instance)
	if err != nil {
		logPrint.Printf("[ERROR] Cannot integrate Admin Console %s. The reason: %s", instance.Name, err)
		return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
	}

	if instance.Status.Status == StatusIntegrationStart {
		logPrint.Println("Admin Console component configuration has been finished")
		err = r.updateStatus(instance, StatusReady)
		if err != nil {
			return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
		}
	}

	err = r.updateAvailableStatus(instance, true)
	if err != nil {
		r.resourceActionFailed(instance, err)
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileAdminConsole) updateStatus(instance *edpv1alpha1.AdminConsole, status string) error {

	instance.Status.Status = status
	instance.Status.LastTimeUpdated = time.Now()
	err := r.client.Status().Update(context.TODO(), instance)
	if err != nil {
		err := r.client.Update(context.TODO(), instance)
		if err != nil {
			return err
		}
	}

	logPrint.Printf("Status for Admin Console %v has been updated to '%v' at %v.", instance.Name, status, instance.Status.LastTimeUpdated)
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
