package main

import (
	"context"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/olivere/elastic"
)

type Collector struct {
	Destination chan []byte

	LogitURL      string
	LogitKey      string
	LogitUser     string
	LogitPassword string
	SearchJson    string
	SearchCadence int

	ElasticsearchClient *elastic.Client
}

type LogitTransport struct {
	apiKey   string
	username string
	password string
}

func (t *LogitTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if t.apiKey != "" {
		r.Header.Set("Apikey", t.apiKey)
	} else {
		r.SetBasicAuth(t.username, t.password)
	}
	return http.DefaultTransport.RoundTrip(r)
}

func (c *Collector) Collect() {
	ticker := time.NewTicker(time.Second * time.Duration(c.SearchCadence))
	defer ticker.Stop()

	httpClient := &http.Client{}
	httpClient.Transport = &LogitTransport{apiKey: c.LogitKey, username: c.LogitUser, password: c.LogitPassword}

	elasticSearch, err := elastic.NewClient(
		elastic.SetURL(c.LogitURL),
		elastic.SetHttpClient(httpClient),
		elastic.SetSniff(false),
	)

	if err != nil {
		log.Fatalf("Collector: could not create elastic client: %s", err)
	}

	c.ElasticsearchClient = elasticSearch

	log.Println("Collector: Start")

	for {
		select {
		case <-ticker.C:
			log.Println("Collector: Tick")

			err := backoff.Retry(
				c.Search,
				backoff.WithMaxRetries(backoff.NewExponentialBackOff(), 10),
			)

			if err != nil {
				log.Fatalf(
					"Collector: Fatal err encountered after 10 retries: %s\n", err,
				)
			}

		}
	}
}

func (c *Collector) Search() error {
	scroll := elastic.NewScrollService(c.ElasticsearchClient).Query(
		elastic.NewRawStringQuery(c.SearchJson),
	).Size(32)

	for {
		results, err := scroll.Do(context.Background())

		if err == io.EOF {
			break
		}

		if err != nil {
			log.Printf("Collector: could not search: %s\n", err)
			return err
		}

		log.Printf("Collector: Found %d results\n", results.TotalHits())

		for _, hit := range results.Hits.Hits {
			msg, err := hit.Source.MarshalJSON()

			if err != nil {
				log.Printf("Collector: could not search: %szn", err)
				return err
			}

			c.Destination <- msg
		}
	}

	return nil
}
