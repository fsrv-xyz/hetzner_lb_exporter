package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/alecthomas/kingpin/v2"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	labelLoadBalancerID     = "loadbalancer_id"
	labelLoadBalancerName   = "loadbalancer_name"
	labelEndpointListenPort = "endpoint_listen_port"
	labelTargetIdentifier   = "target_identifier"
	labelTrafficDirection   = "traffic_direction"
)

var (
	metricLoadBalancerTargetCount = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "hetzner_lb",
		Name:      "target_count",
		Help:      "Number of targets in a load balancer",
	}, []string{
		labelLoadBalancerID,
		labelLoadBalancerName,
	})

	metricLoadBalancerServiceCount = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "hetzner_lb",
		Name:      "service_count",
		Help:      "Number of services in a load balancer",
	}, []string{
		labelLoadBalancerID,
		labelLoadBalancerName,
	})

	metricLoadBalancerTargetHealthStatus = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "hetzner_lb",
		Name:      "target_health_status",
		Help:      "Health status of a target",
	}, []string{
		labelLoadBalancerID,
		labelEndpointListenPort,
		labelTargetIdentifier,
	})

	metricLoadBalancerTraffic = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "hetzner_lb",
		Name:      "traffic_bytes",
		Help:      "traffic of a load balancer",
	}, []string{
		labelLoadBalancerID,
		labelTrafficDirection,
	})
)

type Parameters struct {
	ApiKey           string
	WebListenAddress string
}

var parameters Parameters

func init() {
	app := kingpin.New(os.Args[0], "Hetzner Load Balancer Exporter")
	app.Flag("hetzner.api-key", "Hetzner API key").Envar("HETZNER_API_TOKEN").Required().StringVar(&parameters.ApiKey)
	app.Flag("web.listen-address", "Address to listen on for web interface and telemetry.").Default(":9115").StringVar(&parameters.WebListenAddress)

	kingpin.MustParse(app.Parse(os.Args[1:]))

	prometheus.MustRegister(
		metricLoadBalancerTargetCount,
		metricLoadBalancerServiceCount,
		metricLoadBalancerTargetHealthStatus,
		metricLoadBalancerTraffic,
	)
}

func refreshWorker(ctx context.Context, done chan any, client *hcloud.Client) {
	for {
		select {
		case <-ctx.Done():
			log.Println("refreshWorker: context done")
			done <- true
			return
		case <-time.After(time.Second * 2):
			loadbalancers, _, loadbalancerListError := client.LoadBalancer.List(context.Background(), hcloud.LoadBalancerListOpts{})
			if loadbalancerListError != nil {
				panic(loadbalancerListError)
			}

			for _, loadbalancer := range loadbalancers {
				metricLoadBalancerTargetCount.With(prometheus.Labels{labelLoadBalancerID: strconv.Itoa(int(loadbalancer.ID)), labelLoadBalancerName: loadbalancer.Name}).Set(float64(len(loadbalancer.Targets)))
				metricLoadBalancerServiceCount.With(prometheus.Labels{labelLoadBalancerID: strconv.Itoa(int(loadbalancer.ID)), labelLoadBalancerName: loadbalancer.Name}).Set(float64(len(loadbalancer.Services)))

				for _, target := range loadbalancer.Targets {
					for _, health := range target.HealthStatus {
						var status int
						switch health.Status {
						case hcloud.LoadBalancerTargetHealthStatusStatusHealthy:
							status = 1
						case hcloud.LoadBalancerTargetHealthStatusStatusUnhealthy:
							status = 2
						case hcloud.LoadBalancerTargetHealthStatusStatusUnknown:
							status = 3
						}
						metricLoadBalancerTargetHealthStatus.With(prometheus.Labels{labelLoadBalancerID: strconv.Itoa(int(loadbalancer.ID)), labelEndpointListenPort: strconv.Itoa(health.ListenPort), labelTargetIdentifier: target.IP.IP}).Set(float64(status))
					}
				}

				metricLoadBalancerTraffic.With(prometheus.Labels{labelLoadBalancerID: strconv.Itoa(int(loadbalancer.ID)), labelTrafficDirection: "in"}).Set(float64(loadbalancer.IngoingTraffic))
				metricLoadBalancerTraffic.With(prometheus.Labels{labelLoadBalancerID: strconv.Itoa(int(loadbalancer.ID)), labelTrafficDirection: "out"}).Set(float64(loadbalancer.OutgoingTraffic))
			}
		}
	}
}

func main() {
	// capture input signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	client := hcloud.NewClient(hcloud.WithToken(parameters.ApiKey))

	workerContext, workerCancel := context.WithCancel(context.Background())
	workerDone := make(chan any)
	go refreshWorker(workerContext, workerDone, client)

	server := &http.Server{
		Addr:    parameters.WebListenAddress,
		Handler: nil,
	}

	// set http server routes
	http.Handle("/", http.RedirectHandler("/metrics", http.StatusPermanentRedirect))
	http.Handle("/metrics", promhttp.Handler())

	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Println(err)
		}
	}()

	// wait for incoming interrupts
	<-sigChan

	workerCancel()

	// shutdown server
	if err := server.Shutdown(context.Background()); err != nil {
		log.Fatal(err)
	}
	<-workerDone
}
