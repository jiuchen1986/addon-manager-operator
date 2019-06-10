package addonselector

import (
        "testing"
        //"fmt"
		
        addonmanagerv1alpha1 "github.com/cnde/addon-manager-operator/pkg/apis/addonmanager/v1alpha1"
)

func TestSetStatus(t *testing.T) {
        object_1 := addonmanagerv1alpha1.AddonObject{
                Namespace: "test-ns",
                Name: "test-pod-1",
                Group: "",
                Kind: "Pod",
        }
        
        object_2 := addonmanagerv1alpha1.AddonObject{
                Namespace: "test-ns",
                Name: "test-pod-2",
                Group: "",
                Kind: "Pod",
        }
 
        selector := &addonmanagerv1alpha1.AddonSelector{
                Spec: addonmanagerv1alpha1.AddonSelectorSpec{
                        Addons: []addonmanagerv1alpha1.Addon{{
                                Name: "test-addon",
                                AddonObjects: []addonmanagerv1alpha1.AddonObject{object_1, object_2},
                        }},
                },
        }
 
        err := setAddonObjectStatus(selector, "test-addon", object_1, false)

        if err != nil {
                t.Error(err)
        }

        addon_status, ok := selector.Status.AddonStatuses["test-addon"]

        if !ok {
                t.Error("Add object_1 status to selector failed!")
        }

        if addon_status.AddonObjectStatuses[0].Protect != false || addon_status.AddonObjectStatuses[0].Name != "test-pod-1" {
                t.Error("The object_1 status is wrongly added to selector!")
        }



        err = setAddonObjectStatus(selector, "test-addon", object_1, true)

        if err != nil {
                t.Error(err)
        }

        addon_status, _ = selector.Status.AddonStatuses["test-addon"]

        if addon_status.AddonObjectStatuses[0].Protect != true {
                t.Error("Update object_1 status failed!")
        }



        err = setAddonObjectStatus(selector, "test-addon", object_2, true)

        addon_status, _ = selector.Status.AddonStatuses["test-addon"]

        if len(addon_status.AddonObjectStatuses) != 2 {
                t.Error("Add object_2 status to selector failed!")
        }

        if addon_status.AddonObjectStatuses[1].Protect != true || addon_status.AddonObjectStatuses[1].Name != "test-pod-2" {
                t.Error("The object_2 status is wrongly added to selector!")
        }



        err = setAddonObjectStatus(selector, "test-addon", object_2, false)

        addon_status, _ = selector.Status.AddonStatuses["test-addon"]

        if addon_status.AddonObjectStatuses[1].Protect != false || addon_status.AddonObjectStatuses[1].Name != "test-pod-2" {
                t.Error("The object_2 status is wrongly added to selector!")
        }
}
