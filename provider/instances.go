package provider

import (
	"context"
	"errors"
	"fmt"

	v1 "k8s.io/api/core/v1"
	cloudprovider "k8s.io/cloud-provider"
	"k8s.io/klog/v2"
)

var _ cloudprovider.InstancesV2 = &cloud{}

var errNodeNotFound = errors.New("node not found")

// InstanceExists returns true if the instance for the given node exists according to the cloud provider.
func (c *cloud) InstanceExists(ctx context.Context, node *v1.Node) (bool, error) {
	return true, nil
}

func (c *cloud) InstanceShutdown(ctx context.Context, node *v1.Node) (bool, error) {
	return false, nil
}

// InstanceMetadata returns the instance's metadata. The values returned in InstanceMetadata are
// translated into specific fields and labels in the Node object on registration.
func (c *cloud) InstanceMetadata(ctx context.Context, node *v1.Node) (*cloudprovider.InstanceMetadata, error) {
	klog.V(2).Infof("Check instance metadata for %s", node.Name)
	return &cloudprovider.InstanceMetadata{
		ProviderID:    fmt.Sprintf("%s://%s", c.ProviderName(), node.Name),
		InstanceType:  c.ProviderName(),
	}, nil
}
