package addonselector

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	addonmanagerv1alpha1 "github.com/jiuchen1986/addon-manager-operator/pkg/apis/addonmanager/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	us "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured/unstructuredscheme"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	//"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	"github.com/go-logr/logr"
)

// Handle the deletion of the addon selector cr
func handleAddonSelectorDelete(as *addonmanagerv1alpha1.AddonSelector, log logr.Logger) error {

	// Currently only log the deletion
	log.Info("Detect an AddonSelector is deleted. For now only log the deletion.")
	return nil
}

// Return GVK according to AddonObject, and tell wether the GVK should be unstructured
func genGVK(obj addonmanagerv1alpha1.AddonObject, scheme *runtime.Scheme) (schema.GroupVersionKind, bool, error) {
	gvk := schema.GroupVersionKind{
		Group: obj.Group,
		Kind:  obj.Kind,
	}

	if gvs := scheme.PrioritizedVersionsForGroup(gvk.Group); len(gvs) > 0 {
		found := false
		for _, gv := range gvs {
			if obj.Version == gv.Version {
				gvk.Version = obj.Version
				found = true
				break
			}
		}
		if !found {
			gvk.Version = gvs[0].Version
		}
		return gvk, false, nil

	} else {
		gvk.Version = obj.Version
		return gvk, true, nil
	}
}

// Return an instance of k8s runtime object according to AddonObject
func genRuntimeObject(obj addonmanagerv1alpha1.AddonObject, scheme *runtime.Scheme) (runtime.Object, error) {

	// Generate runtime object from the declaired addon object
	gvk, isUnstructured, err := genGVK(obj, scheme)
	if err != nil {
		return nil, err
	}

	if isUnstructured {
		return us.NewUnstructuredCreator().New(gvk)
	}

	return scheme.New(gvk)
}

// Return the first instance of k8s object matching the name prefix of AddonObject
func getInstanceByNamePrefix(addonObj addonmanagerv1alpha1.AddonObject, r *ReconcileAddonSelector) (runtime.Object, error) {
	listObj, checkObj, err := genListObject(addonObj, r.scheme)
	if err != nil {
		return nil, err
	}
	opts := &client.ListOptions{Namespace: addonObj.Namespace}
	err = r.client.List(context.TODO(), opts, listObj)
	if err != nil {
		return nil, err
	}

	return getInstanceFromListObjByNamePrefix(listObj, checkObj, addonObj.Name)

}

// Generate list object, and corresponding element object for checking
func genListObject(addonObj addonmanagerv1alpha1.AddonObject, scheme *runtime.Scheme) (runtime.Object, runtime.Object, error) {
	// a tricky approach is used here by using "<addonObj_Kind>List" as the target list object's Kind
	// where addonObj_Kind is the non-list object's Kind
	// Version and Group are the same
	gvk, isUnstructured, err := genGVK(addonObj, scheme)
	if err != nil {
		return nil, nil, err
	}
	obj, err := genRuntimeObject(addonObj, scheme)
	if err != nil {
		return nil, nil, err
	}
	listGVK := schema.GroupVersionKind{
		Group:   gvk.Group,
		Version: gvk.Version,
		Kind:    gvk.Kind + "List",
	}
	if isUnstructured {
                if listObj, err := us.NewUnstructuredCreator().New(listGVK); err != nil {
                        return nil, nil, err
                } else {
		        return listObj, obj, nil
                }
	}
	if listObj, err := scheme.New(listGVK); err != nil {
                return nil, nil, err
        } else {
                return listObj, obj, nil
        }
}

// Get instance from a list object matching the given name prefix
func getInstanceFromListObjByNamePrefix(listObj, checkObj runtime.Object, prefix string) (runtime.Object, error) {
	// check whether the input object is a list object
	if _, ok := listObj.(metav1.ListInterface); !ok {
		return nil, fmt.Errorf("Input object %s doesn't implement metav1.ListInterface!", listObj)
	}

	objV := reflect.ValueOf(listObj).Elem()
	objT := objV.Type()
	if _, ok := objT.FieldByName("Items"); !ok {
		return nil, fmt.Errorf("No Items field in object %s!", listObj)
	}
	itemsV := objV.FieldByName("Items")

	// be sure the retrieving Items is a list of struct pointed by checkObj
	checkObjT := reflect.ValueOf(checkObj).Elem().Type()
	if itemsV.Type() != reflect.SliceOf(checkObjT) {
		return nil, fmt.Errorf("The %s is not the list object of object %s!", listObj, checkObj)
	}

	var found interface{} = nil
	for i := 0; i < itemsV.Len(); i++ {
		iV := itemsV.Index(i)
		if !iV.CanAddr() {
			return nil, fmt.Errorf("Type of element in Items of list object %s is not addressable!", listObj)
		}
		iIf := iV.Addr().Interface()
		objMeta, ok := iIf.(metav1.Object)
		if !ok {
			return nil, fmt.Errorf("Type of element in Items of list object %s doesn't implement metav1.Object!", listObj)
		}

		// match name prefix
		if strings.Index(objMeta.GetName(), prefix) == 0 {
			found = iIf
			break
		}
	}
	if found == nil {
		return nil, fmt.Errorf("Object %s with name prefix of %s is not found!", checkObj, prefix)
	}

	return found.(runtime.Object), nil
}

// Set status on an object of an Addon
func setAddonObjectStatus(selector *addonmanagerv1alpha1.AddonSelector, addon, instanceId string, object addonmanagerv1alpha1.AddonObject, protect bool) error {

	if selector.Status.InstanceAwareAddonStatuses == nil {
		selector.Status.InstanceAwareAddonStatuses = make(map[string][]*addonmanagerv1alpha1.AddonStatus)
	}

	objectStatus := &addonmanagerv1alpha1.AddonObjectStatus{
		AddonObject: addonmanagerv1alpha1.AddonObject{
			Namespace: object.Namespace,
			Name:      object.Name,
			Group:     object.Group,
			Kind:      object.Kind,
		},
		Protect: protect,
	}

	instanceStatus, ok := selector.Status.InstanceAwareAddonStatuses[instanceId]
	if !ok {
		firstAddonStatus := &addonmanagerv1alpha1.AddonStatus{
			AddonName:           addon,
			AddonObjectStatuses: []*addonmanagerv1alpha1.AddonObjectStatus{objectStatus},
		}
		selector.Status.InstanceAwareAddonStatuses[instanceId] = []*addonmanagerv1alpha1.AddonStatus{firstAddonStatus}
		return nil
	}
	for _, addonStatus := range instanceStatus {
		if addonStatus.AddonName == addon {
			addonStatus.AddonObjectStatuses = updateAddonObjectStatus(addonStatus.AddonObjectStatuses, objectStatus)
			return nil
		}
	}
	newAddonStatus := &addonmanagerv1alpha1.AddonStatus{
		AddonName:           addon,
		AddonObjectStatuses: []*addonmanagerv1alpha1.AddonObjectStatus{objectStatus},
	}
	selector.Status.InstanceAwareAddonStatuses[instanceId] = append(instanceStatus, newAddonStatus)

	return nil

}

// Append or update an object's status in a list of pointers to AddonObjectStatus
func updateAddonObjectStatus(sts []*addonmanagerv1alpha1.AddonObjectStatus, st *addonmanagerv1alpha1.AddonObjectStatus) []*addonmanagerv1alpha1.AddonObjectStatus {
	found := false
	for _, objectStatus := range sts {
		if reflect.DeepEqual(objectStatus.AddonObject, st.AddonObject) {
			objectStatus.Protect = st.Protect
			found = true
		}
	}
	if !found {
		sts = append(sts, st)
	}
	return sts
}

// Add object to protection by writing manifests to disk
func addObjectToProtect(obj runtime.Object, addon, addonsDir string, addonObj addonmanagerv1alpha1.AddonObject) (runtime.Object, error) {
	genObj, serialized, err := genObjectToProtect(obj, addonObj)
	if err != nil {
		return nil, err
	}

	_, er := writeObjectToDisk(serialized, addon, addonsDir, addonObj)
	if er != nil {
		return genObj, err
	}

	return genObj, nil
}

// Generate the object used for protection
func genObjectToProtect(obj runtime.Object, addonObj addonmanagerv1alpha1.AddonObject) (runtime.Object, []byte, error) {

	// Transform the obj to metav1.Object
	objMeta, ok := obj.(metav1.Object)
	if !ok {
		return nil, nil, fmt.Errorf("Object %v doesn't implement metav1.Object!", obj)
	}

	// Set labels to object recognized by addon-manager
	labels := objMeta.GetLabels()
	if labels == nil {
		objMeta.SetLabels(map[string]string{"addonmanager.kubernetes.io/mode": "Reconcile"})
	} else {
		labels["addonmanager.kubernetes.io/mode"] = "Reconcile"
	}

	// remove unnecessary annotations
	anno := objMeta.GetAnnotations()
	if anno != nil {
		// Remove "kubernetes.io/revision" and "last-applied"
		delKeys := []string{}
		for key := range anno {
			if strings.Index(key, "kubernetes.io/revision") != -1 {
				delKeys = append(delKeys, key)
			}

			if strings.Index(key, "last-applied-configuration") != -1 {
				delKeys = append(delKeys, key)
			}
		}
		for _, k := range delKeys {
			delete(anno, k)
		}
		objMeta.SetAnnotations(anno)
	}

	// Remove runtime information
	objMeta.SetGenerateName("")
	objMeta.SetUID("")
	objMeta.SetResourceVersion("")
	objMeta.SetGeneration(0)
	objMeta.SetSelfLink("")
	objMeta.SetCreationTimestamp(metav1.Time{})
	// objMeta.SetInitializers(&metav1.Initializers{})
	objMeta.SetFinalizers([]string{})
	objMeta.SetOwnerReferences([]metav1.OwnerReference{})

	// Seems for 1.13+, there is no need to clean status for running "kubectl apply -f"
	// Remove status information
	//if unsObj, ok := obj.(*unstructured.Unstructured); ok {
	//        if valueStatus, ok := unsObj.Object["status"]; ok {
	//                unsObj.Object["Status"] = reflect.Zero(reflect.TypeOf(valueStatus))
	//        }
	//} else {
	//        _, ok := reflect.ValueOf(obj).Elem().Type().FieldByName("Status")
	//        if !ok {
	//                return, nil, nil, fmt.Errorf("Object %v doesn't have Status field!")
	//        }
	//        valueStatus := reflect.ValueOf(obj).Elem().FieldByName("Status")
	//        if !reflect.DeepEqual(valueStatus, reflect.ValueOf(nil)) {
	//                if !valueStatus.CanSet() {
	//                        return nil, nil, fmt.Errorf("Status of object %v cannot be set!", obj)
	//                }
	//                valueStatus.Set(reflect.Zero(valueStatus.Type()))
	//        }
	//}

	serialized, err := yaml.Marshal(obj)
	if err != nil {
		return obj, nil, err
	}

	return obj, serialized, nil
}

// Check whether the object has been already protected by check files in disk
func isObjectProtected(obj runtime.Object, addon, addonsDir string, addonObj addonmanagerv1alpha1.AddonObject) (bool, error) {

	filename, er := filepath.Abs(filepath.Join(addonsDir, fmt.Sprintf("%s_%s_%s_%s.yaml", addon, addonObj.Kind, addonObj.Namespace, addonObj.Name)))
	if er != nil {
		return false, er
	}

	if _, err := os.Stat(filename); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	return true, nil

}

// Write the manifest file of an object to the disk
func writeObjectToDisk(serialized []byte, addon, addonsDir string, addonObj addonmanagerv1alpha1.AddonObject) (string, error) {

	dirAbs, err := filepath.Abs(addonsDir)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(dirAbs); err != nil {
		return "", err
	}

	// writes the object to disk
	// Name of object file is formatted as <addon_name>_<object_kind>_<namespace>_<name>.yaml
	var filename string
	filename, err = filepath.Abs(filepath.Join(addonsDir, genManifestFileName(addon, addonObj)))
	if err != nil {
		return "", err
	}

	if err := ioutil.WriteFile(filename, serialized, 0666); err != nil {
		return "", err
	}

	return string(serialized), nil
}

func genManifestFileName(addon string, addonObj addonmanagerv1alpha1.AddonObject) string {
	// Name of object file is formatted as <addon_name>_<object_kind>_<namespace>_<name>.yaml
	return fmt.Sprintf("%s_%s_%s_%s.yaml", addon, addonObj.Kind, addonObj.Namespace, addonObj.Name)
}
