// Copyright 2021 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package metric

import (
	"net/http"
	"strings"

	"github.com/VictoriaMetrics/metrics"
)

// MetricOpts contains naming pieces of the exposed metric
type MetricOpts struct {
	Namespace string
	Subsystem string
	Name      string
}

// StartMetrics adds the metrics handler to a http.ServeMux
func StartMetrics(mux *http.ServeMux) {
	mux.HandleFunc("/metrics", func(rw http.ResponseWriter, r *http.Request) {
		metrics.WritePrometheus(rw, true)
	})
}

// Counter creates and returns a metrics.Counter
func Counter(opts MetricOpts, labels []string) *metrics.Counter {
	return metrics.GetOrCreateCounter(optsToString(opts) + labelsToString(labels))
}

// Gauge creates and returns a metrics.Gauge
func Gauge(opts MetricOpts, labels []string, f func() float64) *metrics.Gauge {
	return metrics.GetOrCreateGauge(optsToString(opts)+labelsToString(labels), f)
}

// Histogram creates and returns a metrics.Histogram
func Histogram(opts MetricOpts, labels []string) *metrics.Histogram {
	return metrics.GetOrCreateHistogram(optsToString(opts) + labelsToString(labels))
}

func optsToString(opts MetricOpts) string {
	if opts.Name == "" {
		return ""
	}
	switch {
	case opts.Namespace != "" && opts.Subsystem != "":
		return strings.Join([]string{opts.Namespace, opts.Subsystem, opts.Name}, "_")
	case opts.Namespace != "":
		return strings.Join([]string{opts.Namespace, opts.Name}, "_")
	case opts.Subsystem != "":
		return strings.Join([]string{opts.Subsystem, opts.Name}, "_")
	}
	return opts.Name
}

func labelsToString(labels []string) string {
	if len(labels) == 0 {
		return ""
	}
	s := "{"
	for _, label := range labels {
		s = s + label + ", "
	}
	return strings.TrimRight(s, ", ") + "}"
}
