package fake

import (
	v1alpha1 "github.com/mjudeikis/ocp-controller/pkg/apis/samplecontroller/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeFoos implements FooInterface
type FakeFoos struct {
	Fake *FakeSamplecontrollerV1alpha1
	ns   string
}

var foosResource = schema.GroupVersionResource{Group: "samplecontroller.k8s.io", Version: "v1alpha1", Resource: "foos"}

var foosKind = schema.GroupVersionKind{Group: "samplecontroller.k8s.io", Version: "v1alpha1", Kind: "Foo"}

// Get takes name of the foo, and returns the corresponding foo object, and an error if there is any.
func (c *FakeFoos) Get(name string, options v1.GetOptions) (result *v1alpha1.Foo, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(foosResource, c.ns, name), &v1alpha1.Foo{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Foo), err
}

// List takes label and field selectors, and returns the list of Foos that match those selectors.
func (c *FakeFoos) List(opts v1.ListOptions) (result *v1alpha1.FooList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(foosResource, foosKind, c.ns, opts), &v1alpha1.FooList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.FooList{}
	for _, item := range obj.(*v1alpha1.FooList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested foos.
func (c *FakeFoos) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(foosResource, c.ns, opts))

}

// Create takes the representation of a foo and creates it.  Returns the server's representation of the foo, and an error, if there is any.
func (c *FakeFoos) Create(foo *v1alpha1.Foo) (result *v1alpha1.Foo, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(foosResource, c.ns, foo), &v1alpha1.Foo{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Foo), err
}

// Update takes the representation of a foo and updates it. Returns the server's representation of the foo, and an error, if there is any.
func (c *FakeFoos) Update(foo *v1alpha1.Foo) (result *v1alpha1.Foo, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(foosResource, c.ns, foo), &v1alpha1.Foo{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Foo), err
}

// Delete takes name of the foo and deletes it. Returns an error if one occurs.
func (c *FakeFoos) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(foosResource, c.ns, name), &v1alpha1.Foo{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeFoos) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(foosResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &v1alpha1.FooList{})
	return err
}

// Patch applies the patch and returns the patched foo.
func (c *FakeFoos) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.Foo, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(foosResource, c.ns, name, data, subresources...), &v1alpha1.Foo{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Foo), err
}
