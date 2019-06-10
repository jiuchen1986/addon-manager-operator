package addonselector

import (
	"context"
        // "fmt"
        "time"

	addonmanagerv1alpha1 "github.com/cnde/addon-manager-operator/pkg/apis/addonmanager/v1alpha1"

        "github.com/go-logr/logr"
        "github.com/spf13/pflag"

        // appsv1beta1 "k8s.io/api/apps/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
        "k8s.io/apimachinery/pkg/runtime/schema"
        "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_addonselector")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new AddonSelector Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileAddonSelector{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {

	// Create a new controller
	c, err := controller.New("addonselector-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource AddonSelector
	err = c.Watch(&source.Kind{Type: &addonmanagerv1alpha1.AddonSelector{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileAddonSelector implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileAddonSelector{}

// ReconcileAddonSelector reconciles a AddonSelector object
type ReconcileAddonSelector struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a AddonSelector object and makes changes based on the state read
// and what is in the AddonSelector.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileAddonSelector) Reconcile(request reconcile.Request) (reconcile.Result, error) {

	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling AddonSelector")

	// Fetch the AddonSelector instance
	instance := &addonmanagerv1alpha1.AddonSelector{}
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

        // As the Reconcile will not be triggered until the CR is updated,
        // the request needs to be requeued if any addon fails to be protected
        requeue := false

        // Process the addons required to be protected
        for _, addon := range instance.Spec.Addons {
               reqLogger.Info("Found a selected addon", "addon.Name", addon.Name)
               for _, o := range addon.AddonObjects {
                        // Generate runtime object from the declaired addon object
                        gvk := schema.GroupVersionKind{
                                Group: o.Group,
                                Version: r.scheme.PrioritizedVersionsForGroup(o.Group)[0].Version,
                                Kind: o.Kind,
                        }
                        logObjectInfoV4(reqLogger, "Generate GVK", gvk.Version, o)
                        runtimeObject, err := r.scheme.New(gvk)
                        if err != nil {
                                logObjectError(reqLogger, err, gvk.Version, o)
                                continue 
                        }

                        // Check wether the object has already been protected
                        var isProtected bool
                        isProtected, err = IsObjectProtected(runtimeObject, addon.Name, o)
                        if err != nil {
                                requeue = true
                                logObjectError(reqLogger, err, gvk.Version, o)
                                continue
                        }

                        if isProtected {
                                logObjectInfo(reqLogger, "Object has already been protected!", gvk.Version, o)
                                SetAddonObjectStatus(instance, addon.Name, o, true)
                                continue
                        }

                        // Get object's instance from cache
                        nn := types.NamespacedName{Namespace: o.Namespace, Name: o.Name,}
                        err = r.client.Get(context.TODO(), nn, runtimeObject)
                        if err != nil {
                                if errors.IsNotFound(err) {
                                        requeue = true
                                        logObjectInfo(reqLogger, "Instance of object is not found!", gvk.Version, o)
                                        continue
                                } else {
                                        requeue = true
                                        logObjectError(reqLogger, err, gvk.Version, o)
                                        continue
                                }
                        }

                        // Add the object to protection
                        _, err = AddObjectToProtect(runtimeObject, addon.Name, o)
                        if err != nil {
                                requeue = true
                                logObjectError(reqLogger, err, gvk.Version, o)
                                continue
                        }
                        // For test
                        // fmt.Println(write_obj)

                        logObjectInfo(reqLogger, "Object is protected!", gvk.Version, o)
                        SetAddonObjectStatus(instance, addon.Name, o, true)
               }
        }

        if requeue {
                delay, err := pflag.CommandLine.GetInt16("requeue-delay")
                if err != nil {
                        reqLogger.Error(err, err.Error())
                        return reconcile.Result{RequeueAfter: time.Second*10,}, nil
                }
                return reconcile.Result{RequeueAfter: time.Second*time.Duration(delay),}, nil
        }

        interval, er := pflag.CommandLine.GetInt16("check-interval")
        if er != nil {
                reqLogger.Error(err, err.Error())
                return reconcile.Result{RequeueAfter: time.Second*600,}, nil
        }
	return reconcile.Result{RequeueAfter: time.Second*time.Duration(interval),}, nil

}

        
func logObjectError(logger logr.Logger, err error, ver string, obj addonmanagerv1alpha1.AddonObject) {

        logger.Error(err, err.Error(), "obj.Group", obj.Group, "obj.Version", ver, "obj.Kind", obj.Kind, "obj.Namespace", obj.Namespace, "obj.Name", obj.Name)

}

func logObjectInfo(logger logr.Logger, msg, ver string, obj addonmanagerv1alpha1.AddonObject) {

        logger.Info(msg, "obj.Group", obj.Group, "obj.Version", ver, "obj.Kind", obj.Kind, "obj.Namespace", obj.Namespace, "obj.Name", obj.Name)

}

func logObjectInfoV4(logger logr.Logger, msg, ver string, obj addonmanagerv1alpha1.AddonObject) {

        logger.V(4).Info(msg, "obj.Group", obj.Group, "obj.Version", ver, "obj.Kind", obj.Kind, "obj.Namespace", obj.Namespace, "obj.Name", obj.Name)

}

