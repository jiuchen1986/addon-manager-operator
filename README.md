# addon-manager-operator
an experimental kubernetes operator managing addons in assistance with addon-manager

## Prerequisites
This project is built by operator-sdk at https://github.com/operator-framework/operator-sdk.
You may check the guides of operator-sdk for HOWTO manage a project using operator-sdk.

## Build
  - mkdir $GOPATH/src/github.com/jiuchen1986
  - cd $GOPATH/src/github.com/jiuchen1986
  - git clone https://github.com/jiuchen1986/addon-manager-operator.git

## Local Debug
  - cd $GOPATH/src/github.com/jiuchen1986/addon-manager-operator
  - deploy $GOPATH/src/github.com/jiuchen1986/addon-manager-operator/deploy/crds/addonmanager_v1alpha1_addonselector_crd.yaml to target cluster
  - edit $GOPATH/src/github.com/jiuchen1986/addon-manager-operator/deploy/crds/addonmanager_v1alpha1_addonselector_cr.yaml
  - operator-sdk up local
  - managing addons with addons' manifests and addonmanager_v1alpha1_addonselector_cr.yaml on target cluster
