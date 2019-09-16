package exporter

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var registry *prometheus.Registry

func init() {
	registry = prometheus.NewRegistry()
	registry.MustRegister(dockerServiceReplicas)
	registry.MustRegister(dockerServiceTasks)
}

var dockerServiceReplicas = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "docker_service_replicas",
		Help: "Current state of Docker service.",
	},
	[]string{"mode", "name"},
)

var dockerServiceTasks = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "docker_service_tasks_state",
		Help: "Current state of Docker service tasks.",
	},
	[]string{"mode", "name", "desiredstate", "state"},
)

func DockerServiceMetrics(ch <-chan Service) {
	go func() {
		for {
			if ch == nil {
				break
			}

			update := <-ch
			name := update.Name
			dockerServiceReplicas.WithLabelValues(update.Mode, name).Set(float64(update.Replicas))
			for _, task := range update.Tasks {
				dockerServiceTasks.WithLabelValues(update.Mode, name, task.DesiredState, task.State).Set(float64(task.Count))
			}
		}
	}()
}

func HandleHTTP(addr string) error {
	handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	http.Handle("/metrics", handler)
	return http.ListenAndServe(addr, nil)
}
