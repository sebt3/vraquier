package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	servicehelper "k8s.io/cloud-provider/service/helpers"
	utilpointer "k8s.io/utils/pointer"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/klog/v2"
)

var (
	finalizerName          = "vraquier.solidite.fr/daemonset"
	svcNameLabel           = "vraquier.solidite.fr/svcname"
	svcNamespaceLabel      = "vraquier.solidite.fr/svcnamespace"
	daemonsetNodeLabel     = "vraquier.solidite.fr/enablelb"
	daemonsetNodePoolLabel = "vraquier.solidite.fr/lbpool"
	nodeSelectorLabel      = "vraquier.solidite.fr/nodeselector"
)
const (
	DefaultLBNS    = meta.NamespaceSystem
	DefaultLBImage = "rancher/klipper-lb:v0.4.9"
)

func generateName(svc *core.Service) string {
	return fmt.Sprintf("vraquier-%s-%s", svc.Name, svc.UID[:8])
}

func (c *cloud) getDaemonSet(ctx context.Context, svc *core.Service) (*apps.DaemonSet, error) {
	klog.Infof("getDaemonSet for %s (%s - %s)", svc.Name, c.LBNamespace, generateName(svc))
	return c.client.AppsV1().DaemonSets(c.LBNamespace).Get(ctx, generateName(svc), meta.GetOptions{})
}

func (c *cloud) getStatus(ctx context.Context, svc *core.Service) (*core.LoadBalancerStatus, error) {
	klog.Infof("getStatus for %s", svc.Name)
	// Get the status of the pods of our daemonset, find the hostIP and set it as ingress IP
	labelSelector := meta.LabelSelector{MatchLabels: map[string]string{
		svcNameLabel:      svc.Name,
		svcNamespaceLabel: svc.Namespace,
	}}
	pods, err := c.client.CoreV1().Pods(c.LBNamespace).List(ctx, meta.ListOptions{LabelSelector: labels.Set(labelSelector.MatchLabels).String()})
	if err != nil {
		return nil, err
	}

	loadbalancer := &core.LoadBalancerStatus{}
	found := false
	for _, p := range pods.Items {
		if p.Status.Phase == core.PodRunning {
			found = true
		}
		found_extern := false
		if node, err := c.client.CoreV1().Nodes().Get(ctx, p.Spec.NodeName, meta.GetOptions{}); err == nil {
			for _, v := range node.Status.Addresses {
				if v.Type == core.NodeExternalIP {
					loadbalancer.Ingress = append(loadbalancer.Ingress, core.LoadBalancerIngress{
						IP: v.Address,
					})
					found_extern = true
				}
			}
			if !found_extern {
				found_extern = true
				if address := node.Annotations[ExternalIPKey]; address != "" {
					for _, v := range strings.Split(address, ",") {
						loadbalancer.Ingress = append(loadbalancer.Ingress, core.LoadBalancerIngress{IP: v})
					}
				} else if address := node.Labels[ExternalIPKey]; address != "" {
					for _, v := range strings.Split(address, ",") {
						loadbalancer.Ingress = append(loadbalancer.Ingress, core.LoadBalancerIngress{IP: v})
					}
				} else if address := node.Annotations[FlannelIPKey]; address != "" {
					loadbalancer.Ingress = append(loadbalancer.Ingress, core.LoadBalancerIngress{
						IP: address,
					})
				} else if address := node.Labels[FlannelIPKey]; address != "" {
					loadbalancer.Ingress = append(loadbalancer.Ingress, core.LoadBalancerIngress{
						IP: address,
					})
				} else {
					found_extern = false
				}
			}
		}
		if !found_extern {
			loadbalancer.Ingress = append(loadbalancer.Ingress, core.LoadBalancerIngress{
				IP: p.Status.HostIP,
			})
		}
	}
	if !found {
		return nil, fmt.Errorf("no running pods found for %s", svc.Name)
	}
	return loadbalancer, nil
}

// deployDaemonSet ensures that there is a DaemonSet for the service.
func (c *cloud) deployDaemonSet(ctx context.Context, svc *core.Service) (*core.LoadBalancerStatus, error) {
	klog.Infof("deployDaemonSet for %s", svc.Name)
	ds, err := c.newDaemonSet(svc)
	if err != nil {
		return nil, err
	}

	if _, err := c.getDaemonSet(ctx, svc); err == nil {
		 _, err := c.client.AppsV1().DaemonSets(c.LBNamespace).Update(ctx, ds, meta.UpdateOptions{})
		 if err != nil {
			return nil, err
		}
	} else if apierrors.IsNotFound(err) {
		_, err := c.client.AppsV1().DaemonSets(c.LBNamespace).Create(ctx, ds, meta.CreateOptions{})
		if err != nil {
			return nil, err
		}
	} else {
		return nil, err
	}
/*	defer k.recorder.Eventf(svc, core.EventTypeNormal, "AppliedDaemonSet", "Applied LoadBalancer DaemonSet %s/%s", ds.Namespace, ds.Name)*/
	if svc.Spec.LoadBalancerIP != "" {
		return &svc.Status.LoadBalancer, nil
	} else {
		return c.getStatus(ctx, svc)
	}
}

// deleteDaemonSet ensures that there are no DaemonSets for the given service.
func (c *cloud) deleteDaemonSet(ctx context.Context, svc *core.Service) error {
	klog.Infof("deleteDaemonSet for %s", svc.Name)

	if err := c.client.AppsV1().DaemonSets(c.LBNamespace).Delete(ctx, generateName(svc), meta.DeleteOptions{}); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}
	//TODO:
	/*defer k.recorder.Eventf(svc, core.EventTypeNormal, "DeletedDaemonSet", "Deleted LoadBalancer DaemonSet %s/%s", k.LBNamespace, name)*/
	return nil
}


func (c *cloud) nodeHasDaemonSetLabel() (bool, error) {
	return false, nil
}

func (c *cloud) ensureServiceLBServiceAccount(ctx context.Context) error {
	sa := c.client.CoreV1().ServiceAccounts(c.LBNamespace)
	if _, err := sa.Get(ctx, "svclb", meta.GetOptions{}); err == nil || !apierrors.IsNotFound(err) {
		return err
	}
	_, err := sa.Create(ctx, &core.ServiceAccount{
		ObjectMeta: meta.ObjectMeta{
			Name:      "svclb",
			Namespace: c.LBNamespace,
		},
	}, meta.CreateOptions{})
	if apierrors.IsAlreadyExists(err) {
		return nil
	}
	return err
}

func (c *cloud) newDaemonSet(svc *core.Service) (*apps.DaemonSet, error) {
	name := generateName(svc)
	oneInt := intstr.FromInt(1)
	localTraffic := servicehelper.RequestsOnlyLocalTraffic(svc)
	sourceRanges, err := servicehelper.GetLoadBalancerSourceRanges(svc)
	if err != nil {
		return nil, err
	}

	var sysctls []core.Sysctl
	for _, ipFamily := range svc.Spec.IPFamilies {
		switch ipFamily {
		case core.IPv4Protocol:
			sysctls = append(sysctls, core.Sysctl{Name: "net.ipv4.ip_forward", Value: "1"})
		case core.IPv6Protocol:
			sysctls = append(sysctls, core.Sysctl{Name: "net.ipv6.conf.all.forwarding", Value: "1"})
		}
	}

	ds := &apps.DaemonSet{
		ObjectMeta: meta.ObjectMeta{
			Name:      name,
			Namespace: c.LBNamespace,
			Labels: labels.Set{
				nodeSelectorLabel: "false",
				svcNameLabel:      svc.Name,
				svcNamespaceLabel: svc.Namespace,
			},
		},
		TypeMeta: meta.TypeMeta{
			Kind:       "DaemonSet",
			APIVersion: "apps/v1",
		},
		Spec: apps.DaemonSetSpec{
			Selector: &meta.LabelSelector{
				MatchLabels: labels.Set{
					"app": name,
				},
			},
			Template: core.PodTemplateSpec{
				ObjectMeta: meta.ObjectMeta{
					Labels: labels.Set{
						"app":             name,
						svcNameLabel:      svc.Name,
						svcNamespaceLabel: svc.Namespace,
					},
				},
				Spec: core.PodSpec{
					ServiceAccountName:           "svclb",
					AutomountServiceAccountToken: utilpointer.Bool(false),
					SecurityContext: &core.PodSecurityContext{
						Sysctls: sysctls,
					},
					Tolerations: []core.Toleration{
						{
							Key:      "node-role.kubernetes.io/master",
							Operator: "Exists",
							Effect:   "NoSchedule",
						},
						{
							Key:      "node-role.kubernetes.io/control-plane",
							Operator: "Exists",
							Effect:   "NoSchedule",
						},
						{
							Key:      "CriticalAddonsOnly",
							Operator: "Exists",
						},
					},
				},
			},
			UpdateStrategy: apps.DaemonSetUpdateStrategy{
				Type: apps.RollingUpdateDaemonSetStrategyType,
				RollingUpdate: &apps.RollingUpdateDaemonSet{
					MaxUnavailable: &oneInt,
				},
			},
		},
	}

	for _, port := range svc.Spec.Ports {
		portName := fmt.Sprintf("lb-%s-%d", strings.ToLower(string(port.Protocol)), port.Port)
		container := core.Container{
			Name:            portName,
			Image:           c.LBImage,
			ImagePullPolicy: core.PullIfNotPresent,
			Ports: []core.ContainerPort{
				{
					Name:          portName,
					ContainerPort: port.Port,
					HostPort:      port.Port,
					Protocol:      port.Protocol,
				},
			},
			Env: []core.EnvVar{
				{
					Name:  "SRC_PORT",
					Value: strconv.Itoa(int(port.Port)),
				},
				{
					Name:  "SRC_RANGES",
					Value: strings.Join(sourceRanges.StringSlice(), " "),
				},
				{
					Name:  "DEST_PROTO",
					Value: string(port.Protocol),
				},
			},
			SecurityContext: &core.SecurityContext{
				Capabilities: &core.Capabilities{
					Add: []core.Capability{
						"NET_ADMIN",
					},
				},
			},
		}

		if localTraffic {
			container.Env = append(container.Env,
				core.EnvVar{
					Name:  "DEST_PORT",
					Value: strconv.Itoa(int(port.NodePort)),
				},
				core.EnvVar{
					Name: "DEST_IPS",
					ValueFrom: &core.EnvVarSource{
						FieldRef: &core.ObjectFieldSelector{
							FieldPath: "status.hostIPs",
						},
					},
				},
			)
		} else {
			container.Env = append(container.Env,
				core.EnvVar{
					Name:  "DEST_PORT",
					Value: strconv.Itoa(int(port.Port)),
				},
				core.EnvVar{
					Name:  "DEST_IPS",
					Value: strings.Join(svc.Spec.ClusterIPs, " "),
				},
			)
		}

		ds.Spec.Template.Spec.Containers = append(ds.Spec.Template.Spec.Containers, container)
	}

	// Add node selector only if label "svccontroller.k3s.cattle.io/enablelb" exists on the nodes
	enableNodeSelector, err := c.nodeHasDaemonSetLabel()
	if err != nil {
		return nil, err
	}
	if enableNodeSelector {
		ds.Spec.Template.Spec.NodeSelector = map[string]string{
			daemonsetNodeLabel: "true",
		}
		// Add node selector for "svccontroller.k3s.cattle.io/lbpool=<pool>" if service has lbpool label
		if svc.Labels[daemonsetNodePoolLabel] != "" {
			ds.Spec.Template.Spec.NodeSelector[daemonsetNodePoolLabel] = svc.Labels[daemonsetNodePoolLabel]
		}
		ds.Labels[nodeSelectorLabel] = "true"
	}

	return ds, nil
}
