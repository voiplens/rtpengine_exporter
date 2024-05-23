// MIT License
// Copyright 2023 Angarium Ltd
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.

package main

import (
	"net/http"
	"os"

	"github.com/alecthomas/kingpin/v2"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/promlog"
	"github.com/prometheus/common/promlog/flag"
	"github.com/prometheus/common/version"
	"github.com/prometheus/exporter-toolkit/web"
	webflag "github.com/prometheus/exporter-toolkit/web/kingpinflag"
	"github.com/voiplens/rtpengine_exporter/collector"
)

func init() {
	prometheus.MustRegister(version.NewCollector("rtpengine_exporter"))
}

func main() {
	var (
		metricsPath = kingpin.Flag(
			"web.telemetry-path",
			"Path under which to expose metrics.",
		).Default("/metrics").String()
		scrapeURI = kingpin.Flag(
			"rtpengine.ng-uri",
			`RTPEngine NG URI. E.g. "udp://localhost:22223"`,
		).Short('u').Default("udp://localhost:22223").String()
		timeout = kingpin.Flag(
			"rtpengine.timeout",
			"Timeout for trying to get data from rtpengine.",
		).Short('t').Default("5s").Duration()
		toolkitFlags = webflag.AddFlags(kingpin.CommandLine, ":9282")
	)
	promlogConfig := &promlog.Config{}
	flag.AddFlags(kingpin.CommandLine, promlogConfig)
	kingpin.Version(version.Print("rtpengine_exporter"))
	kingpin.Parse()
	logger := promlog.New(promlogConfig)

	level.Info(logger).Log("msg", "Starting rtpengine_exporter", "version", version.Info())
	level.Info(logger).Log("msg", "Build context", "build_context", version.BuildContext())

	c, err := collector.New(*scrapeURI, *timeout, logger)

	if err != nil {
		panic(err)
	}

	prometheus.MustRegister(c)

	http.Handle(*metricsPath, promhttp.Handler())
	if *metricsPath != "/" && *metricsPath != "" {
		landingConfig := web.LandingConfig{
			Name:        "RTPEngine Exporter",
			Description: "Prometheus Exporter for RTPEngine servers",
			Version:     version.Info(),
			Links: []web.LandingLinks{
				{
					Address: *metricsPath,
					Text:    "Metrics",
				},
			},
		}
		landingPage, err := web.NewLandingPage(landingConfig)
		if err != nil {
			level.Error(logger).Log("err", err)
			os.Exit(1)
		}
		http.Handle("/", landingPage)
	}
	server := &http.Server{}
	if err := web.ListenAndServe(server, toolkitFlags, logger); err != nil {
		level.Info(logger).Log("err", err)
		os.Exit(1)
	}
}
