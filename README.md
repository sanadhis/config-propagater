# k8s-config-propagator
K8s operator that address common needs in multi-namespaces cluster management: propagating common secret/configmap among namespaces.
Example use case:
- propagating CA trust certificates to all namespaces in the cluster.
- propagating common configuration (configmap) to all namespaces in the cluster.

## Getting Started
Deploy this controller to your cluster and create instances of your solution.
```
k apply -f https://raw.githubusercontent.com/sanadhis/config-propagator/<tag or branch>/dist/install.yaml
```

Tag secret/configmap with annotations:
```
apiVersion: v1
kind: ConfigMap
metadata:
  annotations:
    config-propagator.sanadhis.com/propagate: "true"
  name: config-to-propagate
  namespace: default
data:
  example-key: "example-value"
```

see the sample in `config/samples` for more details.

### Available Annotations
- `config-propagator.sanadhis.com/propagate`: Set to "true" to indicate that this secret/configmap should be propagated or not.
- `config-propagator.sanadhis.com/target-namespaces` (optional): A comma-separated list of target namespaces to which this secret/configmap should be propagated. If not specified, it will be propagated to all namespaces in the cluster.

## Developing the controller

### Prerequisites
- go version v1.24.0+
- docker version 17.03+.
- kubectl version v1.33.0+.
- Access to a Kubernetes v1.33.0+ cluster.

### To Deploy on the local cluster
**Build and push your image to the location specified by `IMG`:**

```sh
make docker-build docker-push IMG=docker.io/sanadhis/config-propagator:local
```

**Deploy the Manager to the cluster with the image specified by `IMG`:**

```sh
make deploy IMG=docker.io/sanadhis/config-propagator:local
```

> **NOTE**: If you encounter RBAC errors, you may need to grant yourself cluster-admin
privileges or be logged in as admin.

**Create instances of your solution**
You can apply the samples (examples) from the config/sample:

```sh
k apply -k config/samples/
```

### To Uninstall
**Delete the instances (CRs) from the cluster:**

```sh
k delete -k config/samples/
```

**UnDeploy the controller from the cluster:**

```sh
make undeploy
```

## Project Distribution

Following the options to release and provide this solution to the users.

### By providing a bundle with all YAML files

1. Build the installer for the image built and published in the registry:

```sh
make build-installer IMG=docker.io/sanadhis/config-propagator:tag
```

**NOTE:** The makefile target mentioned above generates an 'install.yaml'
file in the dist directory. This file contains all the resources built
with Kustomize, which are necessary to install this project without its
dependencies.

2. Using the installer

Users can just run 'k apply -f <URL for YAML BUNDLE>' to install
the project, i.e.:

```sh
k apply -f https://raw.githubusercontent.com/sanadhis/config-propagator/<tag or branch>/dist/install.yaml
```

## License

MIT.
