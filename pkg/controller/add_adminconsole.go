package controller

import (
	"github.com/epmd-edp/admin-console-operator/v2/pkg/controller/adminconsole"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, adminconsole.Add)
}
