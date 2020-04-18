package main

import (
	"log"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

const Exported = "export.json"

func main() {

	var (
		ProxyChan   = make(chan url.URL, 1000)
		ProxyOK     = make(chan url.URL, 1000)
		Results     = &sync.Map{}
		ProducersWG = &sync.WaitGroup{}
		ConsumerWG  = &sync.WaitGroup{}
	)

	// Save mechanics: 1 default regular "save"
	defer SaveResults(Exported, Results)

	// Save mechanics: 2 kill "save"
	var sigsCh = make(chan os.Signal, 1)
	signal.Notify(sigsCh, syscall.SIGINT, syscall.SIGTERM)
	go func(ch chan os.Signal) {
		<-ch
		SaveResults(Exported, Results)
		os.Exit(1)
	}(sigsCh)

	// Save mechanics: 3 emergency "save"
	go func() {
		if err := recover(); err != nil {
			log.Printf("Emergency Save %s", err)
			SaveResults(Exported, Results)
		}
	}()

	// Running Collection and Verification pipelines before spining up producers.
	go CollectionPipeline(ProxyOK, Results)
	go VerificationPipeline(ConsumerWG, ProxyChan, ProxyOK)

	RegisterSource(ProducersWG, Service(ConsumerWG, ProxyChan, FreeProxyList))
	RegisterSource(ProducersWG, Service(ConsumerWG, ProxyChan, UsProxy))
	RegisterSource(ProducersWG, Service(ConsumerWG, ProxyChan, ProxyScrape))

	// Awaiting results.
	ProducersWG.Wait()
	log.Println("Producers Done Producing Proxy List")
	ConsumerWG.Wait()
	log.Println("Consumers Done Validating Proxy List")
	time.Sleep(time.Second)
}
