package tagcontroller

import (
	"fmt"
	"time"

	"github.com/golang/glog"
	tagv1alpha1 "github.com/mjudeikis/ocp-controller/pkg/apis/tagcontroller/v1alpha1"
	tagclientset "github.com/mjudeikis/ocp-controller/pkg/tagcontroller/clientset/versioned"
	tagscheme "github.com/mjudeikis/ocp-controller/pkg/tagcontroller/clientset/versioned/scheme"
	taginformers "github.com/mjudeikis/ocp-controller/pkg/tagcontroller/informers/externalversions"
	taglisters "github.com/mjudeikis/ocp-controller/pkg/tagcontroller/listers/tagcontroller/v1alpha1"
	appsv1 "github.com/openshift/api/apps/v1"
	apps "github.com/openshift/client-go/apps/clientset/versioned"
	appsinformers "github.com/openshift/client-go/apps/informers/externalversions"
	appslisters "github.com/openshift/client-go/apps/listers/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
)

const controllerAgentName = "tag-controller"

const (
	// SuccessSynced is used as part of the Event 'reason' when a Foo is synced
	SuccessSynced = "Synced"
	// ErrResourceExists is used as part of the Event 'reason' when a Foo fails
	// to sync due to a Deployment of the same name already existing.
	ErrResourceExists = "ErrResourceExists"

	// MessageResourceExists is the message used for Events when a resource
	// fails to sync due to a Deployment already existing
	MessageResourceExists = "Resource %q already exists and is not managed by Tag Controller"
	// MessageResourceSynced is the message used for an Event fired when a Foo
	// is synced successfully
	MessageResourceSynced = "Tags synced successfully"
)

// Controller is the controller implementation for Foo resources
type Controller struct {
	// originClientset is standart apps/dc origin clientset
	originClientset apps.Interface
	// kubeclientset is a standard kubernetes clientset
	kubeClientset kubernetes.Interface
	// sampleclientset is a clientset for our own API group
	tagClientset tagclientset.Interface

	//todo Change to DC listener
	appsLister        appslisters.DeploymentConfigLister
	deploymentsSynced cache.InformerSynced
	tagsLister        taglisters.TagLister
	tagsSynced        cache.InformerSynced

	// workqueue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	workqueue workqueue.RateLimitingInterface
	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder record.EventRecorder
}

// NewController returns a new sample controller
func NewController(
	originClientset apps.Interface,
	kubeClientset kubernetes.Interface,
	tagClientset tagclientset.Interface,
	originInformerFactory appsinformers.SharedInformerFactory,
	tagInformerFactory taginformers.SharedInformerFactory) *Controller {

	// obtain references to shared index informers for the Deployment and Foo
	// types.
	deploymentInformer := originInformerFactory.Apps().V1().DeploymentConfigs()
	tagInformer := tagInformerFactory.Deployments().V1alpha1().Tags()

	// Create event broadcaster
	// Add sample-controller types to the default Kubernetes Scheme so Events can be
	// logged for sample-controller types.
	tagscheme.AddToScheme(scheme.Scheme)
	glog.V(4).Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(glog.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeClientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	controller := &Controller{
		originClientset:   originClientset,
		kubeClientset:     kubeClientset,
		tagClientset:      tagClientset,
		appsLister:        deploymentInformer.Lister(),
		deploymentsSynced: deploymentInformer.Informer().HasSynced,
		tagsLister:        tagInformer.Lister(),
		tagsSynced:        tagInformer.Informer().HasSynced,
		workqueue:         workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Foos"),
		recorder:          recorder,
	}

	glog.Info("Setting up event handlers")
	// Set up an event handler for when Foo resources change
	tagInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueTag,
		UpdateFunc: func(old, new interface{}) {
			controller.enqueueTag(new)
		},
	})
	// Set up an event handler for when Deployment resources change. This
	// handler will lookup the owner of the given Deployment, and if it is
	// owned by a Foo resource will enqueue that Foo resource for
	// processing. This way, we don't need to implement custom logic for
	// handling Deployment resources. More info on this pattern:
	// https://github.com/kubernetes/community/blob/8cafef897a22026d42f5e5bb3f104febe7e29830/contributors/devel/controllers.md
	deploymentInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.handleObject,
		UpdateFunc: func(old, new interface{}) {
			newDepl := new.(*appsv1.DeploymentConfig)
			oldDepl := old.(*appsv1.DeploymentConfig)
			if newDepl.ResourceVersion == oldDepl.ResourceVersion {
				// Periodic resync will send update events for all known Deployments.
				// Two different versions of the same Deployment will always have different RVs.
				return
			}
			controller.handleObject(new)
		},
		DeleteFunc: controller.handleObject,
	})

	return controller
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) error {
	defer runtime.HandleCrash()
	defer c.workqueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	glog.Info("Starting Tag controller")

	// Wait for the caches to be synced before starting workers
	glog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.deploymentsSynced, c.tagsSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	glog.Info("Starting workers")
	// Launch two workers to process Foo resources
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	glog.Info("Started workers")
	<-stopCh
	glog.Info("Shutting down workers")

	return nil
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *Controller) runWorker() {
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (c *Controller) processNextWorkItem() bool {
	obj, shutdown := c.workqueue.Get()

	if shutdown {
		return false
	}

	// We wrap this block in a func so we can defer c.workqueue.Done.
	err := func(obj interface{}) error {
		// We call Done here so the workqueue knows we have finished
		// processing this item. We also must remember to call Forget if we
		// do not want this work item being re-queued. For example, we do
		// not call Forget if a transient error occurs, instead the item is
		// put back on the workqueue and attempted again after a back-off
		// period.
		defer c.workqueue.Done(obj)
		var key string
		var ok bool
		// We expect strings to come off the workqueue. These are of the
		// form namespace/name. We do this as the delayed nature of the
		// workqueue means the items in the informer cache may actually be
		// more up to date that when the item was initially put onto the
		// workqueue.
		if key, ok = obj.(string); !ok {
			// As the item in the workqueue is actually invalid, we call
			// Forget here else we'd go into a loop of attempting to
			// process a work item that is invalid.
			c.workqueue.Forget(obj)
			runtime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}
		// Run the syncHandler, passing it the namespace/name string of the
		// Foo resource to be synced.
		if err := c.syncHandler(key); err != nil {
			return fmt.Errorf("error syncing '%s': %s", key, err.Error())
		}
		// Finally, if no error occurs we Forget this item so it does not
		// get queued again until another change happens.
		c.workqueue.Forget(obj)
		glog.Infof("Successfully synced '%s'", key)
		return nil
	}(obj)

	if err != nil {
		runtime.HandleError(err)
		return true
	}

	return true
}

// syncHandler compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the Foo resource
// with the current status of the resource.
func (c *Controller) syncHandler(key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		runtime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	// Get the Tag resource with this namespace/name
	tag, err := c.tagsLister.Tags(namespace).Get(name)
	if err != nil {
		// The Foo resource may no longer exist, in which case we stop
		// processing.
		if errors.IsNotFound(err) {
			runtime.HandleError(fmt.Errorf("foo '%s' in work queue no longer exists", key))
			return nil
		}

		return err
	}

	deploymentName := tag.Spec.DeploymentConfigName
	//deploymentImageName := tag.Spec.ContainerName
	glog.V(4).Infof("deployment from tag object %v", deploymentName)
	if deploymentName == "" {
		// We choose to absorb the error here as the worker would requeue the
		// resource otherwise. Instead, the next time the resource is updated
		// the resource will be queued again.
		runtime.HandleError(fmt.Errorf("%s: deployment name must be specified", key))
		return nil
	}
	//TODO add more validation for other fields

	// Get the deployment with the name specified in Foo.spec
	deployment, err := c.appsLister.DeploymentConfigs(tag.Namespace).Get(deploymentName)
	// If the resource doesn't exist, we'll create it
	if errors.IsNotFound(err) {
		runtime.HandleError(fmt.Errorf("%s: deployment does not exist. Nothing to track", key))
	}

	//get containers section from deploymentConfig
	containers := deployment.Spec.Template.Spec.Containers
	imageTrigger := deployment.Spec.Triggers
	newImage := tag.Spec.ContainerName + ":" + tag.Spec.ImageTag
	var containersKey, triggerKey interface{}

	//get containers section key if match found
	for key, item := range containers {
		if item.Name == tag.Spec.ContainerName {
			containersKey = key
			glog.V(4).Infof("deployment has container for image %v %v", containersKey, item.Name)
		}
	}

	for key, item := range imageTrigger {
		if item.ImageChangeParams == nil {
			continue
		}
		for _, item2 := range item.ImageChangeParams.ContainerNames {
			if item2 == tag.Spec.ContainerName {
				triggerKey = key
				glog.V(4).Infof("deployment has trigger for image %v %v", triggerKey, item2)
			}
		}

	}

	if containersKey != nil && triggerKey != nil {
		//todo add imagestreamtag check for trigger if we have it set
		if deployment.Spec.Triggers[triggerKey.(int)].ImageChangeParams.From.Name != newImage {
			glog.V(4).Infof("Patching deployment: %v to point to %v", deploymentName, tag.Spec.ImageTag)

			deployment.Spec.Template.Spec.Containers[containersKey.(int)].Image = " "
			deployment.Spec.Triggers[triggerKey.(int)].ImageChangeParams.From.Name = tag.Spec.ContainerName + ":" + tag.Spec.ImageTag

			deployment, err = c.originClientset.AppsV1().DeploymentConfigs(tag.Namespace).Update(deployment)
			if err != nil {
				runtime.HandleError(fmt.Errorf("%v: deployment pathc failed", deploymentName))
				return err
			}
			err = c.updateTagStatus(tag, deployment)
			if err != nil {
				return err
			}

			c.recorder.Event(tag, corev1.EventTypeNormal, SuccessSynced, MessageResourceSynced)

			glog.V(4).Infof("Current Deployment %v", deployment)
			return nil
		}
		glog.V(4).Infof("Patching is not required: %v already point to %v", deploymentName, tag.Spec.ImageTag)
	}

	// If an error occurs during Get/Create, we'll requeue the item so we can
	// attempt processing again later. This could have been caused by a
	// temporary network failure, or any other transient reason.

	// If the Deployment is not controlled by this Foo resource, we should log
	// a warning to the event recorder and ret
	//if !metav1.IsControlledBy(deployment, foo) {
	//	msg := fmt.Sprintf(MessageResourceExists, deployment.Name)
	//	c.recorder.Event(foo, corev1.EventTypeWarning, ErrResourceExists, msg)
	//	return fmt.Errorf(msg)
	//}

	// If an error occurs during Update, we'll requeue the item so we can
	// attempt processing again later. THis could have been caused by a
	// temporary network failure, or any other transient reason.
	return nil
}

func (c *Controller) updateTagStatus(tag *tagv1alpha1.Tag, deployment *appsv1.DeploymentConfig) error {
	// NEVER modify objects from the store. It's a read-only, local cache.
	// You can use DeepCopy() to make a deep copy of original object and modify this copy
	// Or create a copy manually for better performance
	tagCopy := tag.DeepCopy()
	tagCopy.Status.DeployedTag = tag.Spec.ImageTag
	// Until #38113 is merged, we must use Update instead of UpdateStatus to
	// update the Status block of the Foo resource. UpdateStatus will not
	// allow changes to the Spec of the resource, which is ideal for ensuring
	// nothing other than resource status has been updated.
	_, err := c.tagClientset.DeploymentsV1alpha1().Tags(tag.Namespace).Update(tagCopy)
	return err
}

// enqueueTag takes a Tag resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than Foo.
func (c *Controller) enqueueTag(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		runtime.HandleError(err)
		return
	}
	c.workqueue.AddRateLimited(key)
}

// handleObject will take any resource implementing metav1.Object and attempt
// to find the Foo resource that 'owns' it. It does this by looking at the
// objects metadata.ownerReferences field for an appropriate OwnerReference.
// It then enqueues that Foo resource to be processed. If the object does not
// have an appropriate OwnerReference, it will simply be skipped.
func (c *Controller) handleObject(obj interface{}) {
	var object metav1.Object
	var ok bool
	if object, ok = obj.(metav1.Object); !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			runtime.HandleError(fmt.Errorf("error decoding object, invalid type"))
			return
		}
		object, ok = tombstone.Obj.(metav1.Object)
		if !ok {
			runtime.HandleError(fmt.Errorf("error decoding object tombstone, invalid type"))
			return
		}
		glog.V(4).Infof("Recovered deleted object '%s' from tombstone", object.GetName())
	}
	glog.V(4).Infof("Processing object: %s", object.GetName())
	if ownerRef := metav1.GetControllerOf(object); ownerRef != nil {
		// If this object is not owned by a Foo, we should not do anything more
		// with it.
		if ownerRef.Kind != "Foo" {
			return
		}

		foo, err := c.tagsLister.Tags(object.GetNamespace()).Get(ownerRef.Name)
		if err != nil {
			glog.V(4).Infof("ignoring orphaned object '%s' of foo '%s'", object.GetSelfLink(), ownerRef.Name)
			return
		}

		c.enqueueTag(foo)
		return
	}
}