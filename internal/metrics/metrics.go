// Copyright 2020 Ohio Supercomputer Center
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/version"
)

const (
	metricsNamespace = "k8_ldap_configmap"
)

var (
	metricBuildInfo = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metricsNamespace,
		Name:      "build_info",
		Help:      "Build information",
		ConstLabels: prometheus.Labels{
			"version":   version.Version,
			"revision":  version.Revision,
			"branch":    version.Branch,
			"builddate": version.BuildDate,
			"goversion": version.GoVersion,
		},
	})
	MetricError = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metricsNamespace,
		Name:      "error",
		Help:      "Indicates an error was encountered",
	})
	MetricErrorsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metricsNamespace,
		Name:      "errors_total",
		Help:      "Total number of errors",
	}, []string{"mapper"})
	MetricDuration = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metricsNamespace,
		Name:      "run_duration_seconds",
		Help:      "Last runtime duration in seconds",
	})
	MetricLastRun = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metricsNamespace,
		Name:      "last_run_timestamp_seconds",
		Help:      "Last timestamp of execution",
	})
	MetricConfigMapSize = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metricsNamespace,
		Name:      "size_bytes",
		Help:      "Size of ConfigMap in bytes",
	}, []string{"configmap"})
	MetricConfigMapKeys = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metricsNamespace,
		Name:      "keys_count",
		Help:      "Number of data keys in ConfigMap",
	}, []string{"configmap"})
)

func init() {
	metricBuildInfo.Set(1)
}

func MetricGathers(processMetrics bool) prometheus.Gatherers {
	registry := prometheus.NewRegistry()
	registry.MustRegister(metricBuildInfo)
	registry.MustRegister(MetricError)
	registry.MustRegister(MetricErrorsTotal)
	registry.MustRegister(MetricDuration)
	registry.MustRegister(MetricLastRun)
	registry.MustRegister(MetricConfigMapSize)
	registry.MustRegister(MetricConfigMapKeys)
	gatherers := prometheus.Gatherers{registry}
	if processMetrics {
		gatherers = append(gatherers, prometheus.DefaultGatherer)
	}
	return gatherers
}
