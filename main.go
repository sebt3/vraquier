package main

import (
	"os"

	"k8s.io/apimachinery/pkg/util/wait"
	cloudprovider "k8s.io/cloud-provider"
	"k8s.io/cloud-provider/app"
	"k8s.io/cloud-provider/app/config"
	"k8s.io/cloud-provider/options"
	"k8s.io/component-base/cli"
	cliflag "k8s.io/component-base/cli/flag"
	_ "k8s.io/component-base/logs/json/register"          // register optional JSON log format
	_ "k8s.io/component-base/metrics/prometheus/clientgo" // load all the prometheus client-go plugins
	_ "k8s.io/component-base/metrics/prometheus/version"  // for version metric registration
	"k8s.io/klog/v2"
	"github.com/sebt3/vraquier/provider"
)

func main() {
	ccmOptions, err := options.NewCloudControllerManagerOptions()
	if err != nil {
		klog.Fatalf("unable to initialize command options: %v", err)
	}

	fss := cliflag.NamedFlagSets{}
	command := app.NewCloudControllerManagerCommand(ccmOptions, cloudInitializer, controllerInitializers(), fss, wait.NeverStop)
	code := cli.Run(command)
	os.Exit(code)
}

// If custom ClientNames are used, as below, then the controller will not use
// the API server bootstrapped RBAC, and instead will require it to be installed
// separately.
func controllerInitializers() map[string]app.ControllerInitFuncConstructor {
	controllerInitializers := app.DefaultInitFuncConstructors
	if constructor, ok := controllerInitializers["service"]; ok {
		constructor.InitContext.ClientName = "vraquier"
		controllerInitializers["service"] = constructor
	}
	return controllerInitializers
}

func cloudInitializer(config *config.CompletedConfig) cloudprovider.Interface {
	return provider.New("vraquier", config.Client)
}