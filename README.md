# Openshift Controller with CRD

ToDo: Write sumup for how to use it

## Usage
`--kubeconfig ~/.kube/config  # Uses ~/.kube/config rather than in cluster configuration`

Start in debug mode:
```
/data/go/src/github.com/mjudeikis/ocp-controller/bin/ocp-controller --kubeconfig ~/openshift.local.config/master/admin.kubeconfig -v 4 -stderrthreshold=info
```



## Development

### Build from source
1. `make install_deps`
2. `make build`
3. `./bin/ocp-controller --kubeconfig ~/.kube/config `


### Regenerate CRD package client
```
make update-codegen
```

### ToDo:

1. Create CRD definition for Image version with name of DC's (list)
2. Create watcher for CRD and patch DC on CRD data.
3. Split to separate API object for all controllers.
4. Move DC controller to same logic as k8s upstream


### Crete CRD:

`oc create -f artifacts/crd.yaml`

### Create instance of our CRD:

`/data/go/src/github.com/mjudeikis/ocp-controller/artifacts/exmple-foo.yaml`