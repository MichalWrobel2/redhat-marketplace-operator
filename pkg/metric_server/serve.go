package metric_server

import (
	"compress/gzip"
	"context"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redhat-marketplace/redhat-marketplace-operator/internal/metrics"
	rhmclient "github.com/redhat-marketplace/redhat-marketplace-operator/pkg/client"
	"github.com/redhat-marketplace/redhat-marketplace-operator/pkg/managers"
	"github.com/redhat-marketplace/redhat-marketplace-operator/pkg/utils/logger"
	"github.com/redhat-marketplace/redhat-marketplace-operator/pkg/utils/reconcileutils"
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kube-state-metrics/pkg/options"
	"k8s.io/kube-state-metrics/pkg/util/proc"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	metricsPath = "/metrics"
	healthzPath = "/healthz"
)

var log = logger.NewLogger("meteric_generator")
var reg = prometheus.NewRegistry()

type Service struct {
	k8sclient       client.Client
	k8sRestClient   clientset.Interface
	opts            *options.Options
	cache           cache.Cache
	metricsRegistry *prometheus.Registry
	cc              reconcileutils.ClientCommandRunner
}

func (s *Service) Serve(done <-chan struct{}) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		err := s.cache.Start(ctx.Done())
		log.Error(err, "error starting cache")
	}()

	rhmclient.AddMeterDefIndex(s.cache)

	opts := s.opts
	storeBuilder := metrics.NewBuilder()
	storeBuilder.WithNamespaces(options.DefaultNamespaces)

	proc.StartReaper()

	storeBuilder.WithContext(ctx)
	storeBuilder.WithKubeClient(s.k8sRestClient)
	storeBuilder.WithClientCommand(s.cc)
	storeBuilder.WithSharding(opts.Shard, opts.TotalShards)

	s.metricsRegistry.MustRegister(
		prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}),
		prometheus.NewGoCollector(),
	)
	go telemetryServer(s.metricsRegistry, s.opts.TelemetryHost, s.opts.TelemetryPort)

	serveMetrics(ctx, storeBuilder, s.opts, s.opts.Host, opts.Port, s.opts.EnableGZIPEncoding)
	return nil
}

func getClientOptions() managers.ClientOptions {
	return managers.ClientOptions{
		Namespace:    "",
		DryRunClient: false,
	}
}

func provideRegistry() *prometheus.Registry {
	return prometheus.NewRegistry()
}

func telemetryServer(registry prometheus.Gatherer, host string, port int) {
	// Address to listen on for web interface and telemetry
	listenAddress := net.JoinHostPort(host, strconv.Itoa(port))

	log.Info("Starting kube-state-metrics self metrics server", "listenAddress", listenAddress)

	mux := http.NewServeMux()

	// Add metricsPath
	mux.Handle(metricsPath, promhttp.HandlerFor(registry, promhttp.HandlerOpts{ErrorLog: promLogger{}}))
	// Add index
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
             <head><title>RHM-Metering-Metrics Metrics Server</title></head>
             <body>
             <h1>RHM-Metering-Metrics Metrics</h1>
			 <ul>
             <li><a href='` + metricsPath + `'>metrics</a></li>
			 </ul>
             </body>
             </html>`))
	})

	err := http.ListenAndServe(listenAddress, mux)
	if err != nil {
		log.Error(err, "failing to listen and serve")
		panic(err)
	}
}

func serveMetrics(ctx context.Context, storeBuilder *metrics.Builder, opts *options.Options, host string, port int, enableGZIPEncoding bool) {
	// Address to listen on for web interface and telemetry
	listenAddress := net.JoinHostPort(host, strconv.Itoa(port))

	log.Info("Starting metrics server", "listenAddress", listenAddress)

	mux := http.NewServeMux()
	stores := storeBuilder.Build()

	log.Info("built stores")

	m := &metricHandler{stores, enableGZIPEncoding}
	mux.Handle(metricsPath, m)

	// Add healthzPath
	mux.HandleFunc(healthzPath, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(http.StatusText(http.StatusOK)))
	})
	// Add index
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
             <head><title>RHM-Metering-Metrics Server</title></head>
             <body>
             <h1>RHM-Metering-Metrics Metrics</h1>
			 <ul>
             <li><a href='` + metricsPath + `'>metrics</a></li>
             <li><a href='` + healthzPath + `'>healthz</a></li>
			 </ul>
             </body>
             </html>`))
	})
	err := http.ListenAndServe(listenAddress, mux)
	if err != nil {
		log.Error(err, "failing to listen and serve")
		panic(err)
	}
}

type metricHandler struct {
	stores             []*metrics.MetricsStore
	enableGZIPEncoding bool
}

func (m *metricHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	resHeader := w.Header()
	var writer io.Writer = w

	// Set the exposition format version of Prometheus.
	// https://prometheus.io/docs/instrumenting/exposition_formats/#text-based-format
	resHeader.Set("Content-Type", `text/plain; version=`+"0.0.4")

	if m.enableGZIPEncoding {
		// Gzip response if requested. Taken from
		// github.com/prometheus/client_golang/prometheus/promhttp.decorateWriter.
		reqHeader := r.Header.Get("Accept-Encoding")
		parts := strings.Split(reqHeader, ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "gzip" || strings.HasPrefix(part, "gzip;") {
				writer = gzip.NewWriter(writer)
				resHeader.Set("Content-Encoding", "gzip")
			}
		}
	}

	for _, c := range m.stores {
		c.WriteAll(w)
	}

	// In case we gzipped the response, we have to close the writer.
	if closer, ok := writer.(io.Closer); ok {
		closer.Close()
	}
}