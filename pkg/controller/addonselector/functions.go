package addonselector

import (
        "fmt"
        "reflect"
        "strings"
        "io/ioutil"
        "path/filepath"
        "os"

	addonmanagerv1alpha1 "github.com/cnde/addon-manager-operator/pkg/apis/addonmanager/v1alpha1"

        metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
        "sigs.k8s.io/yaml"

        "github.com/spf13/pflag"
)

// Set status on an object of an Addon
func SetAddonObjectStatus(selector *addonmanagerv1alpha1.AddonSelector, addon string, object addonmanagerv1alpha1.AddonObject, protect bool) error {

        if selector.Status.AddonStatuses == nil {
               selector.Status.AddonStatuses = make(map[string] *addonmanagerv1alpha1.AddonStatus)
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

        addonStatus, ok := selector.Status.AddonStatuses[addon]
        if !ok {
               selector.Status.AddonStatuses[addon] = &addonmanagerv1alpha1.AddonStatus{
                       AddonObjectStatuses: make([]*addonmanagerv1alpha1.AddonObjectStatus, 1),
               }
               selector.Status.AddonStatuses[addon].AddonObjectStatuses[0] = objectStatus
        } else {
               for _, status := range addonStatus.AddonObjectStatuses {
                         if status.AddonObject == object {
                                   status.Protect = protect
                                   return nil
                         }
               }
               objectStatuses := selector.Status.AddonStatuses[addon].AddonObjectStatuses
               objectStatuses = append(objectStatuses, objectStatus)
               selector.Status.AddonStatuses[addon].AddonObjectStatuses = objectStatuses
        }

        return nil

}

// Add object to protection by writing manifest to disk
func AddObjectToProtect(obj runtime.Object, addon string, addonObj addonmanagerv1alpha1.AddonObject) (runtime.Object, error) {

        // Transform the obj to metav1.Object
        objMeta, ok := obj.(metav1.Object)
        if !ok {
                return nil, fmt.Errorf("Object %v doesn't implement metav1.Object!", obj)
        }

        // Set annotation to object recognized by addon-manager
        anno := objMeta.GetAnnotations()
        if anno == nil {
                objMeta.SetAnnotations(map[string] string{}) 
        }
        anno = objMeta.GetAnnotations()
        anno["addonmanager.kubernetes.io/mode"] = "Reconcile"

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

        // Remove status information
        valueStatus := reflect.ValueOf(obj).Elem().FieldByName("Status")
        if !reflect.DeepEqual(valueStatus, reflect.ValueOf(nil)) {
                if !valueStatus.CanSet() {
                        return nil, fmt.Errorf("Status of object %v cannot be set!", obj)
                }
                valueStatus.Set(reflect.Zero(valueStatus.Type()))
        }
 
        if err := writeObjectToDisk(obj, addon, addonObj); err != nil {
                return nil, err
        }

        return obj, nil
}

// Check whether the object has been already protected by check files in disk
func IsObjectProtected(obj runtime.Object, addon string, addonObj addonmanagerv1alpha1.AddonObject) (bool, error) {

        //For now, only check whether a file named <addon_name>_<object_kind>_<namespace>_<name>.yaml exists
        addonsDir, err := pflag.CommandLine.GetString("addons-dir")
        if err != nil {
                return false, err
        }

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
func writeObjectToDisk(obj runtime.Object, addon string, addonObj addonmanagerv1alpha1.AddonObject) error {

	// writes the object to disk
	serialized, err := yaml.Marshal(obj)
	if err != nil {
                return err
	}

        // fmt.Println(string(serialized))
        var addonsDir string
        addonsDir, err = pflag.CommandLine.GetString("addons-dir")
        if err != nil {
                return err
        }
        // fmt.Println("Addons-dir:", addonsDir)
        
        // Name of object file is formatted as <addon_name>_<object_kind>_<namespace>_<name>.yaml
        var filename string
	filename, err = filepath.Abs(filepath.Join(addonsDir, fmt.Sprintf("%s_%s_%s_%s.yaml", addon, addonObj.Kind, addonObj.Namespace, addonObj.Name)))
        if err != nil {
                return err
        }

	if err := ioutil.WriteFile(filename, serialized, 0666); err != nil {
                return err
	}

	return nil
}
