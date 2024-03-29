// +build !ignore_autogenerated

// Code generated by openapi-gen. DO NOT EDIT.

// This file was autogenerated by openapi-gen. Do not edit it manually!

package v1alpha1

import (
	spec "github.com/go-openapi/spec"
	common "k8s.io/kube-openapi/pkg/common"
)

func GetOpenAPIDefinitions(ref common.ReferenceCallback) map[string]common.OpenAPIDefinition {
	return map[string]common.OpenAPIDefinition{
		"github.com/jiuchen1986/addon-manager-operator/pkg/apis/addonmanager/v1alpha1.AddonSelector":       schema_pkg_apis_addonmanager_v1alpha1_AddonSelector(ref),
		"github.com/jiuchen1986/addon-manager-operator/pkg/apis/addonmanager/v1alpha1.AddonSelectorSpec":   schema_pkg_apis_addonmanager_v1alpha1_AddonSelectorSpec(ref),
		"github.com/jiuchen1986/addon-manager-operator/pkg/apis/addonmanager/v1alpha1.AddonSelectorStatus": schema_pkg_apis_addonmanager_v1alpha1_AddonSelectorStatus(ref),
	}
}

func schema_pkg_apis_addonmanager_v1alpha1_AddonSelector(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "AddonSelector is the Schema for the addonselectors API",
				Properties: map[string]spec.Schema{
					"kind": {
						SchemaProps: spec.SchemaProps{
							Description: "Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"apiVersion": {
						SchemaProps: spec.SchemaProps{
							Description: "APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#resources",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"metadata": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("k8s.io/apimachinery/pkg/apis/meta/v1.ObjectMeta"),
						},
					},
					"spec": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("github.com/jiuchen1986/addon-manager-operator/pkg/apis/addonmanager/v1alpha1.AddonSelectorSpec"),
						},
					},
					"status": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("github.com/jiuchen1986/addon-manager-operator/pkg/apis/addonmanager/v1alpha1.AddonSelectorStatus"),
						},
					},
				},
			},
		},
		Dependencies: []string{
			"github.com/jiuchen1986/addon-manager-operator/pkg/apis/addonmanager/v1alpha1.AddonSelectorSpec", "github.com/jiuchen1986/addon-manager-operator/pkg/apis/addonmanager/v1alpha1.AddonSelectorStatus", "k8s.io/apimachinery/pkg/apis/meta/v1.ObjectMeta"},
	}
}

func schema_pkg_apis_addonmanager_v1alpha1_AddonSelectorSpec(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "AddonSelectorSpec defines the desired state of AddonSelector",
				Properties:  map[string]spec.Schema{},
			},
		},
		Dependencies: []string{},
	}
}

func schema_pkg_apis_addonmanager_v1alpha1_AddonSelectorStatus(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "AddonSelectorStatus defines the observed state of AddonSelector",
				Properties:  map[string]spec.Schema{},
			},
		},
		Dependencies: []string{},
	}
}
