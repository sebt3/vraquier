package provider

import (
	"context"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	cloudprovider "k8s.io/cloud-provider"
	"k8s.io/klog/v2"
)

var _ cloudprovider.LoadBalancer = &cloud{}

// GetLoadBalancer returns whether the specified load balancer exists, and if so, what its status is.
// Parameter 'clusterName' is the name of the cluster as presented to kube-controller-manager
func (c *cloud) GetLoadBalancer(ctx context.Context, clusterName string, service *v1.Service) (status *v1.LoadBalancerStatus, exists bool, err error) {
	klog.Infof("GetLoadBalancer for %s", service.Name)
	if _, err := c.getDaemonSet(ctx, service); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	status, err_status := c.getStatus(service)
	return status, true, err_status
}

// GetLoadBalancerName returns the name of the load balancer.
func (c *cloud) GetLoadBalancerName(ctx context.Context, clusterName string, service *v1.Service) string {
	return generateName(service)
}

// EnsureLoadBalancer creates a new load balancer 'name', or updates the existing one. Returns the status of the balancer
func (c *cloud) EnsureLoadBalancer(ctx context.Context, clusterName string, service *v1.Service, nodes []*v1.Node) (*v1.LoadBalancerStatus, error) {
	if err := c.deployDaemonSet(ctx, service); err != nil {
		return nil, err
	}
	return nil, cloudprovider.ImplementedElsewhere
}

// UpdateLoadBalancer updates hosts under the specified load balancer.
func (c *cloud) UpdateLoadBalancer(ctx context.Context, clusterName string, service *v1.Service, nodes []*v1.Node) error {
	if err := c.deployDaemonSet(ctx, service); err != nil {
		return err
	}
	return cloudprovider.ImplementedElsewhere
}

// EnsureLoadBalancerDeleted deletes the specified load balancer if it
// exists, returning nil if the load balancer specified either didn't exist or
// was successfully deleted.
func (c *cloud) EnsureLoadBalancerDeleted(ctx context.Context, clusterName string, service *v1.Service) error {
	return c.deleteDaemonSet(ctx, service)
}
