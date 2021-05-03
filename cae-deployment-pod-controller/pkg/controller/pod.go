package controller

import (
	"fmt"

	dcv1 "github.com/openshift/api/apps/v1"
	dv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"

	//v1core "k8s.io/api/core/v1"
	//resource "k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/klog"
)

const (
	gslbCacheKey = "GSLB_CACHE_KEY"
	ipCacheKey   = "INTERFACE_IP_CACHE_KEY"
)

var (
	lookups   = initializeLookups()
	blacklist = initializeBlackList()
)

// UpdateRoute is called whenever a new route is created or updated.
// This method will also be called during every cache resync (update).
// UpdateRoute will move routes' hostnames/cnames to the correct reverse
// proxy (RP). If they are already on the correct RP, no-op.

func getObjectType(obj interface{}) *v1.Pod {
	switch obj_type := obj.(type) {
	case *v1.Pod:
		obj_type = obj.(*v1.Pod)
		return obj_type
	default:
		obj_type = nil
	}
	return nil
}

func (c *Controller) UpdatePod(obj interface{}, isGlobalWatcher bool) error {
	//klog.Infof("Inside Pod.go")

	if obj != nil {
		podconfig := getObjectType(obj)
		if podconfig != nil {
			//podconfig := obj.(*v1.Pod)
			podconfigname := podconfig.GetObjectMeta().GetName()
			con_status := podconfig.Status.ContainerStatuses
			deploymentconfig_name := ""
			deployment_name := ""
			label_name := ""
			key := ""
			fmt.Println("ContainerStatus lenght: ", len(con_status))
			if len(con_status) > 0 {
				pod_restartcount := podconfig.Status.ContainerStatuses[0].RestartCount
				if pod_restartcount > 1 {
					key_namespace := podconfig.GetNamespace()
					pod_annotations := podconfig.GetObjectMeta().GetAnnotations()
					pod_labels := podconfig.GetObjectMeta().GetLabels()

					for k := range pod_annotations {
						//fmt.Printf("key[%s] value[%s]\n", k, pod_annotations[k])
						if k == "openshift.io/deployment-config.name" {
							deploymentconfig_name = pod_annotations[k]
							key = fmt.Sprintf("%s%s%s", key_namespace, "/", deploymentconfig_name)
						}
						if k == "openshift.io/deployment.name" {
							deployment_name = pod_annotations[k]
							key = fmt.Sprintf("%s%s%s", key_namespace, "/", deployment_name)
						}
					}

					if deploymentconfig_name == "" && deployment_name == "" {
						for k := range pod_labels {
							//fmt.Printf("key[%s] value[%s]\n", k, pod_annotations[k])
							if k == "name" {
								label_name = pod_labels[k]
								key = fmt.Sprintf("%s%s%s", key_namespace, "/", label_name)
							}
						}

					}
					klog.Infof("Key - %s", key)
					if key != "" {
						klog.Infof("-->PodName - %s, PodNamespace - %s, PodRestartCount - %v, PodDeploymentConfigName - %s, ", podconfigname, key_namespace, pod_restartcount, deploymentconfig_name)
						if key_namespace == "test-bh-alln-7nov" {
							klog.Infof("PodNamespace - %s, ConfigName - %s, ", key_namespace, deploymentconfig_name)

							//key_deploymentconfig := fmt.Sprintf("%s%s%s", key_namespace, "/", deploymentconfig_name)
							obj_deploymentconfig, dc_exists, err := c.DeploymentConfigInformer.GetIndexer().GetByKey(key)
							obj_deployment, d_exists, err := c.DeploymentInformer.GetIndexer().GetByKey(key)
							klog.Infof("DeploymentConfig? - %v , Deployment? - %v", dc_exists, d_exists)
							if dc_exists {
								deploymentconfig := obj_deploymentconfig.(*dcv1.DeploymentConfig)
								deploymentconfigname := deploymentconfig.GetObjectMeta().GetName()
								copy := deploymentconfig.DeepCopy()
								copy.Status.Replicas = 0
								copy.Spec.Replicas = 0

								klog.Infof("Deploymentconfig Key Name - %s, DeploymentConfigName - %s", key, deploymentconfigname)
								if err == nil {
								}
								klog.Infof("UpdatingDeploymentConfig")
								klog.Infof("Deploymentconfig Key Name - %s, DeploymentConfigName - %s", key, deploymentconfigname)
								klog.Infof("deploymentconfig.Namespace: %s", deploymentconfig.Namespace)

								c.DeploymentConfigClient.AppsV1().DeploymentConfigs(deploymentconfig.Namespace).Update(copy)
							} else if d_exists {
								deployment := obj_deployment.(*dv1.Deployment)
								deploymentname := deployment.GetObjectMeta().GetName()
								replica := int32(0)
								var replicacount *int32 = &replica
								copy := deployment.DeepCopy()
								copy.Spec.Replicas = replicacount
								copy.Status.Replicas = 0

								klog.Infof("Deployment Key Name - %s, DeploymentConfigName - %s", key, deploymentname)

								if err == nil {
								}
								klog.Infof("UpdatingDeployment")
								klog.Infof("Deployment Key Name - %s, DeploymentConfigName - %s", key, deploymentname)
								klog.Infof("deployment.Namespace: %s", deployment.Namespace)

								c.KubeClient.AppsV1().Deployments(deployment.Namespace).Update(copy)
							}

						}
					}
				}
			}
		}
	}
	return nil

}

// UpdateGlobalRoute fetches the service and monitor netscaler conifgurations for a given
// reverse proxy, and makes sure that their IP's correspond to expected IP's (whatever IP
// corresponds to the expected RP, fetched from AM). Will ONLY update the applicable IPs
// and serviceName, meaning all other configuration parameters will remain the same.
