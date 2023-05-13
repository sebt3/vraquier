package provider

import (
	cloudprovider "k8s.io/cloud-provider"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

var _ cloudprovider.Interface = &cloud{}

type Config struct {
	LBImage     string `json:"lbImage"`
	LBNamespace string `json:"lbNamespace"`
}

type cloud struct {
	Config
	client *kubernetes.Clientset
	clusterName  string // name of the kind cluster
}
func New(clusterName string, client *kubernetes.Clientset) cloudprovider.Interface {
	klog.V(2).Infof("New for %s", clusterName)
	return &cloud{
		Config: Config{
			LBImage:     DefaultLBImage,
			LBNamespace: DefaultLBNS,
		},
		client: client,
		clusterName:  clusterName,
	}
}

// Initialize passes a Kubernetes clientBuilder interface to the cloud provider
func (c *cloud) Initialize(clientBuilder cloudprovider.ControllerClientBuilder, stopCh <-chan struct{}) {
	klog.V(2).Infof("Initialize")
	// noop
}

// Clusters returns the list of clusters.
func (c *cloud) Clusters() (cloudprovider.Clusters, bool) {
	return nil, false
}

// ProviderName returns the cloud provider ID.
func (c *cloud) ProviderName() string {
	return "vraquier"
}

func (c *cloud) LoadBalancer() (cloudprovider.LoadBalancer, bool) {
	return c, true
}

func (c *cloud) Instances() (cloudprovider.Instances, bool) {
	return nil, false
}

func (c *cloud) Zones() (cloudprovider.Zones, bool) {
	return nil, false
}

func (c *cloud) Routes() (cloudprovider.Routes, bool) {
	return nil, false
}

func (c *cloud) HasClusterID() bool {
	return len(c.clusterName) > 0
}

func (c *cloud) InstancesV2() (cloudprovider.InstancesV2, bool) {
	return c, true
}
