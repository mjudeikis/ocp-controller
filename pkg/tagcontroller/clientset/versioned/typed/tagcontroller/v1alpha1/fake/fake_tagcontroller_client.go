package fake

import (
	v1alpha1 "github.com/mjudeikis/ocp-controller/pkg/tagcontroller/clientset/versioned/typed/tagcontroller/v1alpha1"
	rest "k8s.io/client-go/rest"
	testing "k8s.io/client-go/testing"
)

type FakeDeploymentsV1alpha1 struct {
	*testing.Fake
}

func (c *FakeDeploymentsV1alpha1) Tags(namespace string) v1alpha1.TagInterface {
	return &FakeTags{c, namespace}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakeDeploymentsV1alpha1) RESTClient() rest.Interface {
	var ret *rest.RESTClient
	return ret
}
