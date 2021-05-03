package controller

import (
	"fmt"

	v1 "github.com/openshift/api/route/v1"
)

const (
	cnamePhase   = "xxx.xxx.com/cname-phase"
	initializing = "initializing"
	failed       = "failed"
	complete     = "complete"
)

// getAnnotations returns the cname-phase annotation. If it does not exist,
// set the annotation to a default value of 'initializing', which indicates
// that the Route is waiting for ACC to assign an initial alias.
func getRouteAnnotations(route *v1.Route) (map[string]string, string) {
	annotations := route.GetAnnotations()
	if len(annotations) == 0 {
		annotations = map[string]string{}
	}

	if _, exist := annotations[cnamePhase]; !exist {
		annotations[cnamePhase] = initializing
	}
	return annotations, annotations[cnamePhase]
}

// updateRouteAnnotation will update the cname-phase annotation if it changes
func (c *Controller) updateRouteAnnotation(route *v1.Route, a map[string]string, phase string) {
	if a[cnamePhase] == phase {
		return
	}

	if len(phase) > 0 {
		copy := route.DeepCopy()
		a[cnamePhase] = phase
		copy.SetAnnotations(a)
		//c.RouteClient.RouteV1().Routes(copy.Namespace).Update(copy)
	}
}

// A route is considered to be in the 'initializing' phase if it has the initializing
// phase annotation, and address management does not return any current aliases for the
// cname. If no annotation is found at the beginning of UpdateRoute, add a temporary
// annotation of 'initializing'.
func isRouteInitializing(phase, actual string) bool {
	return phase == initializing && len(actual) == 0
}

// initializingError returns an error indicating that the route is waiting for ACC,
// and sets the route's annotation to 'failed'
func (c *Controller) initializingError(cname string, route *v1.Route, a map[string]string) error {
	c.updateRouteAnnotation(route, a, initializing)
	return fmt.Errorf("Waiting for initial cname assignment - cname: %s, route: %s/%s", cname, route.Namespace, route.Name)
}

func (c *Controller) annotateFailed(err error, route *v1.Route, a map[string]string) error {
	c.updateRouteAnnotation(route, a, failed)
	return err
}

func (c *Controller) annotateComplete(route *v1.Route, a map[string]string) {
	c.updateRouteAnnotation(route, a, complete)
}
