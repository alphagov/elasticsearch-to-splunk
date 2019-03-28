package main

import (
	"log"

	"github.com/willf/bloom"
)

type Deduplicator struct {
	Destination chan []byte
	Source      chan []byte
}

func (d *Deduplicator) Deduplicate() {
	log.Println("Deduplicator: Start")

	seenLogs := bloom.New(1024*42, 9)

	for {
		select {
		case msg := <-d.Source:
			if !seenLogs.Test(msg) {
				d.Destination <- msg
				seenLogs.Add(msg)
			} else {
				log.Println("Deduplicator: Deduplicated a log")
			}
		}
	}
}
