// Copyright © 2019 Banzai Cloud
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package fluentd

import (
	"bytes"
	"fmt"
	"html/template"

	"emperror.dev/errors"
	"github.com/banzaicloud/operator-tools/pkg/reconciler"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type fluentdConfig struct {
	LogLevel string
	Monitor  struct {
		Enabled bool
		Port    int32
		Path    string
	}
	IgnoreSameLogInterval     string
	IgnoreRepeatedLogInterval string
	Workers                   int32
	RootDir                   string
}

func generateConfig(input fluentdConfig) (string, error) {
	output := new(bytes.Buffer)
	tmpl, err := template.New("test").Parse(fluentdInputTemplate)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse template")
	}
	err = tmpl.Execute(output, input)
	if err != nil {
		return "", errors.Wrap(err, "failed to execute template")
	}
	return output.String(), nil
}

func (r *Reconciler) secretConfig() (runtime.Object, reconciler.DesiredState, error) {
	input := fluentdConfig{
		IgnoreSameLogInterval:     r.Logging.Spec.FluentdSpec.IgnoreSameLogInterval,
		IgnoreRepeatedLogInterval: r.Logging.Spec.FluentdSpec.IgnoreRepeatedLogInterval,
		RootDir:                   r.Logging.Spec.FluentdSpec.RootDir,
	}

	if r.Logging.Spec.FluentdSpec.Metrics != nil {
		input.Monitor.Enabled = true
		input.Monitor.Port = r.Logging.Spec.FluentdSpec.Metrics.Port
		input.Monitor.Path = r.Logging.Spec.FluentdSpec.Metrics.Path
	}

	input.LogLevel = r.Logging.Spec.FluentdSpec.LogLevel
	if input.LogLevel == "" {
		input.LogLevel = "info"
	}

	input.Workers = r.Logging.Spec.FluentdSpec.Workers
	if input.Workers <= 0 {
		input.Workers = 1
	}

	inputConfig, err := generateConfig(input)
	if err != nil {
		return nil, reconciler.StatePresent, err
	}

	configs := &corev1.Secret{
		ObjectMeta: r.FluentdObjectMeta(SecretConfigName, ComponentFluentd),
		Data: map[string][]byte{
			"fluent.conf":  []byte(fluentdDefaultTemplate),
			"input.conf":   []byte(inputConfig),
			"devnull.conf": []byte(fluentdOutputTemplate),
		},
	}

	configs.Data["fluentlog.conf"] = []byte(fmt.Sprintf(fluentLog, r.Logging.Spec.FluentdSpec.FluentLogDestination))

	return configs, reconciler.StatePresent, nil
}
