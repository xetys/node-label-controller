
# Node Label Controller

This tiny kubernetes controller makes sure, that all nodes in a cluster running Container Linux
will have a label `"kubermatic.io/uses-container-linux"='true'` on it.

## Setup

To for testing purposes, install [Container Linux Update Operator](https://github.com/coreos/container-linux-update-operator)
first, using the packaged manifests:

```shell script
$ kubectl apply -f deploy/container-linux-update-operator/00-namespace.yaml
$ kubectl apply -f deploy/container-linux-update-operator/rbac
$ kubectl apply -f deploy/container-linux-update-operator/
```

You should discover a starting operator pod, while the agent is still not showing up

Then install the node-label-controller

```shell script
$ kubectl apply -f deploy/node-label-controller
```

As soon the controller runs, it should label all Container Linux node in the cluster, and
thus make the agent spawn

