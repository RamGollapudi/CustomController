package controller

import (
	"fmt"
	"os"
	"strings"

	"github.com/openshift/api/route/v1"
	"xxxxxxgitrepoxxx"
	"k8s.io/klog"
)

// lookups: 1 to 1 mapping between router shard and RP
//          Key - router shard name, corresponds to project/namespace label
// 			Value - associated reverse proxy
func initializeLookups() map[string]string {

	shard2vips := strings.Split(os.Getenv("SHARD2VIP_LOOKUP"), "\n")

	lookups := make(map[string]string)
	for _, entry := range shard2vips {
		kv := strings.SplitN(entry, "=", 2)
		if len(kv) < 2 {
			continue
		}

		// AM response has '.' at end of each RP, example: "my-infra-vip.cisco.com."
		if !strings.HasSuffix(kv[1], ".") {
			kv[1] += "."
		}

		lookups[kv[0]] = kv[1]
	}

	klog.Infof("Populated the lookup table upon pod start-up: %+v", lookups)

	return lookups
}

// lookupRP retrieves the 'router' label stored on the route's namespace object.
// This label corresponds to the router shard used to lookup the correct RP.
// This method is guaranteed to return either a non-empty RP, or an error.
func (c *Controller) lookupRP(route *v1.Route) (string, error) {
	ns, err := c.NamespaceLister.Get(route.Namespace)
	if err != nil {
		return "", fmt.Errorf("Error fetching namespace object with key %s from cache: %v", route.Namespace, err)
	}

	routerLabel, labelExists := ns.Labels["router"]
	if !labelExists {
		return "", errortypes.Errorf("Missing router label on project - route: %s/%s", ns.Name, route.Name)
	}

	expected, lookupExists := lookups[routerLabel]
	if !lookupExists || len(expected) == 0 {
		return "", errortypes.Errorf("Missing routerShard to RP lookup - routerLabel: %s, route: %s/%s", routerLabel, route.Namespace, route.Name)
	}

	return expected, nil
}

func initializeBlackList() map[string]struct{} {
	bl := strings.Split(os.Getenv("BLACKLIST_HOSTS"), "\n")
	blacklist := make(map[string]struct{})
	for _, entry := range bl {
		if len(entry) > 0 {
			blacklist[entry] = struct{}{}
		}
	}
	return blacklist
}

// isBlackListed determines if a route is marked as blacklisted. These values are stored
// in a configMap to support a more dynamic ability to changes these hostnames.
func isBlackListed(cname string, route *v1.Route) bool {
	if _, blacklisted := blacklist[cname]; blacklisted {
		klog.Infof("Found a blacklisted host - cname: %s, route: %s/%s", cname, route.Namespace, route.Name)
		return true
	}
	return false
}
