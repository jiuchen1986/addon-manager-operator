package controller

import (
	"github.com/jiuchen1986/addon-manager-operator/pkg/controller/addonselector"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, addonselector.Add)
}
