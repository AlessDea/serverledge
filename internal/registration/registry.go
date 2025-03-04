package registration

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/grussorusso/serverledge/internal/config"
	"github.com/grussorusso/serverledge/utils"
	"github.com/lithammer/shortuuid"
	_ "go.etcd.io/etcd/client/v3"
	clientv3 "go.etcd.io/etcd/client/v3"
	"golang.org/x/net/context"
)

var BASEDIR = "registry"
var TTL = config.GetInt(config.REGISTRATION_TTL, 20) // lease time in Seconds

// getEtcdKey append to a given unique id the logical path depending on the Area.
// If it is called with  an empty string  it returns the base path for the current local Area.
func (r *Registry) getEtcdKey(id string) (key string) {
	return fmt.Sprintf("%s/%s/%s", BASEDIR, r.Area, id)
}

// RegisterToEtcd make a registration to the local Area; etcd put operation is performed
func (r *Registry) RegisterToEtcd(hostport string) (string, error) {
	etcdClient, err := utils.GetEtcdClient()
	if err != nil {
		log.Fatal(UnavailableClientErr)
		return "", UnavailableClientErr
	}

	ctx, _ := context.WithTimeout(context.Background(), 1*time.Second)
	//generate unique identifier
	id := shortuuid.New() + strconv.FormatInt(time.Now().UnixNano(), 10)
	r.Key = r.getEtcdKey(id)
	resp, err := etcdClient.Grant(ctx, int64(TTL))
	if err != nil {
		log.Fatal(err)
		return "", err
	}

	log.Printf("Registration key: %s\n", r.Key)
	// save couple (id, hostport) to the correct Area-dir on etcd
	_, err = etcdClient.Put(ctx, r.Key, hostport, clientv3.WithLease(resp.ID))
	if err != nil {
		log.Fatal(IdRegistrationErr)
		return "", IdRegistrationErr
	}

	_, err = etcdClient.Put(ctx, "mykey/"+r.Key, hostport, clientv3.WithLease(resp.ID))
	if err != nil {
		log.Fatal(IdRegistrationErr)
		return "", IdRegistrationErr
	}

	cancelCtx, _ := context.WithCancel(etcdClient.Ctx())

	// the key id will be kept alive until a fault will occur
	keepAliveCh, err := etcdClient.KeepAlive(cancelCtx, resp.ID)
	if err != nil || keepAliveCh == nil {
		log.Fatal(KeepAliveErr)
		return "", KeepAliveErr
	}

	go func() {
		for range keepAliveCh {
			// eat messages until keep alive channel closes
			//log.Println(alive.ID)
		}
	}()

	return r.Key, nil
}

// GetAll is used to obtain the list of  other server's addresses under a specific local Area
func (r *Registry) GetAll(remotes bool) (map[string]string, error) {
	var baseDir string
	if remotes {
		baseDir = fmt.Sprintf("%s/%s/%s/", BASEDIR, "cloud", r.Area)
	} else {
		baseDir = r.getEtcdKey("")
	}
	ctx, _ := context.WithTimeout(context.Background(), 1*time.Second)
	etcdClient, err := utils.GetEtcdClient()
	if err != nil {
		log.Fatal(UnavailableClientErr)
		return nil, UnavailableClientErr
	}
	//retrieve all url of the other servers under my Area
	resp, err := etcdClient.Get(ctx, baseDir, clientv3.WithPrefix())
	if err != nil {
		return nil, fmt.Errorf("Could not read from etcd: %v", err)
	}

	servers := make(map[string]string)
	for _, s := range resp.Kvs {
		servers[string(s.Key)] = string(s.Value)
		//audit todo delete the next line
		if remotes {
			log.Printf("found remote server at: %s\n", servers[string(s.Key)])
		} else {
			log.Printf("found edge server at: %s\n", servers[string(s.Key)])
		}
	}

	return servers, nil
}

// GetCloudNodes retrieves the list of Cloud servers in a given region
func GetCloudNodes(region string) (map[string]string, error) {
	baseDir := fmt.Sprintf("%s/%s/%s/", BASEDIR, "cloud", region)
	ctx, _ := context.WithTimeout(context.Background(), 1*time.Second)
	etcdClient, err := utils.GetEtcdClient()
	if err != nil {
		log.Fatal(UnavailableClientErr)
		return nil, UnavailableClientErr
	}

	resp, err := etcdClient.Get(ctx, baseDir, clientv3.WithPrefix())
	if err != nil {
		return nil, fmt.Errorf("Could not read from etcd: %v", err)
	}

	servers := make(map[string]string)
	for _, s := range resp.Kvs {
		servers[string(s.Key)] = string(s.Value)
	}

	return servers, nil
}

// GetCloudNodesInRegion retrieves the list of Cloud servers in a given region
func GetCloudNodesInRegion(region string) (map[string]string, error) {
	baseDir := fmt.Sprintf("%s/%s/%s/", BASEDIR, "cloud", region)
	ctx, _ := context.WithTimeout(context.Background(), 1*time.Second)
	etcdClient, err := utils.GetEtcdClient()
	if err != nil {
		log.Fatal(UnavailableClientErr)
		return nil, UnavailableClientErr
	}

	resp, err := etcdClient.Get(ctx, baseDir, clientv3.WithPrefix())
	if err != nil {
		return nil, fmt.Errorf("Could not read from etcd: %v", err)
	}

	servers := make(map[string]string)
	for _, s := range resp.Kvs {
		servers[string(s.Key)] = string(s.Value)
	}

	return servers, nil
}

// Deregister deletes from etcd the key, value pair previously inserted
func (r *Registry) Deregister() (e error) {
	etcdClient, err := utils.GetEtcdClient()
	if err != nil {
		log.Fatal(UnavailableClientErr)
		return UnavailableClientErr
	}

	ctx, _ := context.WithTimeout(context.Background(), 1*time.Second)
	_, err = etcdClient.Delete(ctx, r.Key)
	if err != nil {
		return err
	}

	log.Println("Deregister : " + r.Key)
	return nil
}

func (r *Registry) saveServersMapToEtcd() (e error) {
	etcdClient, err := utils.GetEtcdClient()
	if err != nil {
		log.Fatal(UnavailableClientErr)
		return UnavailableClientErr
	}

	data, err := json.Marshal(Reg.serversMap)
	if err != nil {
		return fmt.Errorf("Error serializing serversMap: %s\n", err)
	}

	_, err = etcdClient.Put(context.Background(), "/serverledge/serversMap", string(data))
	if err != nil {
		return fmt.Errorf("Error updating serversMap su Etcd: %s\n", err)
	}
	return nil
}

func GetNodeIDFromEtcd() (string, error) {
	etcdClient, err := utils.GetEtcdClient() // Ottieni il client etcd
	if err != nil {
		log.Fatal("Error connecting to etcd:", err)
		return "", err
	}

	// Prefisso dove abbiamo salvato l'ID
	prefix := "mykey/"

	// Creiamo un contesto con timeout per la richiesta
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Eseguiamo la query per recuperare tutte le chiavi con il prefisso "mykey/"
	resp, err := etcdClient.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		log.Fatal("Error retrieving ID from etcd:", err)
		return "", err
	}

	// Se non troviamo nulla, restituiamo un errore
	if len(resp.Kvs) == 0 {
		log.Println("No node ID found in etcd under", prefix)
		return "", fmt.Errorf("no node ID found")
	}

	// Supponiamo che ci sia un solo nodo registrato, prendiamo il primo valore
	nodeID := string(resp.Kvs[0].Value)
	fmt.Println("Found Node ID:", nodeID)

	return nodeID, nil
}
