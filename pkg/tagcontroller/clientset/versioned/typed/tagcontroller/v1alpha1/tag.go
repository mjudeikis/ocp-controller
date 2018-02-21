package v1alpha1

import (
	v1alpha1 "github.com/mjudeikis/ocp-controller/pkg/apis/tagcontroller/v1alpha1"
	scheme "github.com/mjudeikis/ocp-controller/pkg/tagcontroller/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// TagsGetter has a method to return a TagInterface.
// A group's client should implement this interface.
type TagsGetter interface {
	Tags(namespace string) TagInterface
}

// TagInterface has methods to work with Tag resources.
type TagInterface interface {
	Create(*v1alpha1.Tag) (*v1alpha1.Tag, error)
	Update(*v1alpha1.Tag) (*v1alpha1.Tag, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v1alpha1.Tag, error)
	List(opts v1.ListOptions) (*v1alpha1.TagList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.Tag, err error)
	TagExpansion
}

// tags implements TagInterface
type tags struct {
	client rest.Interface
	ns     string
}

// newTags returns a Tags
func newTags(c *DeploymentsV1alpha1Client, namespace string) *tags {
	return &tags{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the tag, and returns the corresponding tag object, and an error if there is any.
func (c *tags) Get(name string, options v1.GetOptions) (result *v1alpha1.Tag, err error) {
	result = &v1alpha1.Tag{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("tags").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of Tags that match those selectors.
func (c *tags) List(opts v1.ListOptions) (result *v1alpha1.TagList, err error) {
	result = &v1alpha1.TagList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("tags").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested tags.
func (c *tags) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("tags").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a tag and creates it.  Returns the server's representation of the tag, and an error, if there is any.
func (c *tags) Create(tag *v1alpha1.Tag) (result *v1alpha1.Tag, err error) {
	result = &v1alpha1.Tag{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("tags").
		Body(tag).
		Do().
		Into(result)
	return
}

// Update takes the representation of a tag and updates it. Returns the server's representation of the tag, and an error, if there is any.
func (c *tags) Update(tag *v1alpha1.Tag) (result *v1alpha1.Tag, err error) {
	result = &v1alpha1.Tag{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("tags").
		Name(tag.Name).
		Body(tag).
		Do().
		Into(result)
	return
}

// Delete takes name of the tag and deletes it. Returns an error if one occurs.
func (c *tags) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("tags").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *tags) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("tags").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched tag.
func (c *tags) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.Tag, err error) {
	result = &v1alpha1.Tag{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("tags").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
