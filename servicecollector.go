package main

import (
	"fmt"
	"log"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/net/context"
)

var dockerClient *client.Client

type Service struct {
	Mode     string
	Replicas uint64
	Name     string
	Tasks    []Task
}

type Task struct {
	DesiredState string
	State        string
	Count        int32
}

type servicesCollector struct {
	serviceReplicas   *prometheus.Desc
	serviceTasksState *prometheus.Desc
}

func newServicesCollector() *servicesCollector {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	dockerClient = cli

	return &servicesCollector{
		serviceReplicas: prometheus.NewDesc("docker_service_replicas",
			"Current state of Docker service.",
			[]string{"mode", "name"}, nil,
		),
		serviceTasksState: prometheus.NewDesc("docker_service_tasks_state",
			"Current state of Docker service tasks.",
			[]string{"mode", "name", "desiredstate", "state"}, nil,
		),
	}
}

func (c *servicesCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.serviceReplicas
	ch <- c.serviceTasksState
}

func (c *servicesCollector) Collect(ch chan<- prometheus.Metric) {
	data := fetchServices()

	for _, svc := range data {
		ch <- prometheus.MustNewConstMetric(c.serviceReplicas, prometheus.CounterValue, float64(svc.Replicas), svc.Mode, svc.Name)
		for _, task := range svc.Tasks {
			ch <- prometheus.MustNewConstMetric(c.serviceTasksState, prometheus.CounterValue, float64(task.Count), svc.Mode, svc.Name, task.DesiredState, task.State)
		}
	}

	return
}

func fetchServices() []Service {
	var svcs []Service

	options := types.ServiceListOptions{}
	ctx := context.Background()

	services, err := dockerClient.ServiceList(ctx, options)

	if err != nil {
		log.Println(err)
	}

	for _, service := range services {
		var mode string
		var replicas uint64
		if service.Spec.Mode.Global != nil {
			mode = "global"
			replicas = 0
		}
		if service.Spec.Mode.Replicated != nil {
			mode = "replicated"
			replicas = *service.Spec.Mode.Replicated.Replicas
		}

		taskOptions := types.TaskListOptions{
			Filters: filters.NewArgs(
				filters.Arg("service", service.ID),
			),
		}
		tasks, err := dockerClient.TaskList(ctx, taskOptions)
		if err != nil {
			log.Println(err)
		}

		var tskMap = make(map[string]Task)
		for _, task := range tasks {
			tsk := Task{
				DesiredState: string(task.DesiredState),
				State:        string(task.Status.State),
				Count:        1,
			}
			key := fmt.Sprintf("%v_%s", string(tsk.DesiredState), string(tsk.State))
			if _, ok := tskMap[key]; ok {
				t := tskMap[key]
				t.Count = t.Count + 1
				tskMap[key] = t
			} else {
				tskMap[key] = tsk
			}
		}

		var tsks []Task
		for _, value := range tskMap {
			tsks = append(tsks, value)
		}

		svc := Service{
			Mode:     mode,
			Name:     service.Spec.Annotations.Name,
			Replicas: replicas,
			Tasks:    tsks,
		}
		svcs = append(svcs, svc)
	}

	return svcs
}
