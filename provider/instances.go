package provider

import (
	"context"
	"errors"
	"strings"
	"fmt"

	v1 "k8s.io/api/core/v1"
	cloudprovider "k8s.io/cloud-provider"
	"k8s.io/klog/v2"
)

var (
	InternalIPKey = "vraquier.solidite.fr/internal-ip"
	ExternalIPKey = "vraquier.solidite.fr/external-ip"
	FlannelIPKey  = "flannel.alpha.coreos.com/public-ip"
	HostnameKey   = "vraquier.solidite.fr/hostname"
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
	if (node.Annotations[ExternalIPKey] == "") && (node.Labels[ExternalIPKey] == "") && (node.Annotations[FlannelIPKey] == "") && (node.Labels[FlannelIPKey] == "") {
		return nil, errors.New("address annotations not yet set")
	}
	currentaddresses := node.Status.Addresses
	addresses := []v1.NodeAddress{}

	// check internal address
	if address := node.Annotations[InternalIPKey]; address != "" {
		for _, v := range strings.Split(address, ",") {
			addresses = append(addresses, v1.NodeAddress{Type: v1.NodeInternalIP, Address: v})
		}
	} else if address = node.Labels[InternalIPKey]; address != "" {
		for _, v := range strings.Split(address, ",") {
			addresses = append(addresses, v1.NodeAddress{Type: v1.NodeInternalIP, Address: v})
		}
	} else {
		klog.Infof("Couldn't find node internal ip annotation or label on node %s, duplicating existing data", node.Name)
		for _, v := range currentaddresses {
			if v.Type == v1.NodeInternalIP {
				addresses = append(addresses, v1.NodeAddress{Type: v1.NodeInternalIP, Address: v.Address})
				break
			}
		}
	}

	// check external address
	if address := node.Annotations[ExternalIPKey]; address != "" {
		for _, v := range strings.Split(address, ",") {
			addresses = append(addresses, v1.NodeAddress{Type: v1.NodeExternalIP, Address: v})
		}
	} else if address = node.Labels[ExternalIPKey]; address != "" {
		for _, v := range strings.Split(address, ",") {
			addresses = append(addresses, v1.NodeAddress{Type: v1.NodeExternalIP, Address: v})
		}
	} else if address = node.Annotations[FlannelIPKey]; address != "" {
		addresses = append(addresses, v1.NodeAddress{Type: v1.NodeExternalIP, Address: address})
	} else if address = node.Labels[FlannelIPKey]; address != "" {
		addresses = append(addresses, v1.NodeAddress{Type: v1.NodeExternalIP, Address: address})
	} else {
		klog.Infof("Couldn't find node external ip annotation or label on node %s, duplicating existing data", node.Name)
		for _, v := range currentaddresses {
			if v.Type == v1.NodeExternalIP {
				addresses = append(addresses, v1.NodeAddress{Type: v1.NodeExternalIP, Address: v.Address})
				break
			}
		}
	}

	// check hostname
	if address := node.Annotations[HostnameKey]; address != "" {
		addresses = append(addresses, v1.NodeAddress{Type: v1.NodeHostName, Address: address})
	} else if address = node.Labels[HostnameKey]; address != "" {
		addresses = append(addresses, v1.NodeAddress{Type: v1.NodeHostName, Address: address})
	} else {
		klog.Infof("Couldn't find node hostname annotation or label on node %s", node.Name)
		for _, v := range currentaddresses {
			if v.Type == v1.NodeHostName {
				addresses = append(addresses, v1.NodeAddress{Type: v1.NodeHostName, Address: v.Address})
				break
			}
		}
	}

	return &cloudprovider.InstanceMetadata{
		ProviderID:    fmt.Sprintf("%s://%s", c.ProviderName(), node.Name),
		InstanceType:  c.ProviderName(),
		NodeAddresses: addresses,
		Zone:          "",
		Region:        "",
	}, nil
}
