package main

import (
	"log"
	"sync"

	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	logitURL = kingpin.Flag(
		"logit-es-url",
		"Logit Elasticsearch URL",
	).Required().Envar("LOGIT_ES_URL").String()

	logitKey = kingpin.Flag(
		"logit-es-key",
		"Logit Elasticsearch API Key",
	).Default("\000").Envar("LOGIT_ES_KEY").String()

	logitUser = kingpin.Flag(
		"logit-es-user",
		"Logit Elasticsearch Username",
	).Default("\000").Envar("LOGIT_ES_USERNAME").String()

	logitPassword = kingpin.Flag(
		"logit-es-password",
		"Logit Elasticsearch Password",
	).Default("\000").Envar("LOGIT_ES_PASSWORD").String()

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

	if *logitKey == "\000" && *logitUser == "\000" {
		log.Fatalf("You must supply logitKey or logitUser")
	}

	collectLogs := make(chan []byte, 1024)
	shipLogs := make(chan []byte, 1024)

	collector := Collector{
		LogitURL:      *logitURL,
		LogitKey:      *logitKey,
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
