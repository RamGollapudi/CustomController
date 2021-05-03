package main

import (
	"flag"
	"os"
	"strconv"
	"time"
	"custom git code"
	"k8s.io/klog"
	deploymentconfigv1client "github.com/openshift/client-go/apps/clientset/versioned"
	deploymentconfigv1factory "github.com/openshift/client-go/apps/informers/externalversions"
	gocache "github.com/patrickmn/go-cache"
	kubernetesfactory "k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"
	"k8s.io/client-go/kubernetes"
)

var (
	resyncPeriod       = getEnvDuration("RESYNC_PERIOD")
	globalResyncPeriod = getEnvDuration("GLOBAL_RESYNC_PERIOD")
	threads            = getEnvThreads("WORKER_THREADS")
	defaultResync      = 24 * time.Hour
	defaultThreads     = 8
)

func initLogs() {
	flag.Set("logtostderr", "true")
	flag.Set("alsologtostderr", "true")
	flag.Set("stderrthreshold", "INFO")
	flag.Set("v", "3")

	flags := flag.NewFlagSet("klog", flag.ExitOnError)
	klog.InitFlags(flags)
	flags.Set("logtostderr", "true")
	flags.Set("alsologtostderr", "true")
	flags.Set("stderrthreshold", "INFO")
	flags.Set("v", "3")
	flag.Parse()
}

func getDeploymentAndkubeClient() (*kubernetes.Clientset, *deploymentconfigv1client.Clientset) {

	// supports passing in a local configuration path for testing purposes
	// will return empty string if 'K8S_CONFIG_PATH' is not set, and default to SA
	kubeConfigPath := os.Getenv("K8S_CONFIG_PATH")

	// get the config object from the running pod's service account (or path)
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		klog.Fatalf("Failed to obtain kubernetes configuration: %v", err)
	}

	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		klog.Fatalf("Error building kubernetes client: %s", err.Error())
	}

	deploymentConfigClient, err := deploymentconfigv1client.NewForConfig(config)
	if err != nil {
		klog.Fatalf("Error building Deployment client: %s", err.Error())
	}

	klog.Info("Successfully constructed kubernetes, deployment and pod client")

	return kubeClient, deploymentConfigClient
}

func main() {

	initLogs()

	// expiration time of 60 minutes, purge expired items every 30 minutes
	gocache := gocache.New(60*time.Minute, 30*time.Minute)

	// get the Kubernetes client for connectivity to the API Server
	kubeClient, deploymentConfigClient := getDeploymentAndkubeClient()

	deploymentConfigInformerFactory := deploymentconfigv1factory.NewSharedInformerFactory(deploymentConfigClient, resyncPeriod)
	kubeInformerFactory := kubernetesfactory.NewSharedInformerFactory(kubeClient, resyncPeriod)
	//kubeInformerFactory := deploymentconfigv1factory.NewSharedInformerFactory(deploymentConfigClient, resyncPeriod)
	deploymentConfigInformer := deploymentConfigInformerFactory.Apps().V1().DeploymentConfigs().Informer()
	deploymentInformer := kubeInformerFactory.Apps().V1().Deployments().Informer()
	podInformer := kubeInformerFactory.Core().V1().Pods().Informer()

	namespaceLister := kubeInformerFactory.Core().V1().Namespaces().Lister() // TODO do I need to sync Lister cache too?

	// create a new queue so that when the informer gets a resource that is either
	// a result of listing or watching, we can add an idenfitying key to the queue
	// so that it can be handled in the handler
	deploymentconfigqueue := workqueue.NewNamedRateLimitingQueue(workqueue.NewItemExponentialFailureRateLimiter(5*time.Second, 300*time.Second), "deploymentcoinfigname")
	deploymentqueue := workqueue.NewNamedRateLimitingQueue(workqueue.NewItemExponentialFailureRateLimiter(5*time.Second, 300*time.Second), "deploymentname")
	podqueue := workqueue.NewNamedRateLimitingQueue(workqueue.NewItemExponentialFailureRateLimiter(5*time.Second, 300*time.Second), "podname")

	deploymentConfigInformer.AddEventHandlerWithResyncPeriod(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				deploymentconfigqueue.Add(key)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(newObj)
			if err == nil {
				deploymentconfigqueue.Add(key)
			}
		},
		DeleteFunc: func(obj interface{}) {
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err == nil {
				deploymentconfigqueue.Add(key)
			}
		},
	}, resyncPeriod)

	deploymentInformer.AddEventHandlerWithResyncPeriod(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				deploymentqueue.Add(key)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(newObj)
			if err == nil {
				deploymentqueue.Add(key)
			}
		},
		DeleteFunc: func(obj interface{}) {
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err == nil {
				deploymentqueue.Add(key)
			}
		},
	}, resyncPeriod)

	podInformer.AddEventHandlerWithResyncPeriod(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			//klog.Infof("Add Pod Key %s......", key)
			if err == nil {
				//klog.Infof("Adding Pod Key %s......", key)
				podqueue.Add(key)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(newObj)
			//klog.Infof("Update Pod Key %s......", key)
			if err == nil {
				//klog.Infof("Updateing Pod Key %s......", key)
				podqueue.Add(key)
			}
		},
		DeleteFunc: func(obj interface{}) {
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			//klog.Infof("Delete Pod Key %s......", key)
			if err == nil {
				//klog.Infof("Deleting Pod Key %s......", key)
				podqueue.Add(key)
			}
		},
	}, resyncPeriod)

	controller := controller.Controller{
		DeploymentConfigClient:   deploymentConfigClient,
		KubeClient:               kubeClient,
		DeploymentConfigInformer: deploymentConfigInformer,
		DeploymentInformer:       deploymentInformer,
		PodInformer:              podInformer,
		DeploymentConfigQueue:    deploymentconfigqueue,
		DeploymentQueue:          deploymentqueue,
		PodQueue:                 podqueue,
		NamespaceLister:          namespaceLister,
		Gocache:                  gocache,
	}

	// set up signals so we handle the first shutdown signal gracefully
	klog.Infof("Starting Informers......")
	stopCh := signals.SetupSignalHandler()
	kubeInformerFactory.Start(stopCh)
	deploymentConfigInformerFactory.Start(stopCh)
	kubeInformerFactory.Start(stopCh)

	if err := controller.Run(threads, stopCh); err != nil {
		klog.Fatalf("Error running Deployment controller: %s", err.Error())
	}
}

func getEnvDuration(key string) time.Duration {
	dur, err := time.ParseDuration(os.Getenv(key))
	if err != nil {
		klog.Fatalf("Error loading environment variable - key: %s, err: %v", key, err)
		return defaultResync
	}
	return dur
}

func getEnvThreads(key string) int {
	vint, err := strconv.Atoi(os.Getenv(key))
	if err != nil {
		klog.Fatalf("Error loading environment variable - key: %s, err: %v", key, err)
		return defaultThreads
	}
	return vint
}
