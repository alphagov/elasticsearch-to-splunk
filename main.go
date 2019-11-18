package main

import (
	"sync"

	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	elasticsearchURL = kingpin.Flag(
		"es-url",
		"Elasticsearch URL",
	).Required().Envar("ES_URL").String()

	logitAPIKey = kingpin.Flag(
		"logit-api-key",
		"Logit Elasticsearch API Key",
	).Envar("LOGIT_API_KEY").String()

	basicAuthUsername = kingpin.Flag(
		"basic-auth-username",
		"Username for HTTP basic auth",
	).Envar("BASIC_AUTH_USERNAME").String()

	basicAuthPassword = kingpin.Flag(
		"basic-auth-password",
		"Password for HTTP basic auth",
	).Envar("BASIC_AUTH_PASSWORD").String()

	splunkURL = kingpin.Flag(
		"splunk-url",
		"Splunk URL",
	).Required().Envar("SPLUNK_URL").String()

	splunkKey = kingpin.Flag(
		"splunk-key",
		"Splunk API Key",
	).Required().Envar("SPLUNK_KEY").String()

	searchJson = kingpin.Flag(
		"search-json",
		"JSON with which to query Elasticsearch",
	).Required().Envar("SEARCH_JSON").String()

	searchCadence = kingpin.Flag(
		"search-cadence",
		"Cadence in seconds for how often to check Elasticsearch for logs",
	).Default("15").Envar("SEARCH_CADENCE").Int()
)

func main() {
	kingpin.Parse()

	collectLogs := make(chan []byte, 1024)
	shipLogs := make(chan []byte, 1024)

	collector := Collector{
		ElasticsearchURL: *elasticsearchURL,

		LogitAPIKey: *logitAPIKey,

		BasicAuthUsername: *basicAuthUsername,
		BasicAuthPassword: *basicAuthPassword,

		SearchCadence: *searchCadence,
		SearchJson:    *searchJson,
		Destination:   collectLogs,
	}

	deduplicator := Deduplicator{
		Source:      collectLogs,
		Destination: shipLogs,
	}

	shipper := Shipper{
		Source:    shipLogs,
		SplunkURL: *splunkURL,
		SplunkKey: *splunkKey,
	}

	waiter := sync.WaitGroup{}

	waiter.Add(1)
	go func() {
		deduplicator.Deduplicate()
		waiter.Done()
	}()

	waiter.Add(1)
	go func() {
		shipper.Ship()
		waiter.Done()
	}()

	waiter.Add(1)
	go func() {
		collector.Collect()
		waiter.Done()
	}()

	waiter.Wait()
}
