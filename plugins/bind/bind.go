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
	"time"
	"github.com/influxdb/telegraf/plugins"
	"launchpad.net/xmlpath"
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
var xp_req = xmlpath.MustCompile("/isc/bind/statistics/server/requests/opcode")
var xp_qry = xmlpath.MustCompile("/isc/bind/statistics/server/queries-in/rdtype")

func (n *Bind) gatherUrl(addr *url.URL, acc plugins.Accumulator) error {
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


	it := xp_req.Iter(doc)
	ok := it.Next()

	for ok == true {
		item := it.Node()
		fmt.Printf("%s=%s", item.name, item.counter)
		ok = it.Next()

	}

 //  fmt.Printf("%s", doc)



	return nil
}




func init() {
	plugins.Add("bind", func() plugins.Plugin {
		return &Bind{}
	})
}
