# DEPRECATED

Since Cilium now does this for all my use-cases, this project is abandonware

# Vraquier

A poor-man "load-balancer" controller for kubernetes clusters without a cloud-controller. For when even kube-vip or metalLB are to much.
It replicate the k3s behaviour by starting `klipper-lb` DaemonSet.

## Requierements

kubelet should be started with `--allowed-unsafe-sysctls 'net.ipv6.conf.all.forwarding,net.ipv4.ip_forward' --provider-id 'vraquier://<node name>' --cloud-provider=external`

## Installation

On master nodes create a manifests for vraquier as follow:
```
---
apiVersion: v1
kind: Pod
metadata:
  creationTimestamp: null
  name: vraquier
  namespace: kube-system
spec:
  containers:
  - image: "sebt3/vraquier:latest"
    imagePullPolicy: Always
    name: vraquier
    volumeMounts:
    - mountPath: /etc/kubernetes/admin.conf
      name: kubeconfig
  volumes:
  - hostPath:
      path: /etc/kubernetes/admin.conf
    name: kubeconfig
status: {}
```

## Adding a node
Until the instance part does it automaticly...
TODO: https://kubernetes.io/fr/docs/tasks/administer-cluster/running-cloud-controller/
```
kubectl label nodes my-k8s-node vraquier.solidite.fr/external-ip=1.2.3.4
```


## Known issues

Calico/Canal doesn't play well with vraquier [by default](https://github.com/k3s-io/klipper-lb/issues/6#issuecomment-709691157).
