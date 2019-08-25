package addonselector

import (
	"context"
	// "fmt"
	"time"

	addonmanagerv1alpha1 "github.com/jiuchen1986/addon-manager-operator/pkg/apis/addonmanager/v1alpha1"

	"github.com/go-logr/logr"
	"github.com/spf13/pflag"

	// appsv1beta1 "k8s.io/api/apps/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
        metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	var err error
	var instanceId, addonsDir string
	var delay, interval int16

	instanceId, err = pflag.CommandLine.GetString("instance-id")
	if err != nil {
		log.Error(err, err.Error())
		return nil
	}

	addonsDir, err = pflag.CommandLine.GetString("addons-dir")
	if err != nil {
		log.Error(err, err.Error())
		return nil
	}

	delay, err = pflag.CommandLine.GetInt16("requeue-delay")
	if err != nil {
		log.Error(err, err.Error())
		return nil
	}

	interval, err = pflag.CommandLine.GetInt16("check-interval")
	if err != nil {
		log.Error(err, err.Error())
		return nil
	}
	return &ReconcileAddonSelector{
		client:       mgr.GetClient(),
		scheme:       mgr.GetScheme(),
		instanceId:   instanceId,
		addonsDir:    addonsDir,
		requeueDelay: delay,
		interval:     interval,
	}
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
	// id of local operator instance
	instanceId string
	// local directory to addons
	addonsDir string
	// delay between each requeque
	requeueDelay int16
	// periodic checking interval
	interval int16
}

// Reconcile reads that state of the cluster for a AddonSelector object and makes changes based on the state read
// and what is in the AddonSelector.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileAddonSelector) Reconcile(request reconcile.Request) (reconcile.Result, error) {

	reqLogger := log.WithValues("InstanceId", r.instanceId, "Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling AddonSelector")

	// Fetch the AddonSelector instance
	instance := &addonmanagerv1alpha1.AddonSelector{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			reqLogger.Info("Request AddonSelector object is not found. Maybe deleted!")
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	if instance.GetDeletionTimestamp() != nil {
		handleAddonSelectorDelete(instance, reqLogger)
	}

	// As the Reconcile will not be triggered until the CR is updated,
	// the request needs to be requeued if any addon fails to be protected
	requeue := false

	// Process the addons required to be protected
	for _, addon := range instance.Spec.Addons {
		reqLogger.Info("Found a selected addon", "addon.Name", addon.Name)
		for _, o := range addon.AddonObjects {
                        // to support name prefix matching which may refer to multiple instances of same kind of object
                        // a slice of runtime.Object is used
                        ros := []runtime.Object{}
                        // when use name prefix matching, the actual name(s) of the instance(s) need to be stored
                        // a slice of addonmanagerv1alpha1.AddonObject is used to store this
                        // each element in slice is a copy of original AddonObject except the name is the actual name
                        aos := []addonmanagerv1alpha1.AddonObject{}

                        if o.IsNamePrefix {
                                var err error
                                // Get instances of runtime object matching name prefix of AddonObject
                                ros, err = getInstanceByNamePrefix(o, r)
                                if err != nil {
                                        if errors.IsNotFound(err) {
                                                requeue = true
                                                logObjectInfo(reqLogger, "No any instance matching name prefix is found!", o)
                                                continue
                                        } else {
                                                logObjectError(reqLogger, err, o)
                                                continue
                                        }
                                }
                                for _, ro := range ros {
                                        ao := o
                                        ao.Name = ro.(metav1.Object).GetName()
                                        aos = append(aos, ao)
                                }
                        } else {

			         // Generate runtime object with the exact name of the AddonObject
			         ro, err := genRuntimeObject(o, r.scheme)
			         if err != nil {
				        logObjectError(reqLogger, err, o)
				        continue
			         }
                                 ros = append(ros, ro)
                                 aos = append(aos, o)
                        }

                        protectCount := 0
			for i:=0;i<len(ros);i++ {
                                ro := ros[i]
                                ao := aos[i]
                                isProtected, err := isObjectProtected(ro, addon.Name, r.addonsDir, ao)
                                if err != nil {
                                        logObjectError(reqLogger, err, ao)
                                        continue
                                }
                                if isProtected {
                                        logObjectInfo(reqLogger, "Instance has already been protected!", ao)
                                        protectCount ++
                                        continue
                                }
                                // Get instance from cache
                                nn := types.NamespacedName{Namespace: ao.Namespace, Name: ao.Name}
                                err = r.client.Get(context.TODO(), nn, ro)
                                if err != nil {
                                        if errors.IsNotFound(err) {
                                                logObjectInfo(reqLogger, "Instance of is not found!", ao)
                                        } else {
                                                logObjectError(reqLogger, err, ao)
                                        }
                                        continue
                                }
                                // Add the instance to protection
                                _, err = addObjectToProtect(ro, addon.Name, r.addonsDir, ao)
                                if err != nil {
                                        logObjectError(reqLogger, err, ao)
                                        continue
                                }
                                logObjectInfo(reqLogger, "Instance is protected!", ao)
                                protectCount++
                        }
                        if protectCount == len(ros) {
                                logObjectInfo(reqLogger, "Entire addon object is protected!", o)
                                setAddonObjectStatus(instance, addon.Name, r.instanceId, o, true)
                        } else {
                                requeue = true
                        }
		}
	}
	r.client.Status().Update(context.TODO(), instance)

	if requeue {
		return reconcile.Result{RequeueAfter: time.Second * time.Duration(r.requeueDelay)}, nil
	}

	return reconcile.Result{RequeueAfter: time.Second * time.Duration(r.interval)}, nil

}

func logObjectError(logger logr.Logger, err error, obj addonmanagerv1alpha1.AddonObject) {

	logger.Error(err, err.Error(), "obj.Group", obj.Group, "obj.Version", obj.Version, "obj.Kind", obj.Kind, "obj.Namespace", obj.Namespace, "obj.Name", obj.Name, "obj.IsNamePrefix", obj.IsNamePrefix)

}

func logObjectInfo(logger logr.Logger, msg string, obj addonmanagerv1alpha1.AddonObject) {

	logger.Info(msg, "obj.Group", obj.Group, "obj.Version", obj.Version, "obj.Kind", obj.Kind, "obj.Namespace", obj.Namespace, "obj.Name", obj.Name, "obj.IsNamePrefix", obj.IsNamePrefix)

}

func logObjectInfoV4(logger logr.Logger, msg string, obj addonmanagerv1alpha1.AddonObject) {

	logger.V(4).Info(msg, "obj.Group", obj.Group, "obj.Version", obj.Version, "obj.Kind", obj.Kind, "obj.Namespace", obj.Namespace, "obj.Name", obj.Name, obj.Name, "obj.IsNamePrefix", obj.IsNamePrefix)

}
