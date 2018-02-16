package tagcontroller

import (
	"fmt"
	"strconv"
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
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
)

const controllerAgentName = "tags-controller"

const (
	// SuccessSynced is used as part of the Event 'reason' when a Tag is synced
	SuccessSynced = "Synced"
	// ErrResourcePatchFailed is used as part of the Event 'reason' when a Tag fails
	// to sync due unknown reason
	ErrResourcePatchFailed = "ErrResourcePatchFailed"

	// ErrResourceExists is used as part of the Event 'reason' when a Foo fails
	// to sync due to a Deployment of the same name already existing.
	ErrResourceExists = "ErrResourceExists"

	// MessageResourceExists is the message used for Events when a resource
	// fails to sync due to a Deployment already existing
	MessageResourceExists = "Resource %q already exists and is not managed by Tag"

	// MessageResourcePatchFailed is the message used for Events when a resource
	// fails to sync due some unknown reasons
	MessageResourcePatchFailed = "Resource %q patch failed with error %v"
	// MessageResourceSynced is the message used for an Event fired when a Foo
	// is synced successfully
	MessageResourceSynced = "Tags synced successfully"

	//Tag ownership annotation
	TagOwnershipAnnotation = "deployments.origin.io/tags"
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

// NewController returns a controller
func NewController(
	originClientset apps.Interface,
	kubeClientset kubernetes.Interface,
	tagClientset tagclientset.Interface,
	originInformerFactory appsinformers.SharedInformerFactory,
	tagInformerFactory taginformers.SharedInformerFactory) *Controller {

	// obtain references to shared index informers for the DeploymentConfig and Tag
	// types.
	deploymentInformer := originInformerFactory.Apps().V1().DeploymentConfigs()
	tagInformer := tagInformerFactory.Deployments().V1alpha1().Tags()

	// Create event broadcaster
	// Add tag-controller types to the default Kubernetes Scheme so Events can be
	// logged for tag-controller types.
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
	// Set up an event handler for when Tag resources change
	tagInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueTag,
		UpdateFunc: func(old, new interface{}) {
			controller.enqueueTag(new)
		},
	})
	// Set up an event handler for when Deployment resources change. This
	// handler will lookup DeploymentConfig and images they are using and if image
	// is different than tag resource, it will patch it.
	//  This way, we don't need to implement custom logic for
	// handling DeploymentConfig resources for images. More info on this pattern:
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
		// Tag resource to be synced.
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
// converge the two. It then updates the Status block of the Tag resource
// with the current status (tag deployed) of the resource.
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
		// The Tag resource may no longer exist, in which case we stop
		// processing.
		if errors.IsNotFound(err) {
			runtime.HandleError(fmt.Errorf("tag object '%s' in work queue no longer exists", key))
			return nil
		}

		return err
	}

	deploymentName := tag.Spec.DeploymentConfigName
	glog.V(4).Infof("deploymentConfig %v is used by tag resource %v", deploymentName, name)
	if deploymentName == "" || tag.Spec.ImageTag == "" || tag.Spec.ContainerName == "" {
		// We choose to absorb the error here as the worker would requeue the
		// resource otherwise. Instead, the next time the resource is updated
		// the resource will be queued again.
		runtime.HandleError(fmt.Errorf("%s: deployment name must be specified", key))
		return nil
	}

	// Get the deployment with the name specified in Foo.spec
	deployment, err := c.appsLister.DeploymentConfigs(tag.Namespace).Get(deploymentName)
	// If the resource doesn't exist, we'll create it
	if errors.IsNotFound(err) {
		runtime.HandleError(fmt.Errorf("%s: deployment does not exist. Nothing to track", key))
		return nil
	}

	// If the DeploymentConfig is not controlled by this Tag resource, we should log
	// a warning to the event recorder and retry
	if ownerAnnotations := deployment.GetAnnotations(); len(ownerAnnotations[TagOwnershipAnnotation]) <= 0 {
		glog.V(4).Infof("deploymentConfig %v is not owned by %v", deploymentName, name)
		msg := fmt.Sprintf(MessageResourceExists, deployment.Name)
		c.recorder.Event(tag, corev1.EventTypeWarning, ErrResourceExists, msg)
		return fmt.Errorf(msg)
	}

	//get containers section from DeploymentConfig
	containers := deployment.Spec.Template.Spec.Containers
	imageTrigger := deployment.Spec.Triggers
	newImage := fmt.Sprintf("%s:%s", tag.Spec.ContainerName, tag.Spec.ImageTag)
	var containersKey, triggerKey interface{}

	//get containers section key if match found
	for key, item := range containers {
		if item.Name == tag.Spec.ContainerName {
			containersKey = key
			glog.V(4).Infof("DeploymentConfig has required image %v", item.Name)
		}
	}

	for key, item := range imageTrigger {
		if item.ImageChangeParams == nil {
			continue
		}
		for _, item2 := range item.ImageChangeParams.ContainerNames {
			if item2 == tag.Spec.ContainerName {
				triggerKey = key
				glog.V(4).Infof("DeploymentConfig has required trigger for image %v", item2)
			}
		}

	}
	if containersKey != nil && triggerKey != nil {
		//todo add imagestreamtag check for trigger if we have it set
		if deployment.Spec.Triggers[triggerKey.(int)].ImageChangeParams.From.Name != newImage {
			glog.V(4).Infof("Patching DeploymentConfig: %v to point to %v", deploymentName, tag.Spec.ImageTag)
			deployment.Spec.Template.Spec.Containers[containersKey.(int)].Image = " "
			deployment.Spec.Triggers[triggerKey.(int)].ImageChangeParams.From.Name = tag.Spec.ContainerName + ":" + tag.Spec.ImageTag
			//add owner ref so we know how owns this stuff :)
			deployment.OwnerReferences = []metav1.OwnerReference{
				*metav1.NewControllerRef(tag, schema.GroupVersionKind{
					Group:   tagv1alpha1.SchemeGroupVersion.Group,
					Version: tagv1alpha1.SchemeGroupVersion.Version,
					Kind:    "Tag",
				})}

			deployment, err = c.originClientset.AppsV1().DeploymentConfigs(tag.Namespace).Update(deployment)
			if err != nil {
				runtime.HandleError(fmt.Errorf("%v: deployment patch failed with error %v", deploymentName, err.Error()))
				msg := fmt.Sprintf(MessageResourcePatchFailed, deploymentName, err.Error())
				c.recorder.Event(tag, corev1.EventTypeWarning, ErrResourcePatchFailed, msg)
				return fmt.Errorf(msg)
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
// to find the Tag resource that 'owns' it. It does this by looking at the
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
	if ownerAnnotations := object.GetAnnotations(); ownerAnnotations[TagOwnershipAnnotation] != "" {
		// If this object does not have annotation for ownership we should not do anything
		// with it.
		glog.V(4).Infof("Checking owner annotations for %v", object.GetName())
		//if annotation is not set to true we dropthis DC
		if s, err := strconv.ParseBool(ownerAnnotations[TagOwnershipAnnotation]); err == nil && !s {
			glog.V(4).Infof("deployment %v annotation is not set to true", object.GetName())
			return
		}

		tags, err := c.tagsLister.Tags(object.GetNamespace()).List(labels.Everything())
		if err != nil {
			glog.V(4).Infof("ignoring orphaned object '%s' of tag '%s'", object.GetSelfLink(), object.GetName())
			return
		}

		for key, tag := range tags {
			//we found tag which want to own this DC - Congradz
			if tag.Spec.DeploymentConfigName == object.GetName() {
				glog.V(4).Infof("enqueue tag %v for processing with %v deployment", tag.Name, object.GetName())
				c.enqueueTag(tags[key])
			}
		}
		return
	}
}
