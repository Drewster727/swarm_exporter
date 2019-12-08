package main

import (
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	addr            string
	pollIntervalSec int
)

func main() {
	loadEnv()

	prometheus.MustRegister(newServicesCollector())

	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(addr, nil))
}

func loadEnv() {
	addr = getEnvOr("SWARM_PROMETHEUS_LISTEN_ADDR", ":9999")
	pollInterval := getEnvOr("SWARM_PROMETHEUS_POLL_INTERVAL_SEC", "3")
	pis, err := strconv.Atoi(pollInterval)
	if err != nil {
		log.Fatal()
	}
	pollIntervalSec = pis
}

func getEnvOr(key string, defaultValue string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}

	return defaultValue
}
