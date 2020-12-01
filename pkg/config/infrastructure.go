package config

import (
	"context"

	openshiftconfigv1 "github.com/openshift/api/config/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type KubernetesInfra struct {
	version,
	platform string
}

type OpenshiftInfra struct {
	version string
}

// Infrastructure stores Kubernetes/Openshift clients
type Infrastructure struct {
	Openshift  OpenshiftInfra
	Kubernetes KubernetesInfra
}

func openshiftInfrastructure(c client.Client) (OpenshiftInfra, error) {
	clusterVersion := &openshiftconfigv1.ClusterVersion{}
	versionNamespacedName := types.NamespacedName{
		Name: "version",
	}
	err := c.Get(context.TODO(), versionNamespacedName, clusterVersion)
	if err != nil {
		return OpenshiftInfra{}, err
	}
	return OpenshiftInfra{
		version: clusterVersion.Status.Desired.Version,
	}, nil
}

func kubernetesInfrastructure(discoveryClient *discovery.DiscoveryClient) (kInf KubernetesInfra, err error) {
	serverVersion, err := discoveryClient.ServerVersion()
	if err == nil {
		kInf = KubernetesInfra{
			version:  serverVersion.GitVersion,
			platform: serverVersion.Platform,
		}
	}
	return
}

func LoadInfrastructure(c client.Client, dc *discovery.DiscoveryClient) (*Infrastructure, error) {
	openshift, err := openshiftInfrastructure(c)
	if err != nil {
		// check if api exists on the cluster
		if !errors.IsNotFound(err) && !meta.IsNoMatchError(err) {
			log.Error(err, "openshift api not found")
		} else {
			return nil, err
		}
	}
	kuberentes, err := kubernetesInfrastructure(dc)
	if err != nil {
		return nil, err
	}

	return &Infrastructure{
		Openshift:  openshift,
		Kubernetes: kuberentes,
	}, nil
}

func (k KubernetesInfra) Version() string {
	return k.version
}

func (k KubernetesInfra) Platform() string {
	return k.platform
}

func (o OpenshiftInfra) Version() string {
	return o.version
}
