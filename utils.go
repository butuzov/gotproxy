package main

/*

General outline of how this all works.

Service(Producer 1) \                           / VerifyProxy(Ip:Port)?OK \
Service(Producer 2) - -> VerificationPipeline --  VerifyProxy(Ip:Port)?OK - -> ProxyCollectionPipeline
Service(Producer 3) /                           \ VerifyProxy(Ip:Port)?OK /

SaveResults triggered if terminated/panics/finishes.
*/

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/url"
	"sync"
	"time"

	"github.com/butuzov/gotproxy/client"
)

// VerificationPipeline is a part of pipeline to validate incoming proxy servers
// it lunches validation in new goroutine.
func VerificationPipeline(wg *sync.WaitGroup, in, out chan url.URL) {
	var limit = make(chan struct{}, 14)
	for ip := range in {
		limit <- struct{}{}
		go VerifyProxy(wg, ip, limit, out)
	}
}

// CollectionPipeline is simple reading URL from channel and load/save into sync map
func CollectionPipeline(in chan url.URL, res *sync.Map) {
	for ip := range in {
		log.Printf("Proxy Validated: %s\n", ip.Host)
		_, _ = res.LoadOrStore(ip, struct{}{})
	}
}

// SaveResults need to be runned in 3 scenarios in order to save what's got now.
func SaveResults(f string, res *sync.Map) {

	var results []string

	res.Range(func(key, value interface{}) bool {
		results = append(results, key.(url.URL).Host)
		return true
	})

	if content, err := json.Marshal(results); err != nil {
		log.Printf("Error in encoding values to json list: %s\n", err)
	} else {
		ioutil.WriteFile(f, content, 0644)
	}
}

// RegisterSource is a small alternative to context package, we use it only to track
// if producer finished it work.
func RegisterSource(wg *sync.WaitGroup, liveness chan struct{}) {
	wg.Add(1)
	go func(wg *sync.WaitGroup, liveness chan struct{}) {
		defer wg.Done()
		select {
		case <-time.After(2 * time.Second):
			return
		case <-liveness:
			return
		}
	}(wg, liveness)
}

// New is url.URL producer.
// TODO: may be write validation for ip:port format ?
func New(ip string) (url.URL, error) {
	return url.URL{Host: ip}, nil
}

// VerifyProxy is trying to reach httpbin and check is we really validated.
func VerifyProxy(wg *sync.WaitGroup, proxy url.URL, limit chan struct{}, out chan url.URL) {
	defer func() { wg.Done() }()
	defer func() { <-limit }()

	var body []byte
	var err error

	if body, err = client.New(20 * time.Second).Proxy(proxy).Read("https://httpbin.org/ip"); err != nil {
		log.Printf("Failed to read response for proxy %s\n", proxy.Hostname())
		return
	}

	var data struct {
		Origin string `json:"origin"`
	}

	if err = json.Unmarshal(body, &data); err != nil {
		log.Printf("Error reading response: %v\n", err)
		return
	} else if data.Origin != proxy.Hostname() {
		log.Printf("IP missmatch %s vs %s\n", proxy.Hostname(), data.Origin)
		return
	}

	out <- proxy
}

// Service is a boilerplate for consuming new IPs (as url.URL) from fnc producer while
// incrementing waitgroup for IPs
func Service(wg *sync.WaitGroup, consumer chan url.URL, fnc func() []string) (live chan struct{}) {

	// var live = make(chan struct{})

	go func(signal chan struct{}, wg *sync.WaitGroup) {
		for _, ip := range fnc() {
			if value, err := New(ip); err == nil {
				wg.Add(1)
				consumer <- value
			}
		}
		signal <- struct{}{}
	}(live, wg)

	return live
}
