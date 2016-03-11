package cluster

import (
	"os"
	"testing"

	"github.com/cchamplin/deployd/cluster"
	"github.com/cchamplin/deployd/deployment"
	"github.com/cchamplin/deployd/log"
	"github.com/coreos/etcd/client"
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
)

type TestingMachine struct {
	Backend *EtcdBackend
	Id      string
	Cluster *cluster.Cluster
}

/*func TestEtcdBackendClusterTracking(t *testing.T) {

	log.InitLogger(os.Stdout, os.Stdout, os.Stdout, os.Stderr)
	test1 := uuid.NewV4().String()
	localM := cluster.LocalMachine(test1, []string{})
	localM.Id = test1
	var localCluster cluster.Cluster
	localBackend := new(EtcdBackend)
	localCluster.Init(localBackend, "../tests")

	var fakeMachines [8]TestingMachine
	for i := 0; i < 5; i++ {
		u1 := uuid.NewV4().String()
		var clstr cluster.Cluster
		backend := new(EtcdBackend)
		clstr.Init(backend, "../tests")
		m := cluster.LocalMachine(u1, []string{})
		m.Id = u1
		backend.Init(&clstr, m, test1)
		fakeMachines[i] = TestingMachine{Backend: backend, Id: u1, Cluster: &clstr}
	}

	localBackend.Init(&localCluster, localM, test1)
	assert.Equal(t, len(localCluster.Machines), 5, "")
	for i := 0; i < 5; i++ {
		foundMachine := ""
		for x := 0; x < len(fakeMachines); x++ {
			if localCluster.Machines[i].Id == fakeMachines[x].Id {
				foundMachine = fakeMachines[x].Id
				break
			}
		}
		assert.Equal(t, localCluster.Machines[i].Id, foundMachine, "")
	}

	for i := 5; i < 8; i++ {
		u1 := uuid.NewV4().String()
		var clstr cluster.Cluster
		backend := new(EtcdBackend)
		clstr.Init(backend, "../tests")
		m := cluster.LocalMachine(u1, []string{})
		m.Id = u1
		backend.Init(&clstr, m, test1)
		fakeMachines[i] = TestingMachine{Backend: backend, Id: u1, Cluster: &clstr}
	}
	assert.Equal(t, len(localCluster.Machines), 8, "")
	for i := 0; i < len(localCluster.Machines); i++ {
		foundMachine := ""
		for x := 0; x < len(fakeMachines); x++ {
			if localCluster.Machines[i].Id == fakeMachines[x].Id {
				foundMachine = fakeMachines[x].Id
				break
			}
		}
		assert.Equal(t, localCluster.Machines[i].Id, foundMachine, "")
	}
	localBackend.Signal <- 1
	for x := 0; x < len(fakeMachines); x++ {
		fakeMachines[x].Backend.Signal <- 1
	}
	localBackend.cleanupEtcd()
}*/

func TestEtcdBackendClusterRecoverNode(t *testing.T) {

	log.InitLogger(os.Stdout, os.Stdout, os.Stdout, os.Stderr)
	test1 := uuid.NewV4().String()
	localM := cluster.LocalMachine(test1, []string{})
	localM.Id = test1
	var localCluster cluster.Cluster
	localBackend := new(EtcdBackend)
	localCluster.Init(localBackend, "../tests")

	var fakeMachines [10]TestingMachine
	for i := 0; i < 10; i++ {
		u1 := uuid.NewV4().String()
		var clstr cluster.Cluster
		backend := new(EtcdBackend)
		clstr.Init(backend, "../tests")
		m := cluster.LocalMachine(u1, []string{})
		m.Id = u1
		backend.Init(&clstr, m)
		backend.setupTestNode(test1)
		for x := 0; x < (10 - i); x++ {
			u1 := uuid.NewV4().String()
			deployment := deployment.Deployment{
				Id:        u1,
				PackageId: "test_noop",
			}
			backend.DeploymentComplete(&deployment)
		}

		fakeMachines[i] = TestingMachine{Backend: backend, Id: u1, Cluster: &clstr}
	}

	localBackend.Init(&localCluster, localM)
	localBackend.setupTestNodeWithRecovery(test1)
	assert.Equal(t, len(localCluster.Machines), 10, "")

	// Take down a machine
	fakeMachines[5].Backend.Signal <- 1

	for {
		status := <-localBackend.Status
		log.Info.Printf("Got Status: %s", status)
		if status == "Not Recovering" || status == "Recovered" {
			break
		}
	}
	assert.Equal(t, len(localCluster.Machines), 11, "")
	localBackend.Signal <- 1
	for x := 0; x < len(fakeMachines); x++ {
		fakeMachines[x].Backend.Signal <- 1
	}
	localBackend.cleanupEtcd()
}

func (e *EtcdBackend) cleanupEtcd() {
	options := client.DeleteOptions{Recursive: true}

	e.kapi.Delete(context.Background(), e.backendConfig.MachinePrefix, &options)
	e.kapi.Delete(context.Background(), e.backendConfig.DeploymentPrefix, &options)
}

func (e *EtcdBackend) setupTestNode(prefix string) {
	e.backendConfig.RecoveryParticipant = false
	e.backendConfig.Prefix = prefix
}

func (e *EtcdBackend) setupTestNodeWithRecovery(prefix string) {
	e.backendConfig.RecoveryParticipant = true
	e.backendConfig.Prefix = prefix
}

func (e *EtcdBackend) etcdExpire(key string, value string) {
	options := client.SetOptions{TTL: 1}

	e.kapi.Set(context.Background(), key, value, &options)
}
