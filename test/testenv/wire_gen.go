// Code generated by Wire. DO NOT EDIT.

//go:generate wire
//+build !wireinject

package testenv

import (
	"github.com/google/wire"
	"github.com/redhat-marketplace/redhat-marketplace-operator/pkg/config"
	"github.com/redhat-marketplace/redhat-marketplace-operator/pkg/controller"
	"github.com/redhat-marketplace/redhat-marketplace-operator/pkg/managers"
	"github.com/redhat-marketplace/redhat-marketplace-operator/pkg/utils/reconcileutils"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// Injectors from wire.go:

func InitializeScheme(cfg *rest.Config) (*runtime.Scheme, error) {
	opsSrcSchemeDefinition := controller.ProvideOpsSrcScheme()
	monitoringSchemeDefinition := controller.ProvideMonitoringScheme()
	olmV1SchemeDefinition := controller.ProvideOLMV1Scheme()
	olmV1Alpha1SchemeDefinition := controller.ProvideOLMV1Alpha1Scheme()
	openshiftConfigV1SchemeDefinition := controller.ProvideOpenshiftConfigV1Scheme()
	localSchemes := controller.ProvideLocalSchemes(opsSrcSchemeDefinition, monitoringSchemeDefinition, olmV1SchemeDefinition, olmV1Alpha1SchemeDefinition, openshiftConfigV1SchemeDefinition)
	scheme, err := managers.ProvideScheme(cfg, localSchemes)
	if err != nil {
		return nil, err
	}
	return scheme, nil
}

func InitializeMainCtrl(cfg *rest.Config) (*managers.ControllerMain, error) {
	defaultCommandRunnerProvider := reconcileutils.ProvideDefaultCommandRunnerProvider()
	operatorConfig, err := config.ProvideConfig()
	if err != nil {
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	marketplaceController := controller.ProvideMarketplaceController(defaultCommandRunnerProvider, operatorConfig, clientset)
	meterbaseController := controller.ProvideMeterbaseController(defaultCommandRunnerProvider)
	meterDefinitionController := controller.ProvideMeterDefinitionController(defaultCommandRunnerProvider)
	razeeDeployController := controller.ProvideRazeeDeployController(operatorConfig)
	olmSubscriptionController := controller.ProvideOlmSubscriptionController()
	meterReportController := controller.ProvideMeterReportController(defaultCommandRunnerProvider, operatorConfig)
	olmClusterServiceVersionController := controller.ProvideOlmClusterServiceVersionController()
	remoteResourceS3Controller := controller.ProvideRemoteResourceS3Controller()
	nodeController := controller.ProvideNodeController()
	rhmSubscriptionController := controller.ProvideRhmSubscriptionController()
	clusterRegistrationController := controller.ProvideClusterRegistrationController()
	controllerList := controller.ProvideControllerList(marketplaceController, meterbaseController, meterDefinitionController, razeeDeployController, olmSubscriptionController, meterReportController, olmClusterServiceVersionController, remoteResourceS3Controller, nodeController, rhmSubscriptionController, clusterRegistrationController)
	opsSrcSchemeDefinition := controller.ProvideOpsSrcScheme()
	monitoringSchemeDefinition := controller.ProvideMonitoringScheme()
	olmV1SchemeDefinition := controller.ProvideOLMV1Scheme()
	olmV1Alpha1SchemeDefinition := controller.ProvideOLMV1Alpha1Scheme()
	openshiftConfigV1SchemeDefinition := controller.ProvideOpenshiftConfigV1Scheme()
	localSchemes := controller.ProvideLocalSchemes(opsSrcSchemeDefinition, monitoringSchemeDefinition, olmV1SchemeDefinition, olmV1Alpha1SchemeDefinition, openshiftConfigV1SchemeDefinition)
	scheme, err := managers.ProvideScheme(cfg, localSchemes)
	if err != nil {
		return nil, err
	}
	options, err := provideOptions(scheme)
	if err != nil {
		return nil, err
	}
	manager, err := managers.ProvideManager(cfg, scheme, localSchemes, options)
	if err != nil {
		return nil, err
	}
	controllerMain := makeMarketplaceController(controllerList, manager)
	return controllerMain, nil
}

// wire.go:

var testControllerSet = wire.NewSet(controller.ControllerSet, controller.ProvideControllerFlagSet, controller.SchemeDefinitions, managers.ProvideConfiglessManagerSet, config.ProvideConfig, reconcileutils.ProvideDefaultCommandRunnerProvider, provideOptions,
	makeMarketplaceController, wire.Bind(new(reconcileutils.ClientCommandRunnerProvider), new(*reconcileutils.DefaultCommandRunnerProvider)),
)

func provideOptions(kscheme *runtime.Scheme) (*manager.Options, error) {
	return &manager.Options{
		Namespace:          "",
		Scheme:             kscheme,
		MetricsBindAddress: "0",
	}, nil
}

func makeMarketplaceController(
	controllerList controller.ControllerList,
	mgr manager.Manager,
) *managers.ControllerMain {
	return &managers.ControllerMain{
		Name:        "redhat-marketplace-operator",
		FlagSets:    []*pflag.FlagSet{},
		Controllers: controllerList,
		Manager:     mgr,
	}
}
