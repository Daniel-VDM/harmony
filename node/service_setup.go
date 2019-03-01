package node

import (
	msg_pb "github.com/harmony-one/harmony/api/proto/message"
	"github.com/harmony-one/harmony/api/service"
	"github.com/harmony-one/harmony/api/service/blockproposal"
	"github.com/harmony-one/harmony/api/service/clientsupport"
	"github.com/harmony-one/harmony/api/service/consensus"
	"github.com/harmony-one/harmony/api/service/discovery"
	"github.com/harmony-one/harmony/api/service/explorer"
	"github.com/harmony-one/harmony/api/service/networkinfo"
	"github.com/harmony-one/harmony/api/service/randomness"
	"github.com/harmony-one/harmony/api/service/staking"
	"github.com/harmony-one/harmony/internal/utils"
	"github.com/harmony-one/harmony/p2p"
)

func (node *Node) setupForShardLeader() {
	nodeConfig, chanPeer := node.initNodeConfiguration()

	// Register peer discovery service. No need to do staking for beacon chain node.
	node.serviceManager.RegisterService(service.PeerDiscovery, discovery.New(node.host, nodeConfig, chanPeer, node.AddBeaconPeer))
	// Register networkinfo service. "0" is the beacon shard ID
	node.serviceManager.RegisterService(service.NetworkInfo, networkinfo.New(node.host, p2p.GroupIDBeacon, chanPeer, nil))

	// Register explorer service.
	node.serviceManager.RegisterService(service.SupportExplorer, explorer.New(&node.SelfPeer))
	// Register consensus service.
	node.serviceManager.RegisterService(service.Consensus, consensus.New(node.BlockChannel, node.Consensus, node.startConsensus))
	// Register new block service.
	node.serviceManager.RegisterService(service.BlockProposal, blockproposal.New(node.Consensus.ReadySignal, node.WaitForConsensusReady))
	// Register client support service.
	node.serviceManager.RegisterService(service.ClientSupport, clientsupport.New(node.blockchain.State, node.CallFaucetContract, node.getDeployedStakingContract, node.SelfPeer.IP, node.SelfPeer.Port))
	// Register randomness service
	node.serviceManager.RegisterService(service.Randomness, randomness.New(node.DRand))
}

func (node *Node) stopShardLeaderServices() {
	node.serviceManager.StopService(service.PeerDiscovery)
	node.serviceManager.StopService(service.NetworkInfo)
	node.serviceManager.StopService(service.SupportExplorer)
	node.serviceManager.StopService(service.Consensus)
	node.serviceManager.StopService(service.BlockProposal)
	node.serviceManager.StopService(service.ClientSupport)
	node.serviceManager.StopService(service.Randomness)
}

func (node *Node) setupForShardValidator() {
	nodeConfig, chanPeer := node.initNodeConfiguration()

	// Register peer discovery service. "0" is the beacon shard ID. No need to do staking for beacon chain node.
	node.serviceManager.RegisterService(service.PeerDiscovery, discovery.New(node.host, nodeConfig, chanPeer, node.AddBeaconPeer))
	// Register networkinfo service. "0" is the beacon shard ID
	node.serviceManager.RegisterService(service.NetworkInfo, networkinfo.New(node.host, p2p.GroupIDBeacon, chanPeer, nil))
}

func (node *Node) stopShardValidatorServices() {
	node.serviceManager.StopService(service.PeerDiscovery)
	node.serviceManager.StopService(service.NetworkInfo)
}

func (node *Node) setupForBeaconLeader() {
	nodeConfig, chanPeer := node.initBeaconNodeConfiguration()

	// Register peer discovery service. No need to do staking for beacon chain node.
	node.serviceManager.RegisterService(service.PeerDiscovery, discovery.New(node.host, nodeConfig, chanPeer, nil))
	// Register networkinfo service.
	node.serviceManager.RegisterService(service.NetworkInfo, networkinfo.New(node.host, p2p.GroupIDBeacon, chanPeer, nil))
	// Register consensus service.
	node.serviceManager.RegisterService(service.Consensus, consensus.New(node.BlockChannel, node.Consensus, node.startConsensus))
	// Register new block service.
	node.serviceManager.RegisterService(service.BlockProposal, blockproposal.New(node.Consensus.ReadySignal, node.WaitForConsensusReady))
	// Register client support service.
	node.serviceManager.RegisterService(service.ClientSupport, clientsupport.New(node.blockchain.State, node.CallFaucetContract, node.getDeployedStakingContract, node.SelfPeer.IP, node.SelfPeer.Port))
	// Register randomness service
	node.serviceManager.RegisterService(service.Randomness, randomness.New(node.DRand))
}

func (node *Node) stopBeaconLeaderServices() {
	node.serviceManager.StopService(service.PeerDiscovery)
	node.serviceManager.StopService(service.NetworkInfo)
	node.serviceManager.StopService(service.Consensus)
	node.serviceManager.StopService(service.BlockProposal)
	node.serviceManager.StopService(service.ClientSupport)
	node.serviceManager.StopService(service.Randomness)
}

func (node *Node) setupForBeaconValidator() {
	nodeConfig, chanPeer := node.initBeaconNodeConfiguration()

	// Register peer discovery service. No need to do staking for beacon chain node.
	node.serviceManager.RegisterService(service.PeerDiscovery, discovery.New(node.host, nodeConfig, chanPeer, nil))
	// Register networkinfo service.
	node.serviceManager.RegisterService(service.NetworkInfo, networkinfo.New(node.host, p2p.GroupIDBeacon, chanPeer, nil))
}

func (node *Node) stopBeaconValidatorServices() {
	node.serviceManager.StopService(service.PeerDiscovery)
	node.serviceManager.StopService(service.NetworkInfo)
}

func (node *Node) setupForNewNode() {
	nodeConfig, chanPeer := node.initNodeConfiguration()

	// Register staking service.
	node.serviceManager.RegisterService(service.Staking, staking.New(node.host, node.AccountKey, node.beaconChain))
	// Register peer discovery service. "0" is the beacon shard ID
	node.serviceManager.RegisterService(service.PeerDiscovery, discovery.New(node.host, nodeConfig, chanPeer, node.AddBeaconPeer))
	// Register networkinfo service. "0" is the beacon shard ID
	node.serviceManager.RegisterService(service.NetworkInfo, networkinfo.New(node.host, p2p.GroupIDBeacon, chanPeer, nil))

	// TODO: how to restart networkinfo and discovery service after receiving shard id info from beacon chain?
}

func (node *Node) stopNewNodeServices() {
	node.serviceManager.StopService(service.PeerDiscovery)
	node.serviceManager.StopService(service.NetworkInfo)
	node.serviceManager.StopService(service.Staking)
}

func (node *Node) setupForClientNode() {
	nodeConfig, chanPeer := node.initNodeConfiguration()

	// Register peer discovery service.
	node.serviceManager.RegisterService(service.PeerDiscovery, discovery.New(node.host, nodeConfig, chanPeer, nil))
	// Register networkinfo service. "0" is the beacon shard ID
	node.serviceManager.RegisterService(service.NetworkInfo, networkinfo.New(node.host, p2p.GroupIDBeacon, chanPeer, nil))
}

func (node *Node) stopClientNodeServices() {
	node.serviceManager.StopService(service.PeerDiscovery)
	node.serviceManager.StopService(service.NetworkInfo)
}

// ServiceManagerSetup setups service store.
func (node *Node) ServiceManagerSetup() {
	node.serviceManager = &service.Manager{}
	node.serviceMessageChan = make(map[service.Type]chan *msg_pb.Message)
	switch node.Role {
	case ShardLeader:
		node.setupForShardLeader()
	case ShardValidator:
		node.setupForShardValidator()
	case BeaconLeader:
		node.setupForBeaconLeader()
	case BeaconValidator:
		node.setupForBeaconValidator()
	case NewNode:
		node.setupForNewNode()
	case ClientNode:
		node.setupForClientNode()
	}
	node.serviceManager.SetupServiceMessageChan(node.serviceMessageChan)
}

// RunServices runs registered services.
func (node *Node) RunServices() {
	if node.serviceManager == nil {
		utils.GetLogInstance().Info("Service manager is not set up yet.")
		return
	}
	node.serviceManager.RunServices()
}

func (node *Node) StopServicesByRole(role Role) {
	switch role {
	case ShardLeader:
		node.stopShardLeaderServices()
	case ShardValidator:
		node.stopShardValidatorServices()
	case BeaconLeader:
		node.stopBeaconLeaderServices()
	case BeaconValidator:
		node.stopBeaconValidatorServices()
	case NewNode:
		node.stopNewNodeServices()
	case ClientNode:
		node.stopClientNodeServices()
	}
}