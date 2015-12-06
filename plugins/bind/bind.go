package bind

import (
/*
	"bufio"
	"strconv"
	"strings"
*/
	"fmt"
	"sync"
	"net/http"
	"net/url"
	"strconv"
	"time"
	"github.com/influxdb/telegraf/plugins"
	"gopkg.in/xmlpath.v1"
	"strings"
)


type Bind struct {
	Urls []string
}

var sampleConfig = `
  # An array of Bind status URI to gather stats from.
  urls = ["http://localhost:8053"]
`
func (n *Bind) SampleConfig() string {
	return sampleConfig
}

func (n *Bind) Description() string {
	return "Read Bind status information, requires statistics-channels to be enabled"
}

func (n *Bind) Gather(acc plugins.Accumulator) error {
	var wg sync.WaitGroup
	var outerr error

	for _, u := range n.Urls {
		addr, err := url.Parse(u)
		if err != nil {
			return fmt.Errorf("Unable to parse address '%s': %s", u, err)
		}

		wg.Add(1)
		go func(addr *url.URL) {
			defer wg.Done()
			outerr = n.gatherUrl(addr, acc)
		}(addr)
	}

	wg.Wait()

	return outerr
}

var tr = &http.Transport {
	ResponseHeaderTimeout: time.Duration(3 * time.Second),
}
var client = &http.Client{Transport: tr}
var xpRequest = xmlpath.MustCompile("/isc/bind/statistics/server/requests/opcode")
var xpQueryDetail = xmlpath.MustCompile("/isc/bind/statistics/server/queries-in/rdtype")

var xpName = xmlpath.MustCompile("./name/text()")
var xpCounter =  xmlpath.MustCompile("./counter/text()")


func (n *Bind) gatherUrl(addr *url.URL, acc plugins.Accumulator) error {
	ts := time.Now()

	tags := map[string]string {
		"server": strings.Split(addr.Host, ":")[0],
	}

	resp, err := client.Get(addr.String())
	if err != nil {
		return fmt.Errorf("error making HTTP request to %s: %s", addr.String(), err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s returned HTTP status %s", addr.String(), resp.Status)
	}

	doc, err := xmlpath.Parse(resp.Body)
	if err != nil {
		return fmt.Errorf("error parsing response as xml: %s", err)
	}

	it := xpRequest.Iter(doc)
	for it.Next() {
		node := it.Node()
		name, _ := xpName.String(node)
		name = strings.ToLower(name)
		value, _ := xpCounter.String(node)
		ival, _ := strconv.ParseUint(value, 10, 64)
		acc.Add(fmt.Sprintf("total_%s", name), ival, tags, ts)
	}

	it = xpQueryDetail.Iter(doc)
	for it.Next() {
		node := it.Node()
		name, _ := xpName.String(node)
		name = strings.ToLower(name)
		value, _ := xpCounter.String(node)
		ival, _ := strconv.ParseUint(value, 10, 64)
		acc.Add(fmt.Sprintf("query_%s", name), ival, tags, ts)
	}

	return nil
}




func init() {
	plugins.Add("bind", func() plugins.Plugin {
		return &Bind{}
	})
}
