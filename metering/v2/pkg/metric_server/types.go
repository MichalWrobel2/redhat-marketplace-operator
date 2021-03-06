// Copyright 2020 IBM Corp.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package metric_server

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
	"k8s.io/kube-state-metrics/pkg/options"
)

var (
	DefaultNamespaces = options.NamespaceList{metav1.NamespaceAll}

	DefaultResources = map[string]struct{}{
		"pods":     struct{}{},
		"services": struct{}{},
	}

	DefaultEnabledResources = []string{"pods", "services"}
)

type promLogger struct{}

func (pl promLogger) Println(v ...interface{}) {
	klog.Error(v...)
}
