package config

import (
	"context"
	"encoding/json"

	openshiftconfigv1 "github.com/openshift/api/config/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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
	clusterVersion := openshiftconfigv1.ClusterVersionStatus{}
	clusterVersionObj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "config.openshift.io/v1",
			"kind":       "ClusterVersion",
			"metadata": map[string]interface{}{
				"name": "version",
			},
			"spec": "console",
		},
	}
	versionNamespacedName := client.ObjectKey{
		Name: "version",
	}

	err := c.Get(context.TODO(), versionNamespacedName, clusterVersionObj)
	if err != nil {
		return OpenshiftInfra{}, err
	}
	marshaledStatus, err := json.Marshal(clusterVersionObj.Object["status"])
	if err != nil {
		return OpenshiftInfra{}, err
	}
	if err := json.Unmarshal(marshaledStatus, &clusterVersion); err != nil {
		return OpenshiftInfra{}, err
	}
	return OpenshiftInfra{
		version: clusterVersion.Desired.Version,
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
			log.Error(err, "unable to get Openshift version")
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
