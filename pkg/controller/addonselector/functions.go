package addonselector

import (
        "fmt"
        "reflect"
        "strings"
        "io/ioutil"
        "path/filepath"
        "os"

	addonmanagerv1alpha1 "github.com/jiuchen1986/addon-manager-operator/pkg/apis/addonmanager/v1alpha1"

        metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
        "sigs.k8s.io/yaml"
)

// Set status on an object of an Addon
func setAddonObjectStatus(selector *addonmanagerv1alpha1.AddonSelector, addon, instanceId string, object addonmanagerv1alpha1.AddonObject, protect bool) error {

        if selector.Status.InstanceAwareAddonStatuses == nil {
               selector.Status.InstanceAwareAddonStatuses = make(map[string] []*addonmanagerv1alpha1.AddonStatus)
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
                        return nil, nil, fmt.Errorf("Status of object %v cannot be set!", obj)
                }
                valueStatus.Set(reflect.Zero(valueStatus.Type()))
        }
 
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
