# logit-to-splunk

Go utility to ship logs from Logit to Splunk.

## Usage

### Source

Use `--logit-es-url` or the `LOGIT_ES_URL` environment variable to configure
the Logit Elasticsearch source. Should be in the format
`https://ffffffff-ffff-ffff-ffff-ffffffffffff-es.logit.io`.

Use `--logit-es-key` or the `LOGIT_ES_KEY` environment variable to configure
the Logit Elasticsearch API key. Should be in the format
`ffffffff-ffff-ffff-ffff-ffffffffffff`.

These variables can be found on the Elasticsearch page within a Logit stack's
settings. 

### Destination

Use `--splunk-url` or the `SPLUNK_URL` environment variable to configure
the Splunk destination. Should be in the format
`https://instance-name.splunkcloud.com:443/services/collector`.

Use `--splunk-key` or the `SPLUNK_KEY` environment variable to configure the
API key used when sending logs to Splunk.

### Content

Use `--search-json` or the `SEARCH_JSON` environment variable to configure the
query used in Elasticsearch. For instance if your Elasticsearch query is:

```
{
  "query": {
    "bool": {
      "must": [{
        "exists": { "field": "message" }
      }, {
        "range": {
          "@timestamp": { "gte" : "now-2m/d" }
        }
      }]
    }
  },
  "sort": [
    {
      "@timestamp": {
        "order": "desc"
      }
    }
  ]
}
```

then you would use:

```
--search-json '{
  "bool": {
    "must": [{
      "exists":{
        "field": "message"
      }
    }, {
      "range": {
        "@timestamp": {
          "gte": "now-2m/d"
        }
      }
    }
  ]}
}'
```

i.e. without the `query` object wrapping the query.  You can compact the json
by using `jq -c`.

It is important you specify a range otherwise you will retrieve all documents.

### Cadence

Use `--search-cadence` or the `SEARCH_CADENCE` environment variable to specify
how many seconds to wait in-between queries to Elasticsearch. Defaults to 15.

## Deduplication and tuning

You should specify a `range` in your Elasticsearch query, e.g. in the format
`now-2m/d` to get all logs in the last two minutes.

Logs are deduplicated probabilistically using a [bloom
filter](https://en.wikipedia.org/wiki/Bloom_filter) using the entire content of
the document returned by Elasticsearch.  This means you could set
`--search-cadence` to `60`, logs which have already been seen will not be
shipped to Splunk.
