package main

import (
	"fmt"
	"log"
	"net/rpc"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/grussorusso/serverledge/dms"
	bully "github.com/grussorusso/serverledge/dms/bully_algorithm"
	"github.com/grussorusso/serverledge/internal/config"
	"github.com/grussorusso/serverledge/internal/node"
	"github.com/grussorusso/serverledge/internal/registration"
	"github.com/grussorusso/serverledge/ms"
)

var ImTheLeader bool
var Reg *registration.Registry

func main() {
	configFileName := ""
	if len(os.Args) > 1 {
		configFileName = os.Args[1]
	}

	config.ReadConfiguration(configFileName)
	var wg sync.WaitGroup
	msUpdate := make(chan bully.NodeInfo)
	bully.ElectionUpdate = make(chan string)
	stopChan := make(chan struct{})
	thisNodeInfo := bully.NodeInfo{}

	// configure reads from etcd (no need to register to an area)
	Reg = new(registration.Registry)
	Reg.Area = config.GetString(config.REGISTRY_AREA, "ROME")

	_, err := Reg.GetAll(true)
	if err != nil {
		log.Fatal(err)
	}

	node.NodeIdentifier, err = registration.GetNodeIDFromEtcd()
	if node.NodeIdentifier == "" {
		log.Println("This node there is not a serverledge node")
	} else {
		log.Printf("This node identifier is %s\n", node.NodeIdentifier)
	}

	// start the monitoring: we need it to discover servers in the area
	err = registration.InitEdgeMonitoring(Reg, false)
	if err != nil {
		log.Fatal(err)
	}

	// start the monitorin system engine
	wg.Add(1)
	go ms.Init(&wg, msUpdate)
	log.Printf("Started the Local Monitoring System engine\n")

	// start the distributed monitorin system engine
	wg.Add(1)

	go func() {
		defer wg.Done()

		// 1. Get the information needed
		// read the configuration: if this node has to be the leader then
		superBully := config.GetBool(config.IM_THE_MASTER, false)
		if superBully {
			thisNodeInfo.SuperBully = true
			// wait for the ms to receive the information needed
			thisNodeInfo = <-msUpdate
		} else {
			thisNodeInfo.SuperBully = false
		}

		wg.Add(1)
		mtx := new(sync.Mutex)
		mtx.Lock()
		go func() {
			for {
				bully.ThisNodeRWMtx.Lock()
				cNodeName, cDist := getCloudNodeDistance(node.NodeIdentifier)
				thisNodeInfo.CloudDist = cDist

				if superBully && thisNodeInfo.CloudDist == -1 {
					// this node can't be the master, it has not a cloud node as neighbour
					superBully = false
				}
				bully.ThisNodeRWMtx.Unlock()
				if len(Reg.NearbyServersMap) <= 0 || cNodeName == "" {
					log.Println("No servers in the area")
				} else {
					log.Println("Found servers in the area: start the algorithm")
					log.Printf("Nearest Cloud node:%s\n", cNodeName)
					mtx.Unlock()
				}
				time.Sleep(30 * time.Second)
			}
		}()

		// 2. Start the algorithm
		// only if there are nodes in the area
		mtx.Lock()
		log.Println("Starting the algorithm")

		bullyNode := bully.NewBullyNode(node.NodeIdentifier)
		bullyNode.Info = thisNodeInfo

		listener, err := bullyNode.NewListener()
		if err != nil {
			log.Println(err)
		}
		defer listener.Close()

		rpcServer := rpc.NewServer()
		rpcServer.Register(bullyNode)

		go rpcServer.Accept(listener)

		bullyNode.ConnectToPeers()
		log.Println("%s is aware of own peers %s", bullyNode.ID, bullyNode.Peers.ToIDs())

		warmupTime := 5 * time.Second
		time.Sleep(warmupTime)

		bullyNode.Elect(bully.ElectionUpdate)
		leader := <-bully.ElectionUpdate
		if strings.Compare(leader, node.NodeIdentifier) == 0 {
			// this node is the master
			ImTheLeader = true
			log.Println("This node is the leader:", leader)
			go dms.Init(stopChan)
		}

		// wait for updates from monitoring system and election algorithm
		for {
			select {
			case update, ok := <-msUpdate:
				if !ok {
					fmt.Println("Closed Channel")
					break
				}
				if !superBully {
					fmt.Println("received:", update)
					thisNodeInfo = update
				}
				break
			case leader := <-bully.ElectionUpdate:
				if strings.Compare(leader, node.NodeIdentifier) == 0 {
					// this node is the new leader
					log.Println("This node is the leader:", leader)
					if ImTheLeader {
						// still the leader: do nothing
					} else {
						ImTheLeader = true
						dms.Init(stopChan)
					}
				} else {
					log.Println("New leader:", leader)
					if ImTheLeader {
						// stop master routine
						ImTheLeader = false
						close(stopChan) // stop the master (leader) routine
					} else {
						// do nothing
					}
				}
				break
			default:
				fmt.Println("No updates from MS")
				time.Sleep(3 * time.Second) // wait 3s
				break
			}
		}

		// c := make(chan os.Signal, 1)
		// signal.Notify(c, os.Interrupt)
		// <-c

	}()

	wg.Wait()

}

func getCloudNodeDistance(s string) (string, time.Duration) {
	Reg.RwMtx.Lock()
	defer Reg.RwMtx.Unlock()

	// check if there is a cloud node in the Area of this node
	if len(Reg.NearbyServersMap) <= 0 {
		return "", -1
	}

	var max time.Duration = 0.0
	var maxNodeID string = ""
	for key, value := range Reg.NearbyServersMap {
		if key != node.NodeIdentifier {
			log.Println("Server:", key)
			bully.Add(key, value.Url)
		}
		if !strings.Contains(key, "cloud") {
			continue
		}
		if Reg.Client.DistanceTo(&value.Coordinates) > max {
			max = Reg.Client.DistanceTo(&value.Coordinates)
			maxNodeID = key
		}
	}

	if maxNodeID != "" {
		return maxNodeID, max
	}
	return "", -1

}
