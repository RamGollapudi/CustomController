# Kubernetes Controllers

In applications of robotics and automation, a control loop is a non-terminating loop
that regulates the state of the system. In Kubernetes, a controller is a control loop
that watches the shared state of the cluster through the API server and makes changes
attempting to move the current state towards the desired state.

## Simplistic Controller

```
for {
  desired := getDesiredState()
  current := getCurrentState()
  makeChanges(desired, current)
}
```

## Controller Components

### Informer/SharedInformer

Sends requests to Kubernetes API Server to retrieve an objects' data. To prevent
constant calls to the API Server, we integrate with a cache that can be shared
across informers (hence the name SharedInformer). This is useful whenever more
than one controller cares about a specific resource. For example, out of the box,
Kubernetes has many controllers that care about the Pod resource. SharedInformers
also come with built-in hooks to receive events for adding, updating, and deleting
a provided resource. There are three main components to an informer:

* ListWatcher - list/watch a given resource (ie pod, deployment, etc.)
* Resource Event Handler - how to handle add, update, delete events
* ResyncPeriod - how often to go through the cache, and trigger an update event for each remaining object

### Queue

To differentiate between informers, each controller has its own queue, which
allows a controller to keep track of it's own progress. Whenever an object get
modified, the informer's Resource Event Handler places it's key on the queue. The
controller has workers that pop the keys off the queue, get the object, and then
do any necessary processing. There are built in implementations of the queue,
such as a Rate Limiting Queue, which allow for useful features like exponential
backoff if processing an object goes wrong.

A combination of the queueing mechanisms and the cache resync capabilities help
ensure that any Kubernetes system is given every opportunity to become eventually
consistent with the desired state.

## What is the difference between Controllers and Operators?

Controllers that implement an API for a specific application, such as Etcd, Spark, or Cassandra are often referred to as Operators.

## How does this relate to Cluster API?

In it's own words:

"The Cluster API consists of a shared set of controllers in order to provide a 
consistent user experience. In order to support multiple cloud environments, 
some of these controllers (e.g. `Cluster`, `Machine`) call provider specific 
actuators to implement the behavior expected of the controller. Other 
controllers (e.g. `MachineSet`) are generic and operate by interacting with 
other Cluster API (and Kubernetes) resources.

The task of the provider implementor is to implement the actuator methods so 
that the shared controllers can call those methods when they must reconcile 
state and update status.

When an actuator function returns an error, if it is [RequeueAfterError](
https://github.com/kubernetes-sigs/cluster-api/blob/fa906f36843b065c5294501efe7d78ebd85c3c04/pkg/controller/error/requeue_error.go#L27) then the object will be
requeued for further processing after the given RequeueAfter time has
passed."

## References/Additional Resources
[K8s Controller Deep Dive](https://engineering.bitnami.com/articles/a-deep-dive-into-kubernetes-controllers.html)  
[K8s Internals Deep Dive](http://borismattijssen.github.io/articles/kubernetes-informers-controllers-reflectors-stores)  
[Sample Controller](https://github.com/kubernetes/sample-controller)  
[Understanding K8s Cache Package](https://lairdnelson.wordpress.com/2018/01/07/understanding-kubernetes-tools-cache-package-part-0/)  
