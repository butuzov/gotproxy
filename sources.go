package main

import (
	"gotproxy/client"
	"log"
	"regexp"
	"strings"
	"time"
)

// hideMyIp is utility method used for two websites developed by same producer,
// so there is actually a reason to use DRY while operating with a products of
// Hidemyip.cm company. https://www.hide-my-ip.com/
func hideMyIP(URL string) []string {

	hmiRegExp := regexp.MustCompile(`<td>([0-9]{2,}.[0-9]{2,}.[0-9]{1,}.[0-9]{1,})</td><td>([0-9]{2,})</td><td>([A-Z]{2})</td>`)

	var (
		body    []byte
		err     error
		results = make([]string, 0, 256)
		client  = client.New(5 * time.Second)
	)

	if body, err = client.Read(URL); err != nil {
		log.Printf("Error Requesting: %v\n", err)
		return results
	}

	for _, m := range hmiRegExp.FindAllSubmatch(body, -1) {
		results = append(results, string(m[1])+":"+string(m[2]))
	}
	return results
}

func FreeProxyList() []string {
	return hideMyIP("https://free-proxy-list.net")
}

func UsProxy() []string {
	return hideMyIP("https://us-proxy.org")
}

// ProxyScrape data mostly useless.
func ProxyScrape() (results []string) {
	var (
		domain = "https://api.proxyscrape.com/?request=getproxies&proxytype=http&timeout=10000"
		err    error
		body   []byte
	)

	if body, err = client.New(10 * time.Second).Read(domain); err != nil {
		log.Printf("Failed to read response for source website %s\n")
		return
	}

	for _, proxy := range strings.Split(string(body), "\n") {
		if strings.Trim(proxy, "\n") != "" {
			results = append(results, strings.Trim(proxy, "\n"))
		}
	}
	return
}
