# Vraquier

A poor-man "load-balancer" controller for kubernetes clusters without a cloud-controller.
It replicate the k3s behaviour by starting `klipper-lb` DaemonSet.

## Requierements

kubelet should be started with `--allowed-unsafe-sysctls 'net.ipv6.conf.all.forwarding,net.ipv4.ip_forward'`

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


## Known issues

Calico/Canal doesn't play well with vraquier [by default](https://github.com/k3s-io/klipper-lb/issues/6#issuecomment-709691157).
