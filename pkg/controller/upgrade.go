package controller

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	protoetcd "kope.io/etcd-manager/pkg/apis/etcd"
)

func (m *EtcdController) stopForUpgrade(parentContext context.Context, clusterSpec *protoetcd.ClusterSpec, clusterState *etcdClusterState) (bool, error) {
	// We start a new context - this is pretty critical-path
	ctx := context.Background()

	// Sanity check
	memberToPeer := make(map[EtcdMemberId]*peer)
	for memberId, member := range clusterState.members {
		found := false
		for _, p := range clusterState.peers {
			if p.info == nil || p.info.NodeConfiguration == nil || p.peer == nil {
				continue
			}
			if p.info.NodeConfiguration.Name == string(member.Name) {
				memberToPeer[memberId] = p.peer
				found = true
			}
		}
		if !found {
			return false, fmt.Errorf("unable to find peer for %q", member.Name)
		}
	}

	if len(clusterState.healthyMembers) != len(clusterState.members) {
		// This one seems hard to relax
		return false, fmt.Errorf("cannot upgrade/downgrade cluster when not all members are healthy")
	}

	if len(clusterState.members) != int(clusterSpec.MemberCount) {
		// We could relax this, but we probably don't want to
		return false, fmt.Errorf("cannot upgrade/downgrade cluster when cluster is not at full member count")
	}

	// Force a backup, even before we start to do anything
	if _, err := m.doClusterBackup(ctx, clusterSpec, clusterState); err != nil {
		return false, fmt.Errorf("error doing backup before upgrade/downgrade: %v", err)
	}

	// We quarantine first, so that we don't have to get down to a single node before it is safe to do a backup
	if _, err := m.updateQuarantine(ctx, clusterState, true); err != nil {
		return false, err
	}

	// We do a backup
	backupResponse, err := m.doClusterBackup(ctx, clusterSpec, clusterState)
	if err != nil {
		return false, err
	}
	glog.Infof("backed up cluster as %v", backupResponse)

	// Stop the whole cluster
	for memberId := range clusterState.members {
		peer := memberToPeer[memberId]
		if peer == nil {
			// We checked this when we built the map
			panic("peer unexpectedly not found - logic error")
		}

		request := &protoetcd.StopEtcdRequest{
			ClusterName:     m.clusterName,
			LeadershipToken: m.leadership.token,
		}
		response, err := peer.rpcStopEtcd(ctx, request)
		if err != nil {
			return false, fmt.Errorf("error stopping etcd peer %q: %v", peer.Id, err)
		}
		glog.Infof("stopped etcd on peer %q: %v", peer.Id, response)
	}

	// TODO: Enforce the upgrade sequence 2.2.1 2.3.7 3.0.17 3.1.11
	// TODO: The 2 -> 3 upgrade is a dump anyway

	return true, nil
}
