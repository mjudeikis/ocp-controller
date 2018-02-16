package fake

import (
	oauth_v1 "github.com/openshift/api/oauth/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeOAuthAuthorizeTokens implements OAuthAuthorizeTokenInterface
type FakeOAuthAuthorizeTokens struct {
	Fake *FakeOauthV1
}

var oauthauthorizetokensResource = schema.GroupVersionResource{Group: "oauth.openshift.io", Version: "v1", Resource: "oauthauthorizetokens"}

var oauthauthorizetokensKind = schema.GroupVersionKind{Group: "oauth.openshift.io", Version: "v1", Kind: "OAuthAuthorizeToken"}

// Get takes name of the oAuthAuthorizeToken, and returns the corresponding oAuthAuthorizeToken object, and an error if there is any.
func (c *FakeOAuthAuthorizeTokens) Get(name string, options v1.GetOptions) (result *oauth_v1.OAuthAuthorizeToken, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(oauthauthorizetokensResource, name), &oauth_v1.OAuthAuthorizeToken{})
	if obj == nil {
		return nil, err
	}
	return obj.(*oauth_v1.OAuthAuthorizeToken), err
}

// List takes label and field selectors, and returns the list of OAuthAuthorizeTokens that match those selectors.
func (c *FakeOAuthAuthorizeTokens) List(opts v1.ListOptions) (result *oauth_v1.OAuthAuthorizeTokenList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(oauthauthorizetokensResource, oauthauthorizetokensKind, opts), &oauth_v1.OAuthAuthorizeTokenList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &oauth_v1.OAuthAuthorizeTokenList{}
	for _, item := range obj.(*oauth_v1.OAuthAuthorizeTokenList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested oAuthAuthorizeTokens.
func (c *FakeOAuthAuthorizeTokens) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(oauthauthorizetokensResource, opts))
}

// Create takes the representation of a oAuthAuthorizeToken and creates it.  Returns the server's representation of the oAuthAuthorizeToken, and an error, if there is any.
func (c *FakeOAuthAuthorizeTokens) Create(oAuthAuthorizeToken *oauth_v1.OAuthAuthorizeToken) (result *oauth_v1.OAuthAuthorizeToken, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(oauthauthorizetokensResource, oAuthAuthorizeToken), &oauth_v1.OAuthAuthorizeToken{})
	if obj == nil {
		return nil, err
	}
	return obj.(*oauth_v1.OAuthAuthorizeToken), err
}

// Update takes the representation of a oAuthAuthorizeToken and updates it. Returns the server's representation of the oAuthAuthorizeToken, and an error, if there is any.
func (c *FakeOAuthAuthorizeTokens) Update(oAuthAuthorizeToken *oauth_v1.OAuthAuthorizeToken) (result *oauth_v1.OAuthAuthorizeToken, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(oauthauthorizetokensResource, oAuthAuthorizeToken), &oauth_v1.OAuthAuthorizeToken{})
	if obj == nil {
		return nil, err
	}
	return obj.(*oauth_v1.OAuthAuthorizeToken), err
}

// Delete takes name of the oAuthAuthorizeToken and deletes it. Returns an error if one occurs.
func (c *FakeOAuthAuthorizeTokens) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteAction(oauthauthorizetokensResource, name), &oauth_v1.OAuthAuthorizeToken{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeOAuthAuthorizeTokens) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(oauthauthorizetokensResource, listOptions)

	_, err := c.Fake.Invokes(action, &oauth_v1.OAuthAuthorizeTokenList{})
	return err
}

// Patch applies the patch and returns the patched oAuthAuthorizeToken.
func (c *FakeOAuthAuthorizeTokens) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *oauth_v1.OAuthAuthorizeToken, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(oauthauthorizetokensResource, name, data, subresources...), &oauth_v1.OAuthAuthorizeToken{})
	if obj == nil {
		return nil, err
	}
	return obj.(*oauth_v1.OAuthAuthorizeToken), err
}
