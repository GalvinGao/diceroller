package main

import "time"

type Deduplicator struct {
	exists map[string]uint32
}

func NewDeduplicator() *Deduplicator {
	d := &Deduplicator{
		exists: make(map[string]uint32),
	}
	go d.GarbageCleanupWorker()

	return d
}

func (d *Deduplicator) GetSet(s string) bool {
	if _, ok := d.exists[s]; ok {
		return false
	} else {
		d.exists[s] = uint32(time.Now().Unix())
		return true
	}
}

func (d *Deduplicator) GarbageCleanupWorker() {
	for {
		time.Sleep(time.Minute * 60 * 24)
		for k, v := range d.exists {
			if v < uint32(time.Now().Unix())-60*60*24 {
				delete(d.exists, k)
			}
		}
	}
}
