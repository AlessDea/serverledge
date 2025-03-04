package registration

import (
	"log"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/grussorusso/serverledge/internal/config"
	"github.com/hexablock/vivaldi"
)

var Reg *Registry

func InitEdgeMonitoring(r *Registry, sedgeNode bool) (e error) {
	Reg = r
	defaultConfig := vivaldi.DefaultConfig()
	defaultConfig.Dimensionality = 3

	client, err := vivaldi.NewClient(defaultConfig)
	if err != nil {
		log.Fatal(err)
		return err
	}
	Reg.Client = client
	Reg.etcdCh = make(chan bool)
	Reg.serversMap = make(map[string]*StatusInformation)
	Reg.NearbyServersMap = make(map[string]*StatusInformation)

	// start listening for incoming udp connections; use case: edge-nodes request for status infos
	if sedgeNode {
		go UDPStatusServer()
	}
	//complete monitoring phase at startup
	monitoring()
	go runMonitor()
	return nil
}

func runMonitor() {
	//todo  adjust default values
	nearbyTicker := time.NewTicker(time.Duration(config.GetInt(config.REG_NEARBY_INTERVAL, 20)) * time.Second)         //wake-up nearby monitoring
	monitoringTicker := time.NewTicker(time.Duration(config.GetInt(config.REG_MONITORING_INTERVAL, 30)) * time.Second) // wake-up general-area monitoring
	for {
		select {
		case <-Reg.etcdCh:
			monitoring()
		case <-monitoringTicker.C:
			monitoring()
		case <-nearbyTicker.C:
			nearbyMonitoring()
		}
	}
}

func mergeMaps(map1, map2 map[string]string) map[string]string {
	merged := make(map[string]string)

	// Copia gli elementi della prima mappa
	for k, v := range map1 {
		merged[k] = v
	}

	// Copia gli elementi della seconda mappa (sovrascrive i duplicati)
	for k, v := range map2 {
		merged[k] = v
	}

	return merged
}

func monitoring() {
	Reg.RwMtx.Lock()
	defer Reg.RwMtx.Unlock()

	// gets info from Etcd about other nodes
	// TODO: check that nodes are filtered by geo zone
	etcdServerMap, err := Reg.GetAll(true)
	if err != nil {
		log.Println(err)
		return
	}

	etcdServerMapBis, err := Reg.GetAll(false)
	if err != nil {
		log.Println(err)
		return
	}

	etcdServerMap = mergeMaps(etcdServerMap, etcdServerMapBis)

	delete(etcdServerMap, Reg.Key) // not consider myself

	for key, url := range etcdServerMap {
		if strings.Contains(key, "cloud") {
			continue
		}
		oldInfo, ok := Reg.serversMap[key]

		ip := url[7 : len(url)-5]
		// use udp socket to retrieve infos about the edge-node status and rtt
		newInfo, rtt := statusInfoRequest(ip)
		if newInfo == nil {
			//unreachable server
			log.Printf("Unreachable %v\n", key)
			delete(Reg.serversMap, key)
			continue
		}

		Reg.serversMap[key] = newInfo
		if (ok && !reflect.DeepEqual(oldInfo.Coordinates, newInfo.Coordinates)) || !ok {
			_, err := Reg.Client.Update("node", &newInfo.Coordinates, rtt)
			if err != nil {
				log.Printf("Error while updating node coordinates: %s\n", err)
				return
			}
		}
	}
	//deletes information about servers that haven't registered anymore
	for key := range Reg.serversMap {
		_, ok := etcdServerMap[key]
		if !ok {
			delete(Reg.serversMap, key)
		}
	}

	// Updates NearbyServersMap with the N closest nodes from serverMap
	getRank(10) //todo change this value
	log.Printf("Nearby map at the end of monitoring: %v\n", Reg.NearbyServersMap)
}

type dist struct {
	key      string
	distance time.Duration
}

// getRank finds servers nearby to the current one
func getRank(rank int) {
	if rank > len(Reg.serversMap) {
		Reg.NearbyServersMap = make(map[string]*StatusInformation)
		for k, v := range Reg.serversMap {
			Reg.NearbyServersMap[k] = v
		}
		return
	}

	var distanceBuf = make([]dist, len(Reg.serversMap)) //distances from current server
	for key, s := range Reg.serversMap {
		distanceBuf = append(distanceBuf, dist{key, Reg.Client.DistanceTo(&s.Coordinates)})
	}
	sort.Slice(distanceBuf, func(i, j int) bool { return distanceBuf[i].distance < distanceBuf[j].distance })
	Reg.NearbyServersMap = make(map[string]*StatusInformation)
	for i := 0; i < rank; i++ {
		k := distanceBuf[i].key
		Reg.NearbyServersMap[k] = Reg.serversMap[k]
	}
}

// nearbyMonitoring check nearby server's status
func nearbyMonitoring() {
	log.Printf("Periodic nearby monitoring\n")

	Reg.RwMtx.Lock()
	defer Reg.RwMtx.Unlock()
	for key, info := range Reg.NearbyServersMap {
		oldInfo, ok := Reg.serversMap[key]

		ip := info.Url[7 : len(info.Url)-5]
		newInfo, rtt := statusInfoRequest(ip)

		if newInfo == nil {
			log.Printf("Unreachable neighbor: %s\n", key)
			//unreachable server
			delete(Reg.serversMap, key)
			//trigger a complete monitoring phase
			go func() { Reg.etcdCh <- true }()
			return
		}
		Reg.serversMap[key] = newInfo
		if (ok && !reflect.DeepEqual(oldInfo.Coordinates, newInfo.Coordinates)) || !ok {
			_, err := Reg.Client.Update("node", &newInfo.Coordinates, rtt)
			if err != nil {
				log.Printf("Error while updating node coordinates: %s\n", err)
			}
		}
	}
}
