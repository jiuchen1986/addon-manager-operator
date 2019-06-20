package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// AddonSelectorSpec defines the desired state of AddonSelector
// +k8s:openapi-gen=true
type AddonSelectorSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
        Addons []Addon `json:"addons"`
}

type Addon struct {
       // name of the addon representing a group of k8s API objects
       Name         string        `json:"name"`
       // API objects constructing the addon
       AddonObjects []AddonObject `json:"addonObjects"`
}

type AddonObject struct {
        // namespace of the object, cluster-scope if ""
        Namespace string `json:"namespace"`
        // name of the object
        Name      string `json:"name"`
        // api group of the object
        Group     string `json:"group"`
        // kind of the object
        Kind      string `json:"kind"`
}

// AddonSelectorStatus defines the observed state of AddonSelector
// +k8s:openapi-gen=true
type AddonSelectorStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
        // Use pointers for easy updating
        // When multiple operator instance exist in non-leadering model, each instance will update status for each object
        // To avoid conficts and implementing lock mechanism, each instance will use Patch to update this part
        // Key of this map is a unique identity of each instance within a cluster, e.g. Pod name 
        InstanceAwareAddonStatuses map[string] []*AddonStatus `json:"instanceAwareAddonStatuses,omitempty"`
}

type AddonStatus struct {
        AddonName string `json:"addonName"`
        // Use pointers for easy updating
        AddonObjectStatuses []*AddonObjectStatus `json:"addonObjectStatuses"`
}

type AddonObjectStatus struct {
        AddonObject  `json:",inline"`
        // whether the addon has been protected
        Protect bool `json:"protect"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AddonSelector is the Schema for the addonselectors API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type AddonSelector struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AddonSelectorSpec   `json:"spec,omitempty"`
	Status AddonSelectorStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AddonSelectorList contains a list of AddonSelector
type AddonSelectorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AddonSelector `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AddonSelector{}, &AddonSelectorList{})
}
