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

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"go.voiplens.io/rtpengine/ng"
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
	sessions        = prometheus.NewDesc("rtpengine_sessions", "Sessions by type", []string{"type"}, nil)
	errorRate       = prometheus.NewDesc("rtpengine_error_rate", "Error Rate by type", []string{"type"}, nil)
	packetRate      = prometheus.NewDesc("rtpengine_packet_rate", "Packet Rate by type", []string{"type"}, nil)
	byteRate        = prometheus.NewDesc("rtpengine_byte_rate", "Byte Rate by type", []string{"type"}, nil)
	mediaStreams    = prometheus.NewDesc("rtpengine_mediastreams", "Media Streams by type", []string{"type"}, nil)
	transcodedMedia = prometheus.NewDesc("rtpengine_transcoded_media", "Transcoded media", []string{}, nil)

	// Total Metrics
	managedSessionsTotal     = prometheus.NewDesc("rtpengine_sessions_total", "Total managed sessions", []string{}, nil)
	uptimeTotal              = prometheus.NewDesc("rtpengine_uptime_seconds", "Uptime of rtpengine", []string{}, nil)
	closedSessionsTotal      = prometheus.NewDesc("rtpengine_closed_sessions_total", "Total closed Sessions by reason", []string{"reason"}, nil)
	relayedPacketsTotal      = prometheus.NewDesc("rtpengine_packets_total", "Total relayed packets by type", []string{"type"}, nil)
	relayedPacketErrorsTotal = prometheus.NewDesc("rtpengine_packet_errors_total", "Total relayed packet errors type", []string{"type"}, nil)
	relayedBytesTotal        = prometheus.NewDesc("rtpengine_bytes_total", "Total relayed bytes by type", []string{"type"}, nil)
	zeroPacketStreamsTotal   = prometheus.NewDesc("rtpengine_zero_packet_streams_total", "Total number of streams with no relayed packets", []string{}, nil)
	oneWaySessionsTotal      = prometheus.NewDesc("rtpengine_one_way_sessions_total", "Total number of 1-way streams", []string{}, nil)
	callsDurationAverage     = prometheus.NewDesc("rtpengine_call_duration_avg", "Average call duration", []string{}, nil)
	callsDurationTotal       = prometheus.NewDesc("rtpengine_call_duration_total", "Total calls duration", []string{}, nil)
	callsDuration2Total      = prometheus.NewDesc("rtpengine_call_duration2_total", "Total calls duration squared", []string{}, nil)
	callsDurationStddev      = prometheus.NewDesc("rtpengine_call_duration_stddev", "Stddev call duration", []string{}, nil)

	// Mos Metrics
	mosTotal        = prometheus.NewDesc("rtpengine_mos_total", "Sum of all MOS values sampled", []string{}, nil)
	mos2Total       = prometheus.NewDesc("rtpengine_mos2_total", "Sum of all MOS square values sampled", []string{}, nil)
	mosSamplesTotal = prometheus.NewDesc("rtpengine_mos_samples_total", "Total number of MOS samples", []string{}, nil)
	mosAverage      = prometheus.NewDesc("rtpengine_mos_avg", "Average of all MOS values sampled", []string{}, nil)
	mosStddev       = prometheus.NewDesc("rtpengine_mos_stddev", "Stddev of all MOS values sampled", []string{}, nil)

	// Voip Metrics
	jitterTotal                = prometheus.NewDesc("rtpengine_jitter_total", "Sum of all jitter (reported) values sampled", []string{}, nil)
	jitter2Total               = prometheus.NewDesc("rtpengine_jitter2_total", "Sum of all jitter (reported) square values sampled", []string{}, nil)
	jitterSamplesTotal         = prometheus.NewDesc("rtpengine_jitter_samples_total", "Total number of jitter (reported) samples", []string{}, nil)
	jitterAverage              = prometheus.NewDesc("rtpengine_jitter_avg", "Average of all jitter (reported) samples", []string{}, nil)
	jitterStddev               = prometheus.NewDesc("rtpengine_jitter_stddev", "Stddev of all jitter (reported) values sampled", []string{}, nil)
	rttE2ETotal                = prometheus.NewDesc("rtpengine_rtt_e2e_total", "Sum of all end-to-end round-trip time values sampled", []string{}, nil)
	rttE2E2Total               = prometheus.NewDesc("rtpengine_rtt_e2e2_total", "Sum of all end-to-end round-trip time square values sampled", []string{}, nil)
	rttE2ESamplesTotal         = prometheus.NewDesc("rtpengine_rtt_e2e_samples_total", "Total number of end-to-end round-trip time samples", []string{}, nil)
	rttE2EAverage              = prometheus.NewDesc("rtpengine_rtt_e2e_avg", "Average of all end-to-end round-trip time values sampled", []string{}, nil)
	rttE2EStddev               = prometheus.NewDesc("rtpengine_rtt_e2e_stddev", "Stddev of all end-to-end round-trip time values sampled", []string{}, nil)
	rttDsctTotal               = prometheus.NewDesc("rtpengine_rtt_dsct_total", "Sum of all discrete round-trip time values sampled", []string{}, nil)
	rttDsct2Total              = prometheus.NewDesc("rtpengine_rtt_dsct2_total", "Sum of all discrete round-trip time square values sampled", []string{}, nil)
	rttDsctSamplesTotal        = prometheus.NewDesc("rtpengine_rtt_dsct_samples_total", "Total number of discrete round-trip time samples", []string{}, nil)
	rttDsctAverage             = prometheus.NewDesc("rtpengine_rtt_dsct_avg", "Average of all discrete round-trip time values sampled", []string{}, nil)
	rttDsctStddev              = prometheus.NewDesc("rtpengine_rtt_dsct_stddev", "Stddev of all discrete round-trip time values sampled", []string{}, nil)
	packetLossTotal            = prometheus.NewDesc("rtpengine_packetloss_total", "Sum of all packet loss values sampled", []string{}, nil)
	packetLoss2Total           = prometheus.NewDesc("rtpengine_packetloss2_total", "Sum of all packet loss square values sampled", []string{}, nil)
	packetLossSamplesTotal     = prometheus.NewDesc("rtpengine_packetloss_samples_total", "Total number of packet loss samples", []string{}, nil)
	packetLossAverage          = prometheus.NewDesc("rtpengine_packetloss_avg", "Average of all packet loss values sampled", []string{}, nil)
	packetLossStddev           = prometheus.NewDesc("rtpengine_packetloss_stddev", "Stddev of all packet loss values sampled", []string{}, nil)
	jitterMeasuredTotal        = prometheus.NewDesc("rtpengine_jitter_measured_total", "Sum of all jitter (measured) values sampled", []string{}, nil)
	jitterMeasured2Total       = prometheus.NewDesc("rtpengine_jitter_measured2_total", "Sum of all jitter (measured) square values sampled", []string{}, nil)
	jitterMeasuredSamplesTotal = prometheus.NewDesc("rtpengine_jitter_measured_samples_total", "Total number of jitter (measured) samples", []string{}, nil)
	jitterMeasuredAverage      = prometheus.NewDesc("rtpengine_jitter_measured_avg", "Average of all jitter (measured) square values sampled", []string{}, nil)
	jitterMeasuredStddev       = prometheus.NewDesc("rtpengine_jitter_measured_stddev", "Stddev of all jitter (measured) square values sampled", []string{}, nil)
	packetsLost                = prometheus.NewDesc("rtpengine_packets_lost", "Packets lost", []string{}, nil)
	rtpReordered               = prometheus.NewDesc("rtpengine_rtp_reordered", "Out-of-order RTP packets", []string{}, nil)
	rtpDuplicates              = prometheus.NewDesc("rtpengine_rtp_duplicates", "Duplicate RTP packets", []string{}, nil)
	rtpSkips                   = prometheus.NewDesc("rtpengine_rtp_skips", "RTP sequence skips", []string{}, nil)
	rtpSeqResets               = prometheus.NewDesc("rtpengine_rtp_seq_resets", "RTP sequence resets", []string{}, nil)

	// Interfaces Metrics
	ifacePackets     = prometheus.NewDesc("rtpengine_interface_packets", "Interface Packets", []string{"name", "address", "direction"}, nil)
	ifacePacketsLost = prometheus.NewDesc("rtpengine_interface_packets_lost", "Interface Packets Lost", []string{"name", "address"}, nil)
	ifaceDuplicates  = prometheus.NewDesc("rtpengine_interface_duplicates", "Interface Duplicates", []string{"name", "address"}, nil)
	ifaceBytes       = prometheus.NewDesc("rtpengine_interface_bytes", "Interface Bytes", []string{"name", "address", "direction"}, nil)
	ifaceErrors      = prometheus.NewDesc("rtpengine_interface_errors", "Interface Errors", []string{"name", "address", "direction"}, nil)
	ifacePorts       = prometheus.NewDesc("rtpengine_interface_ports", "Interface Ports", []string{"name", "address", "type"}, nil)
	ifacePortsLast   = prometheus.NewDesc("rtpengine_ports", "Current Ports", []string{"name", "address"}, nil)
	ifacePortsUsed   = prometheus.NewDesc("rtpengine_ports_used", "Used Ports", []string{"name", "address"}, nil)
	ifacePortsFree   = prometheus.NewDesc("rtpengine_ports_free", "Free Ports", []string{"name", "address"}, nil)

	ifaceJitterMeasuredTotal       = prometheus.NewDesc("rtpengine_interface_jitter_measured_total", "Sum of all jitter (measured) values sampled", []string{"name", "address"}, nil)
	ifaceJitterMeasured2Total      = prometheus.NewDesc("rtpengine_interface_jitter_measured2_total", "Sum of all jitter (measured) square values sampled", []string{"name", "address"}, nil)
	ifaceJitterMeasuredSampleTotal = prometheus.NewDesc("rtpengine_interface_jitter_measured_samples_total", "Total number of jitter (measured) samples", []string{"name", "address"}, nil)
	ifaceJitterMeasuredAverage     = prometheus.NewDesc("rtpengine_interface_jitter_measured_avg", "Average of all jitter (measured) values sampled", []string{"name", "address"}, nil)
	ifaceJitterMeasuredStddev      = prometheus.NewDesc("rtpengine_interface_jitter_measured_stddev", "Stddev of all jitter (measured) values sampled", []string{"name", "address"}, nil)

	ifacePacketslossTotal       = prometheus.NewDesc("rtpengine_interface_packetloss_total", "Sum of all packet loss values sampled", []string{"name", "address"}, nil)
	ifacePacketsloss2Total      = prometheus.NewDesc("rtpengine_interface_packetloss2_total", "Sum of all packet loss square values sampled", []string{"name", "address"}, nil)
	ifacePacketslossSampleTotal = prometheus.NewDesc("rtpengine_interface_packetloss_samples_total", "Total number of packet loss samples", []string{"name", "address"}, nil)
	ifacePacketslossAverage     = prometheus.NewDesc("rtpengine_interface_packetloss_avg", "Average of all packet loss values sampled", []string{"name", "address"}, nil)
	ifacePacketslossStddev      = prometheus.NewDesc("rtpengine_interface_packetloss_stddev", "Stddev of all packet loss values sampled", []string{"name", "address"}, nil)

	ifaceMosTotal       = prometheus.NewDesc("rtpengine_interface_mos_total", "Sum of all MOS values sampled", []string{"name", "address"}, nil)
	ifaceMos2Total      = prometheus.NewDesc("rtpengine_interface_mos2_total", "Sum of all MOS square values sampled", []string{"name", "address"}, nil)
	ifaceMosSampleTotal = prometheus.NewDesc("rtpengine_interface_mos_samples_total", "Total number of MOS samples", []string{"name", "address"}, nil)
	ifaceMosAverage     = prometheus.NewDesc("rtpengine_interface_mos_avg", "Average of all MOS values sampled", []string{"name", "address"}, nil)
	ifaceMosStddev      = prometheus.NewDesc("rtpengine_interface_mos_stddev", "Stddev of all MOS values sampled", []string{"name", "address"}, nil)

	ifaceJitterTotal       = prometheus.NewDesc("rtpengine_interface_jitter_total", "Sum of all jitter values sampled", []string{"name", "address"}, nil)
	ifaceJitter2Total      = prometheus.NewDesc("rtpengine_interface_jitter2_total", "Sum of all jitter square values sampled", []string{"name", "address"}, nil)
	ifaceJitterSampleTotal = prometheus.NewDesc("rtpengine_interface_jitter_samples_total", "Total number of jitter samples", []string{"name", "address"}, nil)
	ifaceJitterAverage     = prometheus.NewDesc("rtpengine_interface_jitter_avg", "Average of all jitter values sampled", []string{"name", "address"}, nil)
	ifaceJitterStddev      = prometheus.NewDesc("rtpengine_interface_jitter_stddev", "Stddev of all jitter values sampled", []string{"name", "address"}, nil)

	ifaceRttE2ETotal       = prometheus.NewDesc("rtpengine_interface_rtt_e2e_total", "Sum of all end-to-end round-trip time values sampled", []string{"name", "address"}, nil)
	ifaceRttE2E2Total      = prometheus.NewDesc("rtpengine_interface_rtt_e2e2_total", "Sum of all end-to-end round-trip time square values sampled", []string{"name", "address"}, nil)
	ifaceRttE2ESampleTotal = prometheus.NewDesc("rtpengine_interface_rtt_e2e_samples_total", "Total number of end-to-end round-trip time samples", []string{"name", "address"}, nil)
	ifaceRttE2EAverage     = prometheus.NewDesc("rtpengine_interface_rtt_e2e_avg", "Average of all end-to-end round-trip time values sampled", []string{"name", "address"}, nil)
	ifaceRttE2EStddev      = prometheus.NewDesc("rtpengine_interface_rtt_e2e_stddev", "Stddev of all end-to-end round-trip time values sampled", []string{"name", "address"}, nil)

	ifaceRttDsctTotal       = prometheus.NewDesc("rtpengine_interface_rtt_dsct_total", "Sum of all discrete round-trip time values sampled", []string{"name", "address"}, nil)
	ifaceRttDsct2Total      = prometheus.NewDesc("rtpengine_interface_rtt_dsct2_total", "Sum of all discrete round-trip time square values sampled", []string{"name", "address"}, nil)
	ifaceRttDsctSampleTotal = prometheus.NewDesc("rtpengine_interface_rtt_dsct_samples_total", "Total number of discrete round-trip time samples", []string{"name", "address"}, nil)
	ifaceRttDsctAverage     = prometheus.NewDesc("rtpengine_interface_rtt_dsct_avg", "Average of all discrete round-trip time values sampled", []string{"name", "address"}, nil)
	ifaceRttDsctStddev      = prometheus.NewDesc("rtpengine_interface_rtt_dsct_stddev", "Stddev of all discrete round-trip time values sampled", []string{"name", "address"}, nil)

	// Proxy Metrics
	proxyRequestsTotal    = prometheus.NewDesc("rtpengine_requests_total", "Requests total", []string{"proxy", "request"}, nil)
	proxyRequestsDuration = prometheus.NewDesc("rtpengine_request_seconds_total", "Requests duration", []string{"proxy", "request"}, nil)
	proxyErrorsTotal      = prometheus.NewDesc("rtpengine_errors_total", "Requests duration", []string{"proxy"}, nil)
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

	for _, proxy := range stats.Control.Proxies {
		setProxyMetrics(ch, proxy)
	}

	return nil
}

func setInterfaceMetrics(ch chan<- prometheus.Metric, iface ng.InterfacesMetrics) {
	name := iface.Name
	address := iface.Address
	ch <- prometheus.MustNewConstMetric(ifacePacketsLost, prometheus.CounterValue, float64(iface.PacketsLost), name, address)
	ch <- prometheus.MustNewConstMetric(ifaceDuplicates, prometheus.CounterValue, float64(iface.Duplicates), name, address)

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

	ch <- prometheus.MustNewConstMetric(ifacePortsUsed, prometheus.GaugeValue, float64(iface.Ports.Used), name, address)
	ch <- prometheus.MustNewConstMetric(ifacePortsFree, prometheus.GaugeValue, float64(iface.Ports.Free), name, address)
	ch <- prometheus.MustNewConstMetric(ifacePortsLast, prometheus.GaugeValue, float64(iface.Ports.Totals), name, address)

	ch <- prometheus.MustNewConstMetric(ifaceJitterMeasuredTotal, prometheus.CounterValue, iface.VoipMetrics.JitterMeasuredTotal, name, address)
	ch <- prometheus.MustNewConstMetric(ifaceJitterMeasured2Total, prometheus.CounterValue, iface.VoipMetrics.JitterMeasured2Total, name, address)
	ch <- prometheus.MustNewConstMetric(ifaceJitterMeasuredSampleTotal, prometheus.CounterValue, float64(iface.VoipMetrics.JitterSamplesTotal), name, address)
	ch <- prometheus.MustNewConstMetric(ifaceJitterMeasuredAverage, prometheus.GaugeValue, iface.VoipMetrics.JitterMeasuredAverage, name, address)
	ch <- prometheus.MustNewConstMetric(ifaceJitterMeasuredStddev, prometheus.CounterValue, iface.VoipMetrics.JitterMeasuredStddev, name, address)

	ch <- prometheus.MustNewConstMetric(ifacePacketslossTotal, prometheus.CounterValue, iface.VoipMetrics.PacketlossTotal, name, address)
	ch <- prometheus.MustNewConstMetric(ifacePacketsloss2Total, prometheus.CounterValue, iface.VoipMetrics.Packetloss2Total, name, address)
	ch <- prometheus.MustNewConstMetric(ifacePacketslossSampleTotal, prometheus.CounterValue, float64(iface.VoipMetrics.PacketlossSamplesTotal), name, address)
	ch <- prometheus.MustNewConstMetric(ifacePacketslossAverage, prometheus.GaugeValue, iface.VoipMetrics.PacketlossAverage, name, address)
	ch <- prometheus.MustNewConstMetric(ifacePacketslossStddev, prometheus.CounterValue, iface.VoipMetrics.PacketlossStddev, name, address)

	ch <- prometheus.MustNewConstMetric(ifaceMosTotal, prometheus.CounterValue, iface.VoipMetrics.MosTotal, name, address)
	ch <- prometheus.MustNewConstMetric(ifaceMos2Total, prometheus.CounterValue, iface.VoipMetrics.Mos2Total, name, address)
	ch <- prometheus.MustNewConstMetric(ifaceMosSampleTotal, prometheus.CounterValue, float64(iface.VoipMetrics.MosSamplesTotal), name, address)
	ch <- prometheus.MustNewConstMetric(ifaceMosAverage, prometheus.GaugeValue, iface.VoipMetrics.MosAverage, name, address)
	ch <- prometheus.MustNewConstMetric(ifaceMosStddev, prometheus.CounterValue, iface.VoipMetrics.MosStddev, name, address)

	ch <- prometheus.MustNewConstMetric(ifaceJitterTotal, prometheus.CounterValue, iface.VoipMetrics.JitterTotal, name, address)
	ch <- prometheus.MustNewConstMetric(ifaceJitter2Total, prometheus.CounterValue, iface.VoipMetrics.Jitter2Total, name, address)
	ch <- prometheus.MustNewConstMetric(ifaceJitterSampleTotal, prometheus.CounterValue, float64(iface.VoipMetrics.JitterSamplesTotal), name, address)
	ch <- prometheus.MustNewConstMetric(ifaceJitterAverage, prometheus.GaugeValue, iface.VoipMetrics.JitterAverage, name, address)
	ch <- prometheus.MustNewConstMetric(ifaceJitterStddev, prometheus.CounterValue, iface.VoipMetrics.JitterStddev, name, address)

	ch <- prometheus.MustNewConstMetric(ifaceRttE2ETotal, prometheus.CounterValue, iface.VoipMetrics.RttE2ETotal, name, address)
	ch <- prometheus.MustNewConstMetric(ifaceRttE2E2Total, prometheus.CounterValue, iface.VoipMetrics.RttE2E2Total, name, address)
	ch <- prometheus.MustNewConstMetric(ifaceRttE2ESampleTotal, prometheus.CounterValue, float64(iface.VoipMetrics.RttE2ESamplesTotal), name, address)
	ch <- prometheus.MustNewConstMetric(ifaceRttE2EAverage, prometheus.GaugeValue, iface.VoipMetrics.RttE2EAverage, name, address)
	ch <- prometheus.MustNewConstMetric(ifaceRttE2EStddev, prometheus.CounterValue, iface.VoipMetrics.RttE2EStddev, name, address)

	ch <- prometheus.MustNewConstMetric(ifaceRttDsctTotal, prometheus.CounterValue, iface.VoipMetrics.RttDsctTotal, name, address)
	ch <- prometheus.MustNewConstMetric(ifaceRttDsct2Total, prometheus.CounterValue, iface.VoipMetrics.RttDsct2Total, name, address)
	ch <- prometheus.MustNewConstMetric(ifaceRttDsctSampleTotal, prometheus.CounterValue, float64(iface.VoipMetrics.RttDsctSamplesTotal), name, address)
	ch <- prometheus.MustNewConstMetric(ifaceRttDsctAverage, prometheus.GaugeValue, iface.VoipMetrics.RttDsctAverage, name, address)
	ch <- prometheus.MustNewConstMetric(ifaceRttDsctStddev, prometheus.CounterValue, iface.VoipMetrics.RttDsctStddev, name, address)
}

func setTotalMetrics(ch chan<- prometheus.Metric, totalMetrics ng.TotalMetrics) {
	ch <- prometheus.MustNewConstMetric(uptimeTotal, prometheus.GaugeValue, float64(totalMetrics.Uptime))

	ch <- prometheus.MustNewConstMetric(managedSessionsTotal, prometheus.CounterValue, float64(totalMetrics.ManagedSessions))
	ch <- prometheus.MustNewConstMetric(closedSessionsTotal, prometheus.CounterValue, float64(totalMetrics.ManagedSessions), "managed")
	ch <- prometheus.MustNewConstMetric(closedSessionsTotal, prometheus.CounterValue, float64(totalMetrics.RejectedSessions), "rejected")
	ch <- prometheus.MustNewConstMetric(closedSessionsTotal, prometheus.CounterValue, float64(totalMetrics.TimeoutSessions), "timeout")
	ch <- prometheus.MustNewConstMetric(closedSessionsTotal, prometheus.CounterValue, float64(totalMetrics.SilentTimeoutSessions), "silent_timeout")
	ch <- prometheus.MustNewConstMetric(closedSessionsTotal, prometheus.CounterValue, float64(totalMetrics.FinalTimeoutSessions), "final_timeout")
	ch <- prometheus.MustNewConstMetric(closedSessionsTotal, prometheus.CounterValue, float64(totalMetrics.OfferTimeoutSessions), "offer_timeout")
	ch <- prometheus.MustNewConstMetric(closedSessionsTotal, prometheus.CounterValue, float64(totalMetrics.RegularTerminatedSessions), "terminated")
	ch <- prometheus.MustNewConstMetric(closedSessionsTotal, prometheus.CounterValue, float64(totalMetrics.ForcedTerminatedSessions), "force_terminated")

	ch <- prometheus.MustNewConstMetric(relayedPacketsTotal, prometheus.CounterValue, float64(totalMetrics.RelayedPackets), "all")
	ch <- prometheus.MustNewConstMetric(relayedPacketsTotal, prometheus.CounterValue, float64(totalMetrics.RelayedPacketsKernel), "kernel")
	ch <- prometheus.MustNewConstMetric(relayedPacketsTotal, prometheus.CounterValue, float64(totalMetrics.RelayedPacketsUser), "userspace")

	ch <- prometheus.MustNewConstMetric(relayedPacketErrorsTotal, prometheus.CounterValue, float64(totalMetrics.RelayedPacketErrors), "all")
	ch <- prometheus.MustNewConstMetric(relayedPacketErrorsTotal, prometheus.CounterValue, float64(totalMetrics.RelayedPacketErrorsKernel), "kernel")
	ch <- prometheus.MustNewConstMetric(relayedPacketErrorsTotal, prometheus.CounterValue, float64(totalMetrics.RelayedPacketErrorsUser), "userspace")

	ch <- prometheus.MustNewConstMetric(relayedBytesTotal, prometheus.CounterValue, float64(totalMetrics.RelayedBytes), "all")
	ch <- prometheus.MustNewConstMetric(relayedBytesTotal, prometheus.CounterValue, float64(totalMetrics.RelayedBytesKernel), "kernel")
	ch <- prometheus.MustNewConstMetric(relayedBytesTotal, prometheus.CounterValue, float64(totalMetrics.RelayedBytesUser), "userspace")

	ch <- prometheus.MustNewConstMetric(zeroPacketStreamsTotal, prometheus.CounterValue, float64(totalMetrics.ZerowayStreams))
	ch <- prometheus.MustNewConstMetric(oneWaySessionsTotal, prometheus.CounterValue, float64(totalMetrics.OnewayStreams))

	ch <- prometheus.MustNewConstMetric(callsDurationAverage, prometheus.CounterValue, totalMetrics.AvgCallDuration)
	ch <- prometheus.MustNewConstMetric(callsDurationTotal, prometheus.CounterValue, totalMetrics.TotalCallsDuration)
	ch <- prometheus.MustNewConstMetric(callsDuration2Total, prometheus.CounterValue, totalMetrics.TotalCallsDuration2)
	ch <- prometheus.MustNewConstMetric(callsDurationStddev, prometheus.CounterValue, totalMetrics.TotalCallsDurationStddev)
}

func setCurrentMetrics(ch chan<- prometheus.Metric, currentMetrics ng.CurrentMetrics) {
	ch <- prometheus.MustNewConstMetric(sessions, prometheus.GaugeValue, float64(currentMetrics.SessionsForeign), "foreign")
	ch <- prometheus.MustNewConstMetric(sessions, prometheus.GaugeValue, float64(currentMetrics.SessionsOwn), "own")

	ch <- prometheus.MustNewConstMetric(errorRate, prometheus.GaugeValue, float64(currentMetrics.ErrorRate), "all")
	ch <- prometheus.MustNewConstMetric(errorRate, prometheus.GaugeValue, float64(currentMetrics.ErrorRateKernel), "kernel")
	ch <- prometheus.MustNewConstMetric(errorRate, prometheus.GaugeValue, float64(currentMetrics.ErrorRateUser), "userspace")

	ch <- prometheus.MustNewConstMetric(byteRate, prometheus.GaugeValue, float64(currentMetrics.ByteRate), "all")
	ch <- prometheus.MustNewConstMetric(byteRate, prometheus.GaugeValue, float64(currentMetrics.ByteRateKernel), "kernel")
	ch <- prometheus.MustNewConstMetric(byteRate, prometheus.GaugeValue, float64(currentMetrics.ByteRateUser), "userspace")

	ch <- prometheus.MustNewConstMetric(packetRate, prometheus.GaugeValue, float64(currentMetrics.PacketRate), "all")
	ch <- prometheus.MustNewConstMetric(packetRate, prometheus.GaugeValue, float64(currentMetrics.PacketRateKernel), "kernel")
	ch <- prometheus.MustNewConstMetric(packetRate, prometheus.GaugeValue, float64(currentMetrics.PacketRateUser), "userspace")

	ch <- prometheus.MustNewConstMetric(transcodedMedia, prometheus.GaugeValue, float64(currentMetrics.TranscodedMedia))

	ch <- prometheus.MustNewConstMetric(mediaStreams, prometheus.GaugeValue, float64(currentMetrics.MediaKernel), "kernel")
	ch <- prometheus.MustNewConstMetric(mediaStreams, prometheus.GaugeValue, float64(currentMetrics.MediaUserspace), "userspace")
	ch <- prometheus.MustNewConstMetric(mediaStreams, prometheus.GaugeValue, float64(currentMetrics.MediaMixed), "mixed")
}

func setVoipMetrics(ch chan<- prometheus.Metric, voipMetrics ng.VoipMetrics) {
	ch <- prometheus.MustNewConstMetric(jitterTotal, prometheus.CounterValue, float64(voipMetrics.JitterTotal))
	ch <- prometheus.MustNewConstMetric(jitter2Total, prometheus.CounterValue, float64(voipMetrics.Jitter2Total))
	ch <- prometheus.MustNewConstMetric(jitterSamplesTotal, prometheus.CounterValue, float64(voipMetrics.JitterSamplesTotal))
	ch <- prometheus.MustNewConstMetric(jitterAverage, prometheus.CounterValue, float64(voipMetrics.JitterAverage))
	ch <- prometheus.MustNewConstMetric(jitterStddev, prometheus.CounterValue, float64(voipMetrics.JitterStddev))
	ch <- prometheus.MustNewConstMetric(rttE2ETotal, prometheus.CounterValue, float64(voipMetrics.RttE2ETotal))
	ch <- prometheus.MustNewConstMetric(rttE2E2Total, prometheus.CounterValue, float64(voipMetrics.RttE2E2Total))
	ch <- prometheus.MustNewConstMetric(rttE2ESamplesTotal, prometheus.CounterValue, float64(voipMetrics.RttE2ESamplesTotal))
	ch <- prometheus.MustNewConstMetric(rttE2EAverage, prometheus.CounterValue, float64(voipMetrics.RttE2EAverage))
	ch <- prometheus.MustNewConstMetric(rttE2EStddev, prometheus.CounterValue, float64(voipMetrics.RttE2EStddev))
	ch <- prometheus.MustNewConstMetric(rttDsctTotal, prometheus.CounterValue, float64(voipMetrics.RttDsctTotal))
	ch <- prometheus.MustNewConstMetric(rttDsct2Total, prometheus.CounterValue, float64(voipMetrics.RttDsct2Total))
	ch <- prometheus.MustNewConstMetric(rttDsctSamplesTotal, prometheus.CounterValue, float64(voipMetrics.RttDsctSamplesTotal))
	ch <- prometheus.MustNewConstMetric(rttDsctAverage, prometheus.CounterValue, float64(voipMetrics.RttDsctAverage))
	ch <- prometheus.MustNewConstMetric(rttDsctStddev, prometheus.CounterValue, float64(voipMetrics.RttDsctStddev))
	ch <- prometheus.MustNewConstMetric(packetLossTotal, prometheus.CounterValue, float64(voipMetrics.PacketLossTotal))
	ch <- prometheus.MustNewConstMetric(packetLoss2Total, prometheus.CounterValue, float64(voipMetrics.PacketLoss2Total))
	ch <- prometheus.MustNewConstMetric(packetLossSamplesTotal, prometheus.CounterValue, float64(voipMetrics.PacketLossSamplesTotal))
	ch <- prometheus.MustNewConstMetric(packetLossAverage, prometheus.CounterValue, float64(voipMetrics.PacketLossAverage))
	ch <- prometheus.MustNewConstMetric(packetLossStddev, prometheus.CounterValue, float64(voipMetrics.PacketLossStddev))
	ch <- prometheus.MustNewConstMetric(jitterMeasuredTotal, prometheus.CounterValue, float64(voipMetrics.JitterMeasuredTotal))
	ch <- prometheus.MustNewConstMetric(jitterMeasured2Total, prometheus.CounterValue, float64(voipMetrics.JitterMeasured2Total))
	ch <- prometheus.MustNewConstMetric(jitterMeasuredSamplesTotal, prometheus.CounterValue, float64(voipMetrics.JitterMeasuredSamplesTotal))
	ch <- prometheus.MustNewConstMetric(jitterMeasuredAverage, prometheus.CounterValue, float64(voipMetrics.JitterMeasuredAverage))
	ch <- prometheus.MustNewConstMetric(jitterMeasuredStddev, prometheus.CounterValue, float64(voipMetrics.JitterMeasuredStddev))
	ch <- prometheus.MustNewConstMetric(packetsLost, prometheus.CounterValue, float64(voipMetrics.PacketsLost))

	ch <- prometheus.MustNewConstMetric(rtpDuplicates, prometheus.CounterValue, float64(voipMetrics.RtpDuplicates))
	ch <- prometheus.MustNewConstMetric(rtpSkips, prometheus.CounterValue, float64(voipMetrics.RtpSkips))
	ch <- prometheus.MustNewConstMetric(rtpSeqResets, prometheus.CounterValue, float64(voipMetrics.RtpSeqResets))
	ch <- prometheus.MustNewConstMetric(rtpReordered, prometheus.CounterValue, float64(voipMetrics.RtpReordered))
}

func setMosMetrics(ch chan<- prometheus.Metric, mos ng.MosMetrics) {
	ch <- prometheus.MustNewConstMetric(mosTotal, prometheus.CounterValue, mos.MosTotal)
	ch <- prometheus.MustNewConstMetric(mos2Total, prometheus.CounterValue, mos.Mos2Total)
	ch <- prometheus.MustNewConstMetric(mosSamplesTotal, prometheus.CounterValue, float64(mos.MosSamplesTotal))
	ch <- prometheus.MustNewConstMetric(mosAverage, prometheus.CounterValue, mos.MosAverage)
	ch <- prometheus.MustNewConstMetric(mosStddev, prometheus.CounterValue, mos.MosStddev)
}

func setProxyMetrics(ch chan<- prometheus.Metric, proxy ng.ProxyMetrics) {
	proxyName := proxy.Proxy

	ch <- prometheus.MustNewConstMetric(proxyErrorsTotal, prometheus.CounterValue, float64(proxy.ErrorCount), proxyName)

	ch <- prometheus.MustNewConstMetric(proxyRequestsTotal, prometheus.CounterValue, float64(proxy.PingCount), proxyName, "ping")
	ch <- prometheus.MustNewConstMetric(proxyRequestsTotal, prometheus.CounterValue, float64(proxy.OfferCount), proxyName, "offer")
	ch <- prometheus.MustNewConstMetric(proxyRequestsTotal, prometheus.CounterValue, float64(proxy.AnswerCount), proxyName, "answer")
	ch <- prometheus.MustNewConstMetric(proxyRequestsTotal, prometheus.CounterValue, float64(proxy.DeleteCount), proxyName, "delete")
	ch <- prometheus.MustNewConstMetric(proxyRequestsTotal, prometheus.CounterValue, float64(proxy.QueryCount), proxyName, "query")
	ch <- prometheus.MustNewConstMetric(proxyRequestsTotal, prometheus.CounterValue, float64(proxy.ListCount), proxyName, "list")
	ch <- prometheus.MustNewConstMetric(proxyRequestsTotal, prometheus.CounterValue, float64(proxy.StartRecordingCount), proxyName, "start recording")
	ch <- prometheus.MustNewConstMetric(proxyRequestsTotal, prometheus.CounterValue, float64(proxy.StopRecordingCount), proxyName, "stop recording")
	ch <- prometheus.MustNewConstMetric(proxyRequestsTotal, prometheus.CounterValue, float64(proxy.PauseRecordingCount), proxyName, "pause recording")
	ch <- prometheus.MustNewConstMetric(proxyRequestsTotal, prometheus.CounterValue, float64(proxy.StartForwardCount), proxyName, "start forwarding")
	ch <- prometheus.MustNewConstMetric(proxyRequestsTotal, prometheus.CounterValue, float64(proxy.StopForwardCount), proxyName, "stop forwarding")
	ch <- prometheus.MustNewConstMetric(proxyRequestsTotal, prometheus.CounterValue, float64(proxy.BlockDtmfCount), proxyName, "block DTMF")
	ch <- prometheus.MustNewConstMetric(proxyRequestsTotal, prometheus.CounterValue, float64(proxy.UnblockDtmfCount), proxyName, "unblock DTMF")
	ch <- prometheus.MustNewConstMetric(proxyRequestsTotal, prometheus.CounterValue, float64(proxy.PlayDtmfCount), proxyName, "play DTMF")
	ch <- prometheus.MustNewConstMetric(proxyRequestsTotal, prometheus.CounterValue, float64(proxy.BlockMediaCount), proxyName, "block media")
	ch <- prometheus.MustNewConstMetric(proxyRequestsTotal, prometheus.CounterValue, float64(proxy.UnblockMediaCount), proxyName, "unblock media")
	ch <- prometheus.MustNewConstMetric(proxyRequestsTotal, prometheus.CounterValue, float64(proxy.PlayMediaCount), proxyName, "play media")
	ch <- prometheus.MustNewConstMetric(proxyRequestsTotal, prometheus.CounterValue, float64(proxy.StopMediaCount), proxyName, "stop media")
	ch <- prometheus.MustNewConstMetric(proxyRequestsTotal, prometheus.CounterValue, float64(proxy.SilenceMediaCount), proxyName, "silence media")
	ch <- prometheus.MustNewConstMetric(proxyRequestsTotal, prometheus.CounterValue, float64(proxy.UnsilenceMediaCount), proxyName, "unsilence media")
	ch <- prometheus.MustNewConstMetric(proxyRequestsTotal, prometheus.CounterValue, float64(proxy.UnsubCount), proxyName, "unsubscribe")
	ch <- prometheus.MustNewConstMetric(proxyRequestsTotal, prometheus.CounterValue, float64(proxy.SubAnsCount), proxyName, "subscribe answer")
	ch <- prometheus.MustNewConstMetric(proxyRequestsTotal, prometheus.CounterValue, float64(proxy.StatsCount), proxyName, "statistics")
	ch <- prometheus.MustNewConstMetric(proxyRequestsTotal, prometheus.CounterValue, float64(proxy.PubCount), proxyName, "publish")
	ch <- prometheus.MustNewConstMetric(proxyRequestsTotal, prometheus.CounterValue, float64(proxy.SubReqCount), proxyName, "subscribe request")

	ch <- prometheus.MustNewConstMetric(proxyRequestsDuration, prometheus.CounterValue, proxy.PingDuration, proxyName, "ping")
	ch <- prometheus.MustNewConstMetric(proxyRequestsDuration, prometheus.CounterValue, proxy.OfferDuration, proxyName, "offer")
	ch <- prometheus.MustNewConstMetric(proxyRequestsDuration, prometheus.CounterValue, proxy.AnswerDuration, proxyName, "answer")
	ch <- prometheus.MustNewConstMetric(proxyRequestsDuration, prometheus.CounterValue, proxy.DeleteDuration, proxyName, "delete")
	ch <- prometheus.MustNewConstMetric(proxyRequestsDuration, prometheus.CounterValue, proxy.QueryDuration, proxyName, "query")
	ch <- prometheus.MustNewConstMetric(proxyRequestsDuration, prometheus.CounterValue, proxy.ListDuration, proxyName, "list")
	ch <- prometheus.MustNewConstMetric(proxyRequestsDuration, prometheus.CounterValue, proxy.StartRecordingDuration, proxyName, "start recording")
	ch <- prometheus.MustNewConstMetric(proxyRequestsDuration, prometheus.CounterValue, proxy.StopRecordingDuration, proxyName, "stop recording")
	ch <- prometheus.MustNewConstMetric(proxyRequestsDuration, prometheus.CounterValue, proxy.PauseRecordingDuration, proxyName, "pause recording")
	ch <- prometheus.MustNewConstMetric(proxyRequestsDuration, prometheus.CounterValue, proxy.StartForwardDuration, proxyName, "start forwarding")
	ch <- prometheus.MustNewConstMetric(proxyRequestsDuration, prometheus.CounterValue, proxy.StopForwardDuration, proxyName, "stop forwarding")
	ch <- prometheus.MustNewConstMetric(proxyRequestsDuration, prometheus.CounterValue, proxy.BlockDtmfDuration, proxyName, "block DTMF")
	ch <- prometheus.MustNewConstMetric(proxyRequestsDuration, prometheus.CounterValue, proxy.UnblockDtmfDuration, proxyName, "unblock DTMF")
	ch <- prometheus.MustNewConstMetric(proxyRequestsDuration, prometheus.CounterValue, proxy.PlayDtmfDuration, proxyName, "play DTMF")
	ch <- prometheus.MustNewConstMetric(proxyRequestsDuration, prometheus.CounterValue, proxy.BlockMediaDuration, proxyName, "block media")
	ch <- prometheus.MustNewConstMetric(proxyRequestsDuration, prometheus.CounterValue, proxy.UnblockMediaDuration, proxyName, "unblock media")
	ch <- prometheus.MustNewConstMetric(proxyRequestsDuration, prometheus.CounterValue, proxy.PlayMediaDuration, proxyName, "play media")
	ch <- prometheus.MustNewConstMetric(proxyRequestsDuration, prometheus.CounterValue, proxy.StopMediaDuration, proxyName, "stop media")
	ch <- prometheus.MustNewConstMetric(proxyRequestsDuration, prometheus.CounterValue, proxy.SilenceMediaDuration, proxyName, "silence media")
	ch <- prometheus.MustNewConstMetric(proxyRequestsDuration, prometheus.CounterValue, proxy.UnsilenceMediaDuration, proxyName, "unsilence media")
	ch <- prometheus.MustNewConstMetric(proxyRequestsDuration, prometheus.CounterValue, proxy.UnsubDuration, proxyName, "unsubscribe")
	ch <- prometheus.MustNewConstMetric(proxyRequestsDuration, prometheus.CounterValue, proxy.SubAnsDuration, proxyName, "subscribe answer")
	ch <- prometheus.MustNewConstMetric(proxyRequestsDuration, prometheus.CounterValue, proxy.StatsDuration, proxyName, "statistics")
	ch <- prometheus.MustNewConstMetric(proxyRequestsDuration, prometheus.CounterValue, proxy.PubDuration, proxyName, "publish")
	ch <- prometheus.MustNewConstMetric(proxyRequestsDuration, prometheus.CounterValue, proxy.SubReqDuration, proxyName, "subscribe request")
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
