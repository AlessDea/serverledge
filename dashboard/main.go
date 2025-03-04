// Backend (Go) per interrogare Etcd e Prometheus
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/rs/cors"

	_ "go.etcd.io/etcd/client/v3"
	clientv3 "go.etcd.io/etcd/client/v3"
)

var etcdClient *clientv3.Client

func initEtcd() {
	var err error
	etcdClient, err = clientv3.New(clientv3.Config{
		Endpoints:   []string{"http://localhost:2379"},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		log.Fatalf("Errore connessione a Etcd: %v", err)
	}
}

func getClusterInfo(w http.ResponseWriter, r *http.Request) {
	resp, err := etcdClient.Get(context.Background(), "/serverledge/nodes", clientv3.WithPrefix())
	if err != nil {
		http.Error(w, "Errore nel recupero dei nodi", http.StatusInternalServerError)
		return
	}
	nodes := make(map[string]string)
	for _, kv := range resp.Kvs {
		nodes[string(kv.Key)] = string(kv.Value)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(nodes)
}

func getMetrics(w http.ResponseWriter, r *http.Request) {
	queries := []string{"node_memory_MemAvailable_bytes", "node_memory_MemFree_bytes", "node_memory_MemTotal_bytes",
		"node_cpu_seconds_total", "namedprocess_namegroup_cpu_seconds_total", "namedprocess_namegroup_memory_bytes",
		"sedge_func_cpu_total", "sedge_func_mem_total", "sedge_exectime", "sedge_completed_total", "sedge_init_time", "sedge_is_cold_start", "sedge_response_time"}
	metrics := make([]map[string]interface{}, 0)

	for _, query := range queries {
		resp, err := http.Get("http://localhost:9090/api/v1/query?query=" + query)
		if err != nil {
			log.Printf("Errore nel recupero della metrica %s: %v", query, err)
			continue
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Errore lettura risposta della metrica %s: %v", query, err)
			continue
		}
		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			log.Printf("Errore parsing JSON della metrica %s: %v", query, err)
			continue
		}
		metrics = append(metrics, result)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
	// resp, err := http.Get("http://localhost:9090/api/v1/query?query=node_memory_MemFree_bytes")
	// if err != nil {
	// 	http.Error(w, "Errore recupero metriche", http.StatusInternalServerError)
	// 	return
	// }
	// defer resp.Body.Close()
	// body, err := io.ReadAll(resp.Body)
	// if err != nil {
	// 	http.Error(w, "Errore lettura risposta", http.StatusInternalServerError)
	// 	return
	// }
	// w.Header().Set("Content-Type", "application/json")
	// w.Write(body)
}

func main() {
	// CORS configuration
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"http://localhost:3000"}, // Aggiungi qui l'URL del tuo client (React/Vue, ecc.)
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE"},
		AllowedHeaders: []string{"Content-Type", "Authorization"},
	})
	initEtcd()
	handler := c.Handler(http.DefaultServeMux)
	http.HandleFunc("/cluster-info", getClusterInfo)
	http.HandleFunc("/metrics", getMetrics)
	fmt.Println("Server avviato su http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", handler))
}
