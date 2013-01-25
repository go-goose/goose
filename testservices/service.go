package testservices

import (
	"launchpad.net/goose/testservices/identityservice"
	"net/http"
	"runtime"
	"strings"
)

// An HttpService provides the HTTP API for a service double.
type HttpService interface {
	SetupHTTP(mux *http.ServeMux)
}

// A ServiceInstance is an Openstack module, one of nova, swift, glance.
type ServiceInstance struct {
	identityservice.ServiceProvider
	ServiceControl
	IdentityService identityservice.IdentityService
	Hostname        string
	VersionPath     string
	TenantId        string
	Region          string
	// Hooks to run when specified control points are reached in the service business logic.
	ControlHooks map[string]ControlProcessor
}

// ControlProcessor defines a function that is run when a specified control point is reached in the service
// business logic. The function receives the service instance so internal state can be inspected, plus for any
// arguments passed to the currently executing service function.
type ControlProcessor func(sc ServiceControl, args ...interface{}) error

// ServiceControl instances allow hooks to be registered for execution at the specified point of execution.
// The control point name can be a function name or a logical execution point meaningful to the service.
// If name is "", the hook for the currently executing function is executed.
type ServiceControl interface {
	RegisterControlPoint(name string, controller ControlProcessor)
}

// currentServiceMethodName returns the method executing on the service when ProcessControlHook was invoked.
func (s *ServiceInstance) currentServiceMethodName() string {
	pc, _, _, ok := runtime.Caller(2)
	if !ok {
		panic("current method name cannot be found")
	}
	return unqualifiedMethodName(pc)
}

func unqualifiedMethodName(pc uintptr) string {
	f := runtime.FuncForPC(pc)
	fullName := f.Name()
	nameParts := strings.Split(fullName, ".")
	return nameParts[len(nameParts)-1]
}

// ProcessControlHook retrieves the ControlProcessor for the specified hook name and runs it, returning any error.
// Use it like this with a "" hookName to invoke a hook registered for the current function:
// if err := n.ProcessControlHook("", <serviceinstance>, <somearg1>, <somearg2>); err != nil {
//     return err
// }
//
// Use it like this to invoke a hook registered for some arbitrary control point:
// if err := n.ProcessControlHook("foobar", <serviceinstance>, <somearg1>, <somearg2>); err != nil {
//     return err
// }
func (s *ServiceInstance) ProcessControlHook(hookName string, sc ServiceControl, args ...interface{}) error {
	if hookName == "" {
		hookName = s.currentServiceMethodName()
	}
	if hook, ok := s.ControlHooks[hookName]; ok {
		return hook(sc, args...)
	}
	return nil
}

// RegisterControlPoint assigns the specified controller to the named hook. If nil, any existing controller for the
// hook is removed.
// hookName is the name of a function on the service or some arbitrarily named control point.
func (s *ServiceInstance) RegisterControlPoint(hookName string, controller ControlProcessor) {
	if controller == nil {
		delete(s.ControlHooks, hookName)
	} else {
		s.ControlHooks[hookName] = controller
	}
}
