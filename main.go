package main

import (
	"context"
	"crypto/tls"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	kingpin "github.com/alecthomas/kingpin/v2"
	"github.com/budacom/nchan-prometheus-exporter/client"
	"github.com/budacom/nchan-prometheus-exporter/collector"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors/version"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/promslog"
	promflag "github.com/prometheus/common/promslog/flag"
	commonversion "github.com/prometheus/common/version"
	"github.com/prometheus/exporter-toolkit/web"
	"github.com/prometheus/exporter-toolkit/web/kingpinflag"
)

const exporterName = "nchan_exporter"

func main() {
	webConfig := kingpinflag.AddFlags(kingpin.CommandLine, ":9113")
	metricsPath := kingpin.Flag("web.telemetry-path", "Path under which to expose metrics.").Default("/metrics").String()
	scrapeURI := kingpin.Flag("scrape-uri", "URI of the nchan_stub_status page.").Default("http://127.0.0.1:81/nchan_status").String()
	sslVerify := kingpin.Flag("ssl-verify", "Verify the server certificate when scrape-uri is https.").Default("true").Envar("SSL_VERIFY").Bool()
	nginx := kingpin.Flag("nginx", "Ignored. Scrape nginx stub_status with the official nginx-prometheus-exporter.").Default("false").Bool()
	nginxScrapeURI := kingpin.Flag("nginx.scrape-uri", "Ignored. Scrape nginx stub_status with the official nginx-prometheus-exporter.").Default("").String()

	logConfig := &promslog.Config{}
	promflag.AddFlags(kingpin.CommandLine, logConfig)
	kingpin.Version(commonversion.Print(exporterName))
	kingpin.HelpFlag.Short('h')
	addMissingEnvironmentFlags(kingpin.CommandLine)

	// k8s passes the 0.2.0 single-dash flags.
	for i, arg := range os.Args {
		if strings.HasPrefix(arg, "-") && !strings.HasPrefix(arg, "--") && len(arg) > 2 {
			os.Args[i] = "-" + arg
		}
	}

	kingpin.Parse()
	logger := promslog.New(logConfig)

	if *nginx || *nginxScrapeURI != "" {
		logger.Warn("ignoring nginx flags; run nginx/nginx-prometheus-exporter as a second sidecar")
	}
	logger.Info("starting nchan exporter", "version", commonversion.Info(), "uri", *scrapeURI)

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: !*sslVerify,
	}
	httpClient := &http.Client{Timeout: 5 * time.Second, Transport: transport}

	prometheus.MustRegister(version.NewCollector(exporterName))
	prometheus.MustRegister(collector.New(client.New(httpClient, *scrapeURI), logger))

	http.Handle(*metricsPath, promhttp.Handler())
	if *metricsPath != "/" && *metricsPath != "" {
		landing, err := web.NewLandingPage(web.LandingConfig{
			Name:        "Nchan Prometheus Exporter",
			Description: "Prometheus exporter for Nchan",
			Version:     commonversion.Info(),
			Links: []web.LandingLinks{{
				Address: *metricsPath,
				Text:    "Metrics",
			}},
		})
		if err != nil {
			logger.Error("failed to create landing page", "error", err)
			os.Exit(1)
		}
		http.Handle("/", landing)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	srv := &http.Server{ReadHeaderTimeout: 5 * time.Second}
	go func() {
		if err := web.ListenAndServe(srv, webConfig, logger); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("HTTP server failed", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	logger.Info("shutting down")
	srvCtx, srvCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer srvCancel()
	_ = srv.Shutdown(srvCtx)
}

// addMissingEnvironmentFlags sets an env var on web.* flags that do not have one.
// web.listen-address becomes LISTEN_ADDRESS, matching the 0.2.0 exporter.
func addMissingEnvironmentFlags(ka *kingpin.Application) {
	for _, f := range ka.Model().Flags {
		if !strings.HasPrefix(f.Name, "web.") || f.Envar != "" {
			continue
		}
		if flag := ka.GetFlag(f.Name); flag != nil {
			flag.Envar(convertFlagToEnvar(strings.TrimPrefix(f.Name, "web.")))
		}
	}
}

func convertFlagToEnvar(name string) string {
	env := strings.ToUpper(name)
	for _, s := range []string{"-", "."} {
		env = strings.ReplaceAll(env, s, "_")
	}
	return env
}
