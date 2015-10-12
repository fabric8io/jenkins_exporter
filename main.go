// Copyright 2015
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"flag"
	"net/http"
	"sync"
	"time"

	"github.com/yosida95/golang-jenkins"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/log"
)

// Interval how often to poll Jenkins
const jenkinsPollInterval = 30 * time.Second

var (
	webAddress       = flag.String("web.listen-address", ":9103", "Address on which to expose metrics and web interface.")
	metricsPath      = flag.String("web.telemetry-path", "/metrics", "Path under which to expose Prometheus metrics.")
	jenkinsUrl       = flag.String("jenkins.url","","URL to Jenkins API")
	jenkinsUser      = flag.String("jenkins.user","","User name for Jenkins")
	jenkinsToken     = flag.String("jenkins.token","","API token for Jenkins access")
	lastFetch        = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "jenkins_last_fetch_timestamp_seconds",
			Help: "Unix timestamp of the last fetched jenkins metrics.",
		},
	)
)

// Single metrics entry used in local metrics cache
type metricsStoreEntry struct {
	metric prometheus.Metric
	timestamp time.Time
}

// Collector object caching values obtained from periodically
// querying Jenkins
type jenkinsCollector struct {
	metricsStore []metricsStoreEntry
	mu           *sync.Mutex
}

// Create an (async) collector for periodically fetching jenkins metrics
func newJenkinsMetricsCollector() *jenkinsCollector {
	c := &jenkinsCollector{
		metricsStore: make([]metricsStoreEntry,20),
		mu:         &sync.Mutex{},
	}
	go c.fetchJenkinsMetrics()
	return c
}

func (c *jenkinsCollector) fetchJenkinsMetrics() {
	ticker := time.NewTicker(jenkinsPollInterval).C
	auth := &gojenkins.Auth {
		ApiToken: *jenkinsToken,
		Username: *jenkinsUser,
	}
	// Jenkins client object
	jenkins := gojenkins.NewJenkins(auth, *jenkinsUrl)
	for {
		select {
		case <-ticker:

		    // TODO:
		    // - Fetch Jenkins Jobs list --> jenkins.GetJobs()
		    // - Foreach Job 'job' :
		    //   - Fetch last 5 builds  --> job.LastCompletetedBuild.Number - 5 ... job.LastCompletetedBuild.Number
		    //   - Foreach Build 'build' :
		    //     - Create prometheus.Metrics from build.duration etc
		    //       with prometheus.NewConstMetric(desc, valueType, value)
		    //     - Create metricsStoreEntry from current timestamp & metrics
		    //     - Append metricsStoreEntry to metricsStore

			jobs, err := jenkins.GetJobs()
			log.Printf("Jobs: %v, jobs: %v", jobs, err)

		    // Garbage collect expired value lists.
			now := time.Now()
			c.mu.Lock()

		    // TODO: Clean up entries older than a certain timestamp
		    // Maybe change metricStore to a queue data structure to easily find out how far to cleanup
			log.Printf("GC at %v", now)
			c.mu.Unlock()
		}
	}
}

// =====================================================================================
// Called when prometheus passes by for scraping:

// Collect implements prometheus.Collector.
func (c jenkinsCollector) Collect(ch chan<- prometheus.Metric) {
	ch <- lastFetch

	c.mu.Lock()
	for _, entry := range c.metricsStore {
		ch <- entry.metric;
	}
	c.mu.Unlock()
}

// Describe implements prometheus.Collector.
func (c jenkinsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- lastFetch.Desc()
}

func main() {
	flag.Parse()

	c := newJenkinsMetricsCollector()
	prometheus.MustRegister(c)

	http.Handle(*metricsPath, prometheus.Handler())

	log.Infof("Starting Server: %s", *webAddress)
	http.ListenAndServe(*webAddress, nil)
}
