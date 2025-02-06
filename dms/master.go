package dms

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/grussorusso/serverledge/internal/config"
	"github.com/grussorusso/serverledge/internal/registration"
)

var PROTO string = "http://"
var PORT string = ":" + strconv.Itoa(config.GetInt(config.METRICS_EXPORT_PORT, 2112)) //":2112"
var ROUTE string = "/metrics/json"                                                    //TODO: add a route to metrics in order to export the metrics as a json file (chek if the default isn't already like this)

type Node struct {
	Name string `json:"name"`
	IP   string `json:"ip"`
	Type string `json:"type"` // edge or cloud
}

type Metric struct {
	Name  string `json:"name"`
	Value string `json:"value"` // read all the values as a string
}

type result struct {
	valid   bool
	node    Node
	metrics []Metric
}

// maps the nodes with their metrics
var NodesMetricsMap = make(map[Node][]Metric)

// TODO: check for updates of the nodes
func Init(stopChan chan struct{}) {

retry:
	log.Println("DMS: waiting to read registry")
	// create the NodesMetricMap
	registration.Reg.RwMtx.Lock()
	if registration.Reg != nil && len(registration.Reg.NearbyServersMap) != 0 {
		for key, value := range registration.Reg.NearbyServersMap {
			parsedURL, err := url.Parse(value.Url)
			if err != nil {
				registration.Reg.RwMtx.Unlock()
				log.Println("DMS: Error: no valid ip", err)
				return
			}
			var node Node
			if strings.Contains(key, "cloud") {
				//node = Node{key, parsedURL.Hostname(), "cloud"}
				continue // we get metrics only from edge nodes
			} else {
				node = Node{key, parsedURL.Hostname(), "edge"}

			}

			NodesMetricsMap[node] = []Metric{}
			registration.Reg.RwMtx.Unlock()
		}
	} else {
		registration.Reg.RwMtx.Unlock()
		time.Sleep(1 * time.Second)
		goto retry
	}

	retriever(stopChan)

}

func retriever(stopChan chan struct{}) {
	var wg sync.WaitGroup
	for {
		c := make(chan result)
		defer close(c)
		for key, _ := range NodesMetricsMap {
			wg.Add(1)
			go retrieveMetrics(key, c)

		}
		for i := 0; i < len(NodesMetricsMap); i++ {
			m := <-c
			if m.valid {
				NodesMetricsMap[m.node] = m.metrics
			} else {
				log.Println("DMS: Error during metrics retrieve")
			}
		}
		// wg.Wait() // useless because we wait for all the routines writing into the channel
		// time.Sleep(1 * time.Second)
		select {
		case <-stopChan: // Received stop signal
			fmt.Println("Received stop signal")
			return
		}
	}
}

func retrieveMetrics(n Node, ch chan result) {
	var metrics []Metric

	log.Println("DMS: Retrieving metrics")

	// make HTTP request to PROTO + n.IP + PORT + ROUTE
	url := PROTO + n.IP + PORT + ROUTE
	log.Println("DMS: Doing HTTP Get request to", url)
	resp, err := http.Get(url)
	if err != nil {
		log.Println("Error during HTTP request:", err)
		ch <- result{
			valid:   false,
			node:    n,
			metrics: nil,
		}
		return
	}
	defer resp.Body.Close() // close response

	// read response ioutil.ReadAll is deprecated...
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("Error reading response: %v\n", err)
		ch <- result{
			valid:   false,
			node:    n,
			metrics: nil,
		}
		return
	}

	var jsonMetrics map[string]interface{}
	if err := json.Unmarshal(body, &jsonMetrics); err != nil {
		ch <- result{
			valid:   false,
			node:    n,
			metrics: nil,
		}
		return
	}

	// Iterate over the received metrics
	for name, metricData := range jsonMetrics {
		metricMap, ok := metricData.(map[string]interface{})
		if !ok {
			continue
		}

		// Estrai i valori delle metriche
		if metricsArray, exists := metricMap["metric"]; exists {
			for _, m := range metricsArray.([]interface{}) {
				metricInstance := m.(map[string]interface{})
				value := ""
				if counter, hasCounter := metricInstance["counter"]; hasCounter {
					value = fmt.Sprintf("%v", counter.(map[string]interface{})["value"])
				} else if gauge, hasGauge := metricInstance["gauge"]; hasGauge {
					value = fmt.Sprintf("%v", gauge.(map[string]interface{})["value"])
				}

				// Aggiungi la metrica alla lista
				metrics = append(metrics, Metric{
					Name:  name,
					Value: value,
				})
			}
		}
	}

	// log.Println("Metrics for node:", n.Name, n.IP)
	// for _, m := range metrics {
	// 	log.Println(m.Name)
	// }

	ch <- result{
		valid:   true,
		node:    n,
		metrics: metrics,
	}
}
