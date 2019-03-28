package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/cenkalti/backoff"
	"github.com/gojektech/heimdall/httpclient"
)

type Shipper struct {
	Source chan []byte

	SplunkURL string
	SplunkKey string
}

type SplunkResponse struct {
	Text string `json:"text"`
	Code int    `json:"code"`
}

type SplunkHTTPClient struct {
	client    http.Client
	SplunkKey string
}

func (c *SplunkHTTPClient) Do(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", fmt.Sprintf("Splunk %s", c.SplunkKey))
	req.Header.Set("Content-Type", "application/json")
	return c.client.Do(req)
}

func (s *Shipper) Ship() {
	log.Println("Shipper: Start")

	splunkClient := httpclient.NewClient(
		httpclient.WithHTTPClient(
			&SplunkHTTPClient{
				client:    *http.DefaultClient,
				SplunkKey: s.SplunkKey,
			},
		),
	)

	for {
		select {
		case msg := <-s.Source:

			splunkMsg := fmt.Sprintf(
				`{"source": "logit-to-splunk", "event": %s}`,
				msg,
			)

			shipLog := func() error {
				log.Println("Shipper: shipping log")

				res, err := splunkClient.Post(
					s.SplunkURL,
					bytes.NewReader([]byte(splunkMsg)),
					http.Header{},
				)

				if err != nil {
					log.Printf("Shipper: errored shipping log:%s\n", err)
					return err
				}

				if 200 <= res.StatusCode && res.StatusCode < 300 {
					log.Printf("Shipper: shipped log: %s\n", splunkMsg)
					return nil
				}

				body, err := ioutil.ReadAll(res.Body)

				if err != nil {
					log.Printf("Shipper: could not read body: %s\n", err)
					return err
				}

				log.Printf(
					"Shipper: received non-200 status code %d\n%s",
					res.StatusCode,
					string(body),
				)

				return fmt.Errorf("HTTP NOT OKAY")
			}

			err := backoff.Retry(
				shipLog,
				backoff.WithMaxRetries(backoff.NewExponentialBackOff(), 10),
			)

			if err != nil {
				log.Fatalf(
					"Shipper: Fatal err encountered after 10 retries: %s\n", err,
				)
			}

		}
	}
}
