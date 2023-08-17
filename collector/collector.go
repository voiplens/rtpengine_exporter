// MIT License
// Copyright 2023 Angarium Ltd
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.

package collector

import (
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/angarium-cloud/go-rtpengine/ng"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
)

// Collector implements prometheus.Collector (see below).
// it also contains the config of the exporter.
type Collector struct {
	URI     string
	Timeout time.Duration

	mutex sync.Mutex

	logger        log.Logger
	up            prometheus.Gauge
	failedScrapes prometheus.Counter
	totalScrapes  prometheus.Counter
}

const (
	namespace = "rtpengine"
)

var (
	// Current Metrics
	sessions   = prometheus.NewDesc("rtpengine_sessions", "Sessions", []string{"type"}, nil)
	errorRate  = prometheus.NewDesc("rtpengine_error_rate", "ErrorRate", []string{"type"}, nil)
	packetRate = prometheus.NewDesc("rtpengine_packet_rate", "PacketRate", []string{"type"}, nil)
	byteRate   = prometheus.NewDesc("rtpengine_byte_rate", "Byte Rate", []string{"type"}, nil)
	media      = prometheus.NewDesc("rtpengine_media", "Media", []string{"type"}, nil)

	// Total Metrics
	uptimeTotal              = prometheus.NewDesc("rtpengine_uptime_total", "Uptime", []string{}, nil)
	sessionsTotal            = prometheus.NewDesc("rtpengine_sessions_total", "Sessions", []string{"type"}, nil)
	timeoutSessionsTotal     = prometheus.NewDesc("rtpengine_timeout_sessions_total", "Timeout Session", []string{"type"}, nil)
	terminatedSessionsTotal  = prometheus.NewDesc("rtpengine_terminated_sessions_total", "Terminated Sessions", []string{"type"}, nil)
	relayedPacketsTotal      = prometheus.NewDesc("rtpengine_relayed_packets_total", "relayedpackets", []string{"type"}, nil)
	relayedPacketErrorsTotal = prometheus.NewDesc("rtpengine_relayed_packet_errors_total", "relayedpacketerrors", []string{"type"}, nil)
	relayedBytesTotal        = prometheus.NewDesc("rtpengine_relayed_bytes_total", "relayedbytes", []string{"type"}, nil)
	streamsTotal             = prometheus.NewDesc("rtpengine_zeroway_streams_total", "zerowaystreams", []string{"type"}, nil)
	callsDurationAvg         = prometheus.NewDesc("rtpengine_calls_duration_avg", "avgcallduration", []string{}, nil)
	callsDurationTotal       = prometheus.NewDesc("rtpengine_calls_duration_total", "totalcallsduration", []string{}, nil)
	callsDuration2Total      = prometheus.NewDesc("rtpengine_calls_duration2_total", "totalcallsduration2", []string{}, nil)
	callsDurationStddev      = prometheus.NewDesc("rtpengine_calls_duration_stddev", "totalcallsduration_stddev", []string{}, nil)

	// Mos Metrics
	mosTotal        = prometheus.NewDesc("rtpengine_mos_total", "MosTotal", []string{}, nil)
	mos2Total       = prometheus.NewDesc("rtpengine_mos2_total", "Mos2Total", []string{}, nil)
	mosSamplesTotal = prometheus.NewDesc("rtpengine_mos_samples_total", "MosSamplesTotal", []string{}, nil)
	mosAverage      = prometheus.NewDesc("rtpengine_mos_avg", "MosAverage", []string{}, nil)
	mosStddev       = prometheus.NewDesc("rtpengine_mos_stddev", "MosStddev", []string{}, nil)

	// Voip Metrics
	jitterTotal                = prometheus.NewDesc("rtpengine_jitter_total", "jitterTotal", []string{}, nil)
	jitter2Total               = prometheus.NewDesc("rtpengine_jitter2_total", "jitter2Total", []string{}, nil)
	jitterSamplesTotal         = prometheus.NewDesc("rtpengine_jitter_samples_total", "jitterSamplesTotal", []string{}, nil)
	jitterAverage              = prometheus.NewDesc("rtpengine_jitter_avg", "jitterAverage", []string{}, nil)
	jitterStddev               = prometheus.NewDesc("rtpengine_jitter_stddev", "jitterStddev", []string{}, nil)
	rttE2ETotal                = prometheus.NewDesc("rtpengine_rtt_e2e_total", "rttE2ETotal", []string{}, nil)
	rttE2E2Total               = prometheus.NewDesc("rtpengine_rtt_e2e2_total", "rttE2E2Total", []string{}, nil)
	rttE2ESamplesTotal         = prometheus.NewDesc("rtpengine_rtt_e2e_samples_total", "rttE2ESamplesTotal", []string{}, nil)
	rttE2EAverage              = prometheus.NewDesc("rtpengine_rtt_e2e_avg", "rttE2EAverage", []string{}, nil)
	rttE2EStddev               = prometheus.NewDesc("rtpengine_rtt_e2e_stddev", "rttE2EStddev", []string{}, nil)
	rttDsctTotal               = prometheus.NewDesc("rtpengine_rtt_dsct_total", "rttDsctTotal", []string{}, nil)
	rttDsct2Total              = prometheus.NewDesc("rtpengine_rtt_dsct2_total", "rttDsct2Total", []string{}, nil)
	rttDsctSamplesTotal        = prometheus.NewDesc("rtpengine_rtt_dsct_samples_total", "rttDsctSamplesTotal", []string{}, nil)
	rttDsctAverage             = prometheus.NewDesc("rtpengine_rtt_dsct_avg", "rttDsctAverage", []string{}, nil)
	rttDsctStddev              = prometheus.NewDesc("rtpengine_rtt_dsct_stddev", "rttDsctStddev", []string{}, nil)
	packetLossTotal            = prometheus.NewDesc("rtpengine_packetloss_total", "packetLossTotal", []string{}, nil)
	packetLoss2Total           = prometheus.NewDesc("rtpengine_packetloss2_total", "packetLoss2Total", []string{}, nil)
	packetLossSamplesTotal     = prometheus.NewDesc("rtpengine_packetloss_samples_total", "packetLossSamplesTotal", []string{}, nil)
	packetLossAverage          = prometheus.NewDesc("rtpengine_packetloss_avg", "packetLossAverage", []string{}, nil)
	packetLossStddev           = prometheus.NewDesc("rtpengine_packetloss_stddev", "packetLossStddev", []string{}, nil)
	jitterMeasuredTotal        = prometheus.NewDesc("rtpengine_jitter_measured_total", "jitterMeasuredTotal", []string{}, nil)
	jitterMeasured2Total       = prometheus.NewDesc("rtpengine_jitter_measured2_total", "jitterMeasured2Total", []string{}, nil)
	jitterMeasuredSamplesTotal = prometheus.NewDesc("rtpengine_jitter_measured_samples_total", "jitterMeasuredSamplesTotal", []string{}, nil)
	jitterMeasuredAverage      = prometheus.NewDesc("rtpengine_jitter_measured_avg", "jitterMeasuredAverage", []string{}, nil)
	jitterMeasuredStddev       = prometheus.NewDesc("rtpengine_jitter_measured_stddev", "jitterMeasuredStddev", []string{}, nil)
	packetsLost                = prometheus.NewDesc("rtpengine_packets_lost", "packetsLost", []string{}, nil)
	rtp                        = prometheus.NewDesc("rtpengine_rtp", "RTP", []string{"type"}, nil)

	// Interfaces Metrics
	ifacePackets = prometheus.NewDesc("rtpengine_iface_packets_lost", "Interface packetsLost", []string{"name", "address", "type"}, nil)
	ifaceBytes   = prometheus.NewDesc("rtpengine_iface_bytes", "Interface Bytes", []string{"name", "address", "type"}, nil)
	ifaceErrors  = prometheus.NewDesc("rtpengine_iface_errors", "Interface Errors", []string{"name", "address", "type"}, nil)
	ifacePorts   = prometheus.NewDesc("rtpengine_iface_ports", "Interface Ports", []string{"name", "address", "type"}, nil)
)

// New processes uri, timeout and methods and returns a new Collector.
func New(uri string, timeout time.Duration, logger log.Logger) (*Collector, error) {
	var c Collector

	c.URI = uri
	c.Timeout = timeout
	c.logger = logger

	var url *url.URL
	var err error

	if _, err = url.Parse(c.URI); err != nil {
		return nil, fmt.Errorf("cannot parse URI: %w", err)
	}

	c.up = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "up",
		Help:      "Was the last scrape successful.",
	})

	c.totalScrapes = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "exporter_total_scrapes",
		Help:      "Current total freeswitch scrapes.",
	})

	c.failedScrapes = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "exporter_failed_scrapes",
		Help:      "Number of failed freeswitch scrapes.",
	})

	return &c, nil
}

// scrape will connect to the freeswitch instance and push metrics to the Prometheus channel.
func (c *Collector) scrape(ch chan<- prometheus.Metric) error {
	c.totalScrapes.Inc()

	rtpClient, err := ng.NewClient(c.URI, 5*time.Second, c.logger)
	if err != nil {
		return err
	}
	defer rtpClient.Close()
	stats, err := rtpClient.GetStatistics()
	if err != nil {
		return err
	}

	setCurrentMetrics(ch, stats.Current)
	setTotalMetrics(ch, stats.Total)
	setMosMetrics(ch, stats.Mos)
	setVoipMetrics(ch, stats.Voip)
	for _, iface := range stats.Interfaces {
		setInterfaceMetrics(ch, iface)
	}

	return nil
}

func setInterfaceMetrics(ch chan<- prometheus.Metric, iface ng.InterfacesMetrics) {
	name := iface.Name
	address := iface.Address
	ch <- prometheus.MustNewConstMetric(ifacePackets, prometheus.GaugeValue, float64(iface.PacketsLost), name, address, "lost")
	ch <- prometheus.MustNewConstMetric(ifacePackets, prometheus.GaugeValue, float64(iface.Duplicates), name, address, "duplicates")
	ch <- prometheus.MustNewConstMetric(ifacePackets, prometheus.GaugeValue, float64(iface.Ingress.Packets), name, address, "ingress")
	ch <- prometheus.MustNewConstMetric(ifacePackets, prometheus.GaugeValue, float64(iface.Egress.Packets), name, address, "egress")

	ch <- prometheus.MustNewConstMetric(ifaceBytes, prometheus.GaugeValue, float64(iface.Ingress.Bytes), name, address, "ingress")
	ch <- prometheus.MustNewConstMetric(ifaceBytes, prometheus.GaugeValue, float64(iface.Egress.Bytes), name, address, "egress")

	ch <- prometheus.MustNewConstMetric(ifaceErrors, prometheus.GaugeValue, float64(iface.Ingress.Errors), name, address, "ingress")
	ch <- prometheus.MustNewConstMetric(ifaceErrors, prometheus.GaugeValue, float64(iface.Egress.Errors), name, address, "egress")

	ch <- prometheus.MustNewConstMetric(ifacePorts, prometheus.GaugeValue, float64(iface.Ports.Min), name, address, "min")
	ch <- prometheus.MustNewConstMetric(ifacePorts, prometheus.GaugeValue, float64(iface.Ports.Max), name, address, "max")
	ch <- prometheus.MustNewConstMetric(ifacePorts, prometheus.GaugeValue, float64(iface.Ports.Used), name, address, "used")
	ch <- prometheus.MustNewConstMetric(ifacePorts, prometheus.GaugeValue, float64(iface.Ports.Free), name, address, "free")
	ch <- prometheus.MustNewConstMetric(ifacePorts, prometheus.GaugeValue, float64(iface.Ports.Totals), name, address, "total")
	ch <- prometheus.MustNewConstMetric(ifacePorts, prometheus.GaugeValue, float64(iface.Ports.Last), name, address, "last")
}

func setTotalMetrics(ch chan<- prometheus.Metric, totalMetrics ng.TotalMetrics) {
	ch <- prometheus.MustNewConstMetric(uptimeTotal, prometheus.GaugeValue, float64(totalMetrics.Uptime))
	ch <- prometheus.MustNewConstMetric(sessionsTotal, prometheus.GaugeValue, float64(totalMetrics.ManagedSessions), "managed")
	ch <- prometheus.MustNewConstMetric(sessionsTotal, prometheus.GaugeValue, float64(totalMetrics.RejectedSessions), "rejected")
	ch <- prometheus.MustNewConstMetric(sessionsTotal, prometheus.GaugeValue, float64(totalMetrics.TimeoutSessions), "timeout")
	ch <- prometheus.MustNewConstMetric(timeoutSessionsTotal, prometheus.GaugeValue, float64(totalMetrics.SilentTimeoutSessions), "silent")
	ch <- prometheus.MustNewConstMetric(timeoutSessionsTotal, prometheus.GaugeValue, float64(totalMetrics.FinalTimeoutSessions), "final")
	ch <- prometheus.MustNewConstMetric(timeoutSessionsTotal, prometheus.GaugeValue, float64(totalMetrics.OfferTimeoutSessions), "offer")

	ch <- prometheus.MustNewConstMetric(terminatedSessionsTotal, prometheus.GaugeValue, float64(totalMetrics.RegularTerminatedSessions), "regular")
	ch <- prometheus.MustNewConstMetric(terminatedSessionsTotal, prometheus.GaugeValue, float64(totalMetrics.ForcedTerminatedSessions), "forced")

	ch <- prometheus.MustNewConstMetric(relayedPacketsTotal, prometheus.GaugeValue, float64(totalMetrics.RelayedPackets), "all")
	ch <- prometheus.MustNewConstMetric(relayedPacketsTotal, prometheus.GaugeValue, float64(totalMetrics.RelayedPacketsKernel), "kernel")
	ch <- prometheus.MustNewConstMetric(relayedPacketsTotal, prometheus.GaugeValue, float64(totalMetrics.RelayedPacketsUser), "user")

	ch <- prometheus.MustNewConstMetric(relayedPacketErrorsTotal, prometheus.GaugeValue, float64(totalMetrics.RelayedPacketErrors), "all")
	ch <- prometheus.MustNewConstMetric(relayedPacketErrorsTotal, prometheus.GaugeValue, float64(totalMetrics.RelayedPacketErrorsKernel), "kernel")
	ch <- prometheus.MustNewConstMetric(relayedPacketErrorsTotal, prometheus.GaugeValue, float64(totalMetrics.RelayedPacketErrorsUser), "user")

	ch <- prometheus.MustNewConstMetric(relayedBytesTotal, prometheus.GaugeValue, float64(totalMetrics.RelayedBytes), "all")
	ch <- prometheus.MustNewConstMetric(relayedBytesTotal, prometheus.GaugeValue, float64(totalMetrics.RelayedBytesKernel), "kernel")
	ch <- prometheus.MustNewConstMetric(relayedBytesTotal, prometheus.GaugeValue, float64(totalMetrics.RelayedBytesUser), "user")

	ch <- prometheus.MustNewConstMetric(streamsTotal, prometheus.GaugeValue, float64(totalMetrics.ZerowayStreams), "zeroway")
	ch <- prometheus.MustNewConstMetric(streamsTotal, prometheus.GaugeValue, float64(totalMetrics.ZerowayStreams), "oneway")

	ch <- prometheus.MustNewConstMetric(callsDurationAvg, prometheus.GaugeValue, totalMetrics.AvgCallDuration)
	ch <- prometheus.MustNewConstMetric(callsDurationTotal, prometheus.GaugeValue, totalMetrics.TotalCallsDuration)
	ch <- prometheus.MustNewConstMetric(callsDuration2Total, prometheus.GaugeValue, totalMetrics.TotalCallsDuration2)
	ch <- prometheus.MustNewConstMetric(callsDurationStddev, prometheus.GaugeValue, totalMetrics.TotalCallsDurationStddev)
}

func setCurrentMetrics(ch chan<- prometheus.Metric, currentMetrics ng.CurrentMetrics) {
	ch <- prometheus.MustNewConstMetric(sessions, prometheus.GaugeValue, float64(currentMetrics.SessionsForeign), "foreign")
	ch <- prometheus.MustNewConstMetric(sessions, prometheus.GaugeValue, float64(currentMetrics.SessionsOwn), "own")
	ch <- prometheus.MustNewConstMetric(sessions, prometheus.GaugeValue, float64(currentMetrics.SessionsTotal), "total")

	ch <- prometheus.MustNewConstMetric(errorRate, prometheus.GaugeValue, float64(currentMetrics.ErrorRate), "all")
	ch <- prometheus.MustNewConstMetric(errorRate, prometheus.GaugeValue, float64(currentMetrics.ErrorRateKernel), "kernel")
	ch <- prometheus.MustNewConstMetric(errorRate, prometheus.GaugeValue, float64(currentMetrics.ErrorRateUser), "user")

	ch <- prometheus.MustNewConstMetric(byteRate, prometheus.GaugeValue, float64(currentMetrics.ByteRate), "all")
	ch <- prometheus.MustNewConstMetric(byteRate, prometheus.GaugeValue, float64(currentMetrics.ByteRateKernel), "kernel")
	ch <- prometheus.MustNewConstMetric(byteRate, prometheus.GaugeValue, float64(currentMetrics.ByteRateUser), "user")

	ch <- prometheus.MustNewConstMetric(packetRate, prometheus.GaugeValue, float64(currentMetrics.PacketRate), "all")
	ch <- prometheus.MustNewConstMetric(packetRate, prometheus.GaugeValue, float64(currentMetrics.PacketRateKernel), "kernel")
	ch <- prometheus.MustNewConstMetric(packetRate, prometheus.GaugeValue, float64(currentMetrics.PacketRateUser), "user")

	ch <- prometheus.MustNewConstMetric(media, prometheus.GaugeValue, float64(currentMetrics.TranscodedMedia), "transcoded")
	ch <- prometheus.MustNewConstMetric(media, prometheus.GaugeValue, float64(currentMetrics.MediaKernel), "kernel")
	ch <- prometheus.MustNewConstMetric(media, prometheus.GaugeValue, float64(currentMetrics.MediaUserspace), "user")
	ch <- prometheus.MustNewConstMetric(media, prometheus.GaugeValue, float64(currentMetrics.MediaMixed), "mixed")
}

func setVoipMetrics(ch chan<- prometheus.Metric, voipMetrics ng.VoipMetrics) {
	ch <- prometheus.MustNewConstMetric(jitterTotal, prometheus.GaugeValue, float64(voipMetrics.JitterTotal))
	ch <- prometheus.MustNewConstMetric(jitter2Total, prometheus.GaugeValue, float64(voipMetrics.Jitter2Total))
	ch <- prometheus.MustNewConstMetric(jitterSamplesTotal, prometheus.GaugeValue, float64(voipMetrics.JitterSamplesTotal))
	ch <- prometheus.MustNewConstMetric(jitterAverage, prometheus.GaugeValue, float64(voipMetrics.JitterAverage))
	ch <- prometheus.MustNewConstMetric(jitterStddev, prometheus.GaugeValue, float64(voipMetrics.JitterStddev))
	ch <- prometheus.MustNewConstMetric(rttE2ETotal, prometheus.GaugeValue, float64(voipMetrics.RttE2ETotal))
	ch <- prometheus.MustNewConstMetric(rttE2E2Total, prometheus.GaugeValue, float64(voipMetrics.RttE2E2Total))
	ch <- prometheus.MustNewConstMetric(rttE2ESamplesTotal, prometheus.GaugeValue, float64(voipMetrics.RttE2ESamplesTotal))
	ch <- prometheus.MustNewConstMetric(rttE2EAverage, prometheus.GaugeValue, float64(voipMetrics.RttE2EAverage))
	ch <- prometheus.MustNewConstMetric(rttE2EStddev, prometheus.GaugeValue, float64(voipMetrics.RttE2EStddev))
	ch <- prometheus.MustNewConstMetric(rttDsctTotal, prometheus.GaugeValue, float64(voipMetrics.RttDsctTotal))
	ch <- prometheus.MustNewConstMetric(rttDsct2Total, prometheus.GaugeValue, float64(voipMetrics.RttDsct2Total))
	ch <- prometheus.MustNewConstMetric(rttDsctSamplesTotal, prometheus.GaugeValue, float64(voipMetrics.RttDsctSamplesTotal))
	ch <- prometheus.MustNewConstMetric(rttDsctAverage, prometheus.GaugeValue, float64(voipMetrics.RttDsctAverage))
	ch <- prometheus.MustNewConstMetric(rttDsctStddev, prometheus.GaugeValue, float64(voipMetrics.RttDsctStddev))
	ch <- prometheus.MustNewConstMetric(packetLossTotal, prometheus.GaugeValue, float64(voipMetrics.PacketLossTotal))
	ch <- prometheus.MustNewConstMetric(packetLoss2Total, prometheus.GaugeValue, float64(voipMetrics.PacketLoss2Total))
	ch <- prometheus.MustNewConstMetric(packetLossSamplesTotal, prometheus.GaugeValue, float64(voipMetrics.PacketLossSamplesTotal))
	ch <- prometheus.MustNewConstMetric(packetLossAverage, prometheus.GaugeValue, float64(voipMetrics.PacketLossAverage))
	ch <- prometheus.MustNewConstMetric(packetLossStddev, prometheus.GaugeValue, float64(voipMetrics.PacketLossStddev))
	ch <- prometheus.MustNewConstMetric(jitterMeasuredTotal, prometheus.GaugeValue, float64(voipMetrics.JitterMeasuredTotal))
	ch <- prometheus.MustNewConstMetric(jitterMeasured2Total, prometheus.GaugeValue, float64(voipMetrics.JitterMeasured2Total))
	ch <- prometheus.MustNewConstMetric(jitterMeasuredSamplesTotal, prometheus.GaugeValue, float64(voipMetrics.JitterMeasuredSamplesTotal))
	ch <- prometheus.MustNewConstMetric(jitterMeasuredAverage, prometheus.GaugeValue, float64(voipMetrics.JitterMeasuredAverage))
	ch <- prometheus.MustNewConstMetric(jitterMeasuredStddev, prometheus.GaugeValue, float64(voipMetrics.JitterMeasuredStddev))
	ch <- prometheus.MustNewConstMetric(packetsLost, prometheus.GaugeValue, float64(voipMetrics.PacketsLost))

	ch <- prometheus.MustNewConstMetric(rtp, prometheus.GaugeValue, float64(voipMetrics.RtpDuplicates), "duplicates")
	ch <- prometheus.MustNewConstMetric(rtp, prometheus.GaugeValue, float64(voipMetrics.RtpSkips), "skips")
	ch <- prometheus.MustNewConstMetric(rtp, prometheus.GaugeValue, float64(voipMetrics.RtpSeqResets), "seq_resets")
	ch <- prometheus.MustNewConstMetric(rtp, prometheus.GaugeValue, float64(voipMetrics.RtpReordered), "reordered")
}

func setMosMetrics(ch chan<- prometheus.Metric, mos ng.MosMetrics) {
	ch <- prometheus.MustNewConstMetric(mosTotal, prometheus.GaugeValue, mos.MosTotal)
	ch <- prometheus.MustNewConstMetric(mos2Total, prometheus.GaugeValue, mos.Mos2Total)
	ch <- prometheus.MustNewConstMetric(mosSamplesTotal, prometheus.GaugeValue, float64(mos.MosSamplesTotal))
	ch <- prometheus.MustNewConstMetric(mosAverage, prometheus.GaugeValue, mos.MosAverage)
	ch <- prometheus.MustNewConstMetric(mosStddev, prometheus.GaugeValue, mos.MosStddev)
}

// Describe implements prometheus.Collector.
func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	prometheus.DescribeByCollect(c, ch)
}

// Collect implements prometheus.Collector.
func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if err := c.scrape(ch); err != nil {
		level.Error(c.logger).Log("msg", "Error scraping rtpengine:", "err", err)
		c.failedScrapes.Inc()
		c.failedScrapes.Collect(ch)
		c.up.Set(0)
	} else {
		c.up.Set(1)
	}

	ch <- c.up
	ch <- c.totalScrapes
}
