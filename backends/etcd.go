// The MIT License (MIT)
//
// Copyright (c) 2015 Caleb Champlin (caleb.champlin@gmail.com)
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package backends

import (
	"../cluster"
	"../deployment"
	"../log"
	"encoding/json"
	"github.com/coreos/etcd/client"
	"golang.org/x/net/context"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type EtcdBackend struct {
	etcdConfig      client.Config
	etcdClient      client.Client
	kapi            client.KeysAPI
	backendConfig   EtcdBackendConfig
	deploymentCount int
	mutex           *sync.Mutex
	machine         *cluster.Machine
	cluster         *cluster.Cluster
	machineConfig   string
	nodeListeners   map[string]chan *cluster.Machine
	Status          chan string
	Signal          chan int
}

type EtcdBackendConfig struct {
	Endpoints           []string `json:"endpoints"`
	MachinePrefix       string   `json:"machine-prefix"`
	DeploymentPrefix    string   `json:"deployment-prefix"`
	TTL                 int64    `json:"ttl"`
	FailoverTimeout     int64
	FailoverUnit        time.Duration
	Prefix              string `json:"node-prefix"`
	RecoveryParticipant bool   `json:"recovery-participant"`
}

func (e *EtcdBackend) Init(clstr *cluster.Cluster, m *cluster.Machine) {
	e.mutex = &sync.Mutex{}
	e.parseConfig(clstr, e.backendConfig.Prefix)
	e.cluster = clstr
	e.machine = m
	e.Status = make(chan string, 100)
	e.Signal = make(chan int, 8)
	e.nodeListeners = make(map[string]chan *cluster.Machine)
	e.machineConfig = m.Serialize()
	e.etcdConfig = client.Config{
		Endpoints:               e.backendConfig.Endpoints,
		Transport:               client.DefaultTransport,
		HeaderTimeoutPerRequest: time.Second * 5,
	}

	// Initialize etcd client
	c, err := client.New(e.etcdConfig)
	e.etcdClient = c
	if err != nil {
		log.Error.Printf("Failed to initialize etcd client: %v", err)
		return
	}

	// Create a keys api
	// Are we okay to use this instances for the lifetime
	// of the application? What happens if the etcd
	// instance we are connecting dies?
	e.kapi = client.NewKeysAPI(e.etcdClient)

	// Is quorum necessary to ensure a correct count?
	options := client.GetOptions{Quorum: true}

	val, err := e.kapi.Get(context.Background(), e.backendConfig.MachinePrefix+"/deployments/"+e.machine.Id, &options)
	if err != nil {
		log.Info.Printf("Recieved an error when fetching existing deployment count: %v", err)

		e.deploymentCount = 0
		_, err := e.kapi.Set(context.Background(), e.backendConfig.MachinePrefix+"/deployments/"+e.machine.Id, strconv.Itoa(e.deploymentCount), nil)
		if err != nil {
			handleEtcdError(err, "machine")
		}

	} else {

		// TODO we need to appropriately handle this case
		// Do we need some mechanism for retrieving existing Deployments
		// how do we validate them?
		log.Info.Printf("Recieved existing deployment count: %s", val.Node.Value)
		e.deploymentCount, _ = strconv.Atoi(val.Node.Value)
	}

	// Notify the cluster of this node
	createOptions := client.SetOptions{TTL: (time.Second * time.Duration(e.backendConfig.TTL)), PrevExist: client.PrevNoExist}
	_, err = e.kapi.Set(context.Background(), e.backendConfig.MachinePrefix+"/status/"+e.machine.Id, e.machineConfig, &createOptions)
	if err != nil {
		handleEtcdError(err, "machine")
		log.Error.Printf("Could not join the cluster, another node exists with our machine ID")
		return
	}

	// Start jobs
	e.loadMachines()
	e.keepAlive(e.backendConfig.MachinePrefix+"/status/"+e.machine.Id, e.machineConfig)
	e.monitor()
	e.Status <- "Started"
}

func (e *EtcdBackend) GetValue(key string) map[string]interface{} {
	result, err := e.kapi.Get(context.Background(), key, nil)
	if err != nil {
		log.Warning.Printf("Could not load key value: %s", key)
		return nil
	}
	// Return the first value if it's a list
	if result.Node.Nodes != nil && len(result.Node.Nodes) > 1 {
		nodes := result.Node.Nodes
		for _, node := range nodes {
			var output map[string]interface{}
			if err = json.Unmarshal([]byte(node.Value), &output); err != nil {
				log.Error.Printf("Failed to parse file %s: %v", node.Value, err)
				return nil
			}
			return output
		}
	} else {
		var output map[string]interface{}
		if err = json.Unmarshal([]byte(result.Node.Value), &output); err != nil {
			log.Error.Printf("Failed to parse file %s: %v", result.Node.Value, err)
			return nil
		}
		return output
	}
	return nil
}

func (e *EtcdBackend) GetString(key string) string {
	result, err := e.kapi.Get(context.Background(), key, nil)
	if err != nil {
		log.Warning.Printf("Could not load key value: %s", key)
		return ""
	}
	// Return the first value if it's a list
	if result.Node.Nodes != nil && len(result.Node.Nodes) > 1 {
		nodes := result.Node.Nodes
		for _, node := range nodes {
			return node.Value
		}
	} else {
		return result.Node.Value
	}
	return ""
}

func (e *EtcdBackend) GetValues(key string) map[string]interface{} {
	result, err := e.kapi.Get(context.Background(), key, nil)
	if err != nil {
		log.Warning.Printf("Could not load key value: %s", key)
		return nil
	}
	nodes := result.Node.Nodes
	var results = make(map[string]interface{})
	for _, node := range nodes {
		var output map[string]interface{}
		if err = json.Unmarshal([]byte(node.Value), &output); err != nil {
			log.Error.Printf("Failed to parse file %s: %v", node.Value, err)
			continue
		}
		results[node.Key] = output
	}
	return results
}

func (e *EtcdBackend) loadMachines() {
	result, err := e.kapi.Get(context.Background(), e.backendConfig.MachinePrefix+"/status/", nil)
	if err != nil {
		// TODO we probably need to try again, what happens if this is never successful
		log.Warning.Printf("Could not load cluster machine list")
		return
	}
	nodes := result.Node.Nodes
	for _, node := range nodes {
		//log.Info.Printf("Node: %s = %s ", node.Key, node.Value)
		m := cluster.DeserializeMachine(node.Value)
		if m.Id != e.machine.Id {
			e.cluster.AddMachine(m)
		}
	}
}

// Will run forever settings a TTL key on etcd with this node's info
// Should this key expire another node should catch it in the monitor
// and start the recovery process
func (e *EtcdBackend) keepAlive(key string, value string) {
	go func() {
		retries := 10
		options := client.SetOptions{TTL: (time.Second * time.Duration(e.backendConfig.TTL)), PrevExist: client.PrevExist}
		for {
			_, err := e.kapi.Set(context.Background(), key, value, &options)
			if err != nil {
				handleEtcdError(err, "machine")

				// We sleep for a few seconds and try again
				// TODO should we have a max retry count for this
				// what do we do if this fails forever?
				select {
				case sig := <-e.Signal:
					e.Signal <- sig
					log.Trace.Printf("Recieved shutdown: aborting keepalive")
					e.Status <- "Not Recovering"
					return
				case <-time.After(2 * time.Second):
				}
				retries -= 1
				if retries == 0 {
					return
				}
			} else {
				retries = 10
				// Give ourselves atleast 15 seconds (in properly configured)
				// environments to reset the ttl before it expires
				var duration time.Duration
				if e.backendConfig.TTL > 30 {
					// TODO set this to a const, or configured value
					duration = time.Duration(e.backendConfig.TTL - 15)
				} else {
					duration = time.Duration(e.backendConfig.TTL / 2)
				}
				select {
				case sig := <-e.Signal:
					e.Signal <- sig
					options = client.SetOptions{TTL: (time.Second * 1), PrevExist: client.PrevExist}
					e.kapi.Set(context.Background(), key, value, &options)
					log.Trace.Printf("Recieved shutdown: aborting keepalive")
					e.Status <- "Not Recovering"
					return
				case <-time.After(duration * time.Second):
				}
			}
		}
	}()
}

// Callback for storing a deployment information to etcd
func (e *EtcdBackend) DeploymentComplete(d *deployment.Deployment) {
	go func() {
		// TODO should we be worried about possible serialization errors?
		data, _ := json.Marshal(d)
		_, err := e.kapi.Set(context.Background(), e.backendConfig.DeploymentPrefix+"/"+e.machine.Id+"/"+d.Id, string(data), nil)
		if err != nil {
			// TODO how to handle errors in this case?
			// TODO this should retry atleast 3 times
			handleEtcdError(err, "deployment")
		} else {
			e.IncrementDeploymentCount()
		}
	}()
}

// Noop
func (e *EtcdBackend) DeploymentFailed(d *deployment.Deployment) {

}

// Increment the count, which is used in recovery
// situations
func (e *EtcdBackend) IncrementDeploymentCount() {
	// TODO can this be done without a mutex by relying
	// on the safety machnisms provided by etcd?
	// possibly a compare and set and then increment the
	// local variable?
	e.mutex.Lock()

	deploymentOptions := client.SetOptions{PrevValue: strconv.Itoa(e.deploymentCount), PrevExist: "true"}

	e.deploymentCount += 1

	_, err := e.kapi.Set(context.Background(), e.backendConfig.MachinePrefix+"/deployments/"+e.machine.Id, strconv.Itoa(e.deploymentCount), &deploymentOptions)
	if err != nil {
		// TODO handle this error
		// TODO this should retry atleast 3 times
		handleEtcdError(err, "machine")
	}

	e.mutex.Unlock()
}

// Set up a etcd watch for changes to machine
// statuses, if a key expires we should go into
// recovery procedures. If a machine is added
// we should add it to our internal list
func (e *EtcdBackend) monitor() {
	log.Trace.Printf("Starting machine monitor")

	go func() {
		// Right now etcd is notifying us of any changes to the keys
		// including ttl updates. This is potentially a huge amount of traffic
		// and a huge amount of overhead for etcd to manage
		// Hopefully etcd will eventually provide a mechanism to ignore
		// ttl updates
		options := client.WatcherOptions{Recursive: true}
		watcher := e.kapi.Watcher(e.backendConfig.MachinePrefix+"/status/", &options)
		ctx, cancelFunc := context.WithCancel(context.Background())
		go func() {
			sig := <-e.Signal
			e.Signal <- sig
			cancelFunc()
			return
		}()
		for {
			resp, err := watcher.Next(ctx)
			if err != nil {
				handleEtcdError(err, "watch")
				log.Trace.Printf("Recieved shutdown: aborting monitor")
				e.Status <- "Not Recovering"
				return
			} else {
				//log.Trace.Printf("Watch completed %v", resp)
				switch resp.Action {
				case "expire":
					if e.backendConfig.RecoveryParticipant {
						go e.handleFailure(resp.PrevNode)
					}
				case "create":
					// If a key expired and the machine went downed
					// but then comes back up before recovery is under way
					// we want to cancel the recovery wait period
					go e.handleRecovery(resp.Node)
					go e.handleNewNode(resp.Node)
				}
			}
		}
	}()
}

func (e *EtcdBackend) Watch(key string, callback func(string)) {
	log.Trace.Printf("Starting watch")

	go func() {
		// Right now etcd is notifying us of any changes to the keys
		// including ttl updates. This is potentially a huge amount of traffic
		// and a huge amount of overhead for etcd to manage
		// Hopefully etcd will eventually provide a mechanism to ignore
		// ttl updates
		options := client.WatcherOptions{Recursive: true}
		watcher := e.kapi.Watcher(key, &options)
		ctx, cancelFunc := context.WithCancel(context.Background())
		go func() {
			sig := <-e.Signal
			e.Signal <- sig
			cancelFunc()
			return
		}()
		for {
			resp, err := watcher.Next(ctx)
			if err != nil {
				handleEtcdError(err, "watch")
				log.Trace.Printf("Recieved shutdown: aborting monitor")
				e.Status <- "Not Recovering"
				return
			} else {
				log.Trace.Printf("Watch completed %v", resp)
				if resp.Node != nil {
					callback(resp.Node.Value)
				}
			}
		}
	}()
}

// Updates the channel blocking for recovery
func (e *EtcdBackend) handleRecovery(node *client.Node) {
	m := cluster.DeserializeMachine(node.Value)
	chn, ok := e.nodeListeners[m.Id]
	if ok {
		chn <- m
	}
}

func (e *EtcdBackend) handleFailure(node *client.Node) {
	e.Status <- "Waiting to recover"
	m := cluster.DeserializeMachine(node.Value)
	if m.Id == e.machine.Id {
		log.Error.Printf("Our key expired but we are still alive! %s", m.Id)
		return
	}
	check := e.cluster.GetMachine(m.Id)
	if check == nil {
		log.Error.Printf("Received expiration of a machine we weren't aware of %s", m.Id)
		return
	}
	// Create a listener for this machine to be notified
	// if the machine returns
	// Do we need a buffered channel for this use case?
	log.Info.Printf("Handling machine failure of %v", m)
	listener := make(chan *cluster.Machine, 8)
	e.nodeListeners[m.Id] = listener
	select {
	case <-listener:
		// The machine appears to have recovered, which is great news
		// for us because we don't have to do any work
		e.Status <- "Not Recovering"
		delete(e.nodeListeners, m.Id)
		return
	case sig := <-e.Signal:
		e.Signal <- sig
		log.Trace.Printf("Recieved shutdown: aborting recovery")
		e.Status <- "Not Recovering"
		return
	case <-time.After(e.backendConfig.FailoverUnit * time.Duration(e.backendConfig.FailoverTimeout)):
		// The machine has expired, so we will start the recovery
		// process.
		// We'll start by fighting against every machine
		// in the cluster to obtain a lock, once we do
		// we'll grab all the machines and identify the one
		// with the least deployments.
		// Then we'll go through all of the downed machine's
		// Deployments, and start forwarding them to the
		// machine with the least, failing over to the next
		// least if the previouys machine already has said
		// deployment until all the deployments have been
		// reassigned.
		e.AttemptRecovery(m)
		return
	}
}

func (e *EtcdBackend) AttemptRecovery(m *cluster.Machine) {
	// First double check that the node still doesn't exist
	// this could potentially happen during a race condition
	// where the node goes down and immediately comes back up
	// but our blocking channel hasn't been setup yet so
	// we miss the notification
	e.Status <- "Attempting Recovery"
	log.Info.Printf("Starting recovery for node: %s", m.Id)
	options := client.GetOptions{Quorum: true}
	_, err := e.kapi.Get(context.Background(), e.backendConfig.MachinePrefix+"/status/"+m.Id, &options)
	if err == nil {
		// Looks like the machine popped back up
		delete(e.nodeListeners, m.Id)
		return
	}
	// Try to grab the lock and hope that we aren't the
	// unlucky node to do so.
	setOptions := client.SetOptions{PrevExist: "false"}
	_, err = e.kapi.Set(context.Background(), e.backendConfig.MachinePrefix+"/recovery/"+m.Id, e.machine.Id, &setOptions)
	if err != nil {
		// Another node got the lock, we're done
		// TODO remove the machine from our list
		log.Info.Printf("Could not obtain recovery lock for %s", m.Id)
		delete(e.nodeListeners, m.Id)
		e.Status <- "Not Recovering"
		return
	}
	e.Status <- "Recovering"
	log.Info.Printf("Performing recovery of %s", m.Id)

	// TODO what happens if we get the lock and then crash or get shutdown before recovery has completed?
	// We are the unlucky node :(
	// Next pull a list of all the machines in the cluster
	// and how many deployments they have.
	result, err := e.kapi.Get(context.Background(), e.backendConfig.MachinePrefix+"/deployments/", nil)
	if err != nil {
		// TODO we probably need to try again, what happens if this is never successful
		log.Warning.Printf("Could not load machine deployment counts in recovery")
	}
	nodes := DNodes{result.Node.Nodes}
	sort.Sort(nodes)

	// Pull all the deployments the downed node had
	// Start redeploying those deployments
	result, err = e.kapi.Get(context.Background(), e.backendConfig.DeploymentPrefix+"/"+m.Id, nil)
	if err != nil {
		// TODO we probably need to try again, what happens if this is never successful
		log.Warning.Printf("Could not load deployments in recovery")
		return
	}
	deployments := result.Node.Nodes
	for _, deploymentNode := range deployments {
		var deployment deployment.Deployment
		err = json.Unmarshal([]byte(deploymentNode.Value), &deployment)
		if err != nil {
			log.Error.Printf("Failed to parse json deployment json: %v \"%s\"", err, deploymentNode.Value)
			continue
		}
		cache := make(map[string]*cluster.Machine)
		i := 0
		for {
			node := nodes.Nodes[i]
			i += 1
			log.Info.Printf("Node: %s = %s ", node.Key, node.Value)
			machineName := node.Key[len(e.backendConfig.MachinePrefix+"/deployments/"):]
			// TODO we should be able to deploy to ourselves
			if machineName == e.machine.Id || machineName == m.Id {
				continue
			}
			machine, ok := cache[machineName]
			if !ok {
				machine = e.cluster.GetMachine(machineName)
				if machine == nil {
					log.Error.Printf("Attempted to get info for a machine we aren't aware of: %s", machineName)
					continue
				}
				cache[machineName] = machine
			}
			if ok := machine.TryDeploy(deployment); ok {
				log.Trace.Printf("Deployed: %s to machine %s", deployment.Id, machine.Id)
				curDeployments, _ := strconv.Atoi(node.Value)
				curDeployments += 1
				node.Value = strconv.Itoa(curDeployments)
				x := i
				ii := i - 1
				for {
					nextNode := nodes.Nodes[x]
					nextDeployments, _ := strconv.Atoi(nextNode.Value)
					if nextDeployments >= curDeployments {
						break
					}
					log.Trace.Printf("Swapping: %s with %s", nodes.Nodes[ii].Key, nodes.Nodes[x].Key)
					nodes.Nodes[ii], nodes.Nodes[x] = nodes.Nodes[x], nodes.Nodes[ii]
					x += 1
					ii += 1
				}
				for z := 0; z < len(nodes.Nodes); z++ {
					log.Trace.Printf("Node: %s = %s", nodes.Nodes[z].Key, nodes.Nodes[z].Value)
				}
				break
			}
		}

	}
	e.Status <- "Recovered"

}

type DNodes struct {
	client.Nodes
}

func (ns DNodes) Len() int { return len(ns.Nodes) }
func (ns DNodes) Less(i, j int) bool {
	iVal, _ := strconv.Atoi(ns.Nodes[i].Value)
	jVal, _ := strconv.Atoi(ns.Nodes[j].Value)
	return iVal < jVal
}
func (ns DNodes) Swap(i, j int) { ns.Nodes[i], ns.Nodes[j] = ns.Nodes[j], ns.Nodes[i] }

func (e *EtcdBackend) handleNewNode(node *client.Node) {
	m := cluster.DeserializeMachine(node.Value)
	e.cluster.AddMachine(m)
}

func (e *EtcdBackend) parseConfig(cluster *cluster.Cluster, prefix string) {
	cfg := EtcdBackendConfig{}
	backendConfig := cluster.ClusterConfig["backend-config"].(map[string]interface{})
	for key, val := range backendConfig {
		switch key {
		case "endpoints":
			var endpoints []string
			for _, eval := range val.([]interface{}) {
				endpoints = append(endpoints, eval.(string))
			}

			cfg.Endpoints = endpoints
		case "recovery-participant":
			cfg.RecoveryParticipant = val.(bool)
		case "node-prefix":
			cfg.Prefix = val.(string)
		case "machine-prefix":
			cfg.MachinePrefix = val.(string) + "/" + prefix
		case "deployment-prefix":
			cfg.DeploymentPrefix = val.(string) + "/" + prefix
		case "failover-timeout":
			timespan := val.(string)
			unit := timespan[len(timespan)-1:]
			duration := timespan[:len(timespan)-1]
			switch strings.ToLower(unit) {
			case "s":
				cfg.FailoverUnit = time.Second
			case "m":
				cfg.FailoverUnit = time.Minute
			case "h":
				cfg.FailoverUnit = time.Hour
			// TODO handle undefined unit types
			default:
			}
			// TODO Handle parse failures
			cfg.FailoverTimeout, _ = strconv.ParseInt(duration, 10, 64)
		case "ttl":
			cfg.TTL = int64(val.(float64))
		}
	}
	log.Trace.Printf("Loaded Config: %v", cfg)
	e.backendConfig = cfg
}

func handleEtcdError(err error, key string) {
	if err == context.Canceled {
		log.Error.Printf("Failed to set "+key+" key (context canceled): %v", err)
	} else if err == context.DeadlineExceeded {
		log.Error.Printf("Failed to set "+key+" key (timeout): %v", err)
	} else if cerr, ok := err.(*client.ClusterError); ok {
		log.Error.Printf("Failed to set "+key+" key (cluster error): %s", cerr.Detail())
	} else {
		log.Error.Printf("Failed to set "+key+" key (invalid endpoint): %v", err)
	}
}
