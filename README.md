# Openshift Controller with CRD Example

This repository shows how we can use [CRD](https://kubernetes.io/docs/concepts/api-extension/custom-resources/), [operators](https://coreos.com/operators/) and [controllers](https://github.com/kubernetes/community/blob/master/contributors/devel/controllers.md) design patterns.

In this repository we create new CRD with name `Tag`. This resource referre `DeploymentConfig` name, container name, and image tag. Without `Controller` pattern this resource is useless . We monitor this CRD object using [dynamically generated](https://github.com/kubernetes/code-generator) client. This allows us to take actions based on resource changes. In this example we patch `DeploymentConfig` to deploy image tag from object `Tag`.  This is trivial example, just to show possible functionality and capabilities of this pattern.

## Usage Flags

`--kubeconfig ~/.kube/config` -  Uses ~/.kube/config rather than in cluster configuration
`-v 4 -stderrthreshold=info` - Logging level

### Flow

1. CRD resource is created
2. New `Tag` resource is created, pointing to existing or non existing `DeploymentConfig`

3. `DeploymentConfig` is annotated to tell controller that it can be `owned` by `Tag` resource.

```bash
oc annotate dc httpd-example deployments.origin.io/tags=True
```

4. If `Tag` resource `Spec` data is matched with `DeploymentConfig` data DC is patched. When resource is patched, ownership metadata is updated:

```yaml
ownerReferences:
    - apiVersion: deployments.origin.io/v1alpha1
      blockOwnerDeletion: true
      controller: true
      kind: Tag
      name: httpd-example
      uid: aa3e3830-1634-11e8-8eb2-52540008e27c
```

5. If any of the resource is being Updated/Deleted/Created - Controller will check if it needs to do anything with resource state.

## Development

### Build from source

1. `make install_deps`
2. `make build`
3. `./bin/tagcontroller --kubeconfig ~/.kube/config`

### Regenerate CRD package client

```bash
make update-codegen
```

### Dev notes

Start controller outside cluster:

```bash
/data/go/src/github.com/mjudeikis/ocp-controller/bin/tagcontroller --kubeconfig ~/openshift.local.config/master/admin.kubeconfig -v 4 -stderrthreshold=info
```

Start controller in the cluster:

```bash
oc create -f /data/go/src/github.com/mjudeikis/ocp-controller/artifacts/controller.yaml
```

Create example applications:

```bash
oc process template/httpd-example -n openshift -p NAME=httpd-example-two| oc create -f -
oc process template/httpd-example -n openshift -p NAME=httpd-example| oc create -f -
```

Annotate DeploymentConfigs to ne "tracked":

```bash
oc annotate dc httpd-example deployments.origin.io/tags=True
oc annotate dc httpd-example-two deployments.origin.io/tags=True
```

Create CRD resource:

```bash
oc create -f /data/go/src/github.com/mjudeikis/ocp-controller/artifacts/crd.yaml
```

Create Tag resources:

```bash
oc create -f /data/go/src/github.com/mjudeikis/ocp-controller/artifacts/example-tag-one.yaml
oc create -f /data/go/src/github.com/mjudeikis/ocp-controller/artifacts/example-tag-two.yaml
```