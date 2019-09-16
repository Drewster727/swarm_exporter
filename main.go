package main

import (
	"fmt"
	"log"
	"time"

	"swarm_exporter/exporter"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"golang.org/x/net/context"
)

func main() {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}

	options := types.ServiceListOptions{}
	serviceCh := make(chan exporter.Service, 10)

	go exporter.DockerServiceMetrics(serviceCh)
	go func() {
		for {
			services, err := cli.ServiceList(ctx, options)

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
				tasks, err := cli.TaskList(ctx, taskOptions)
				if err != nil {
					log.Println(err)
				}

				var tskMap = make(map[string]exporter.Task)
				for _, task := range tasks {
					tsk := exporter.Task{
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

				var tsks []exporter.Task
				for _, value := range tskMap {
					tsks = append(tsks, value)
				}

				serviceCh <- exporter.Service{
					Mode:     mode,
					Name:     service.Spec.Annotations.Name,
					Replicas: replicas,
					Tasks:    tsks,
				}
			}

			time.Sleep(1 * time.Second)
		}
	}()

	err = exporter.HandleHTTP(":9999")
	if err != nil {
		log.Println(err)
	}
}
