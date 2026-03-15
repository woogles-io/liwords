package entity

import (
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"

	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// CensorRacks strips rack information from a ServerGameplayEvent.
func CensorRacks(sge *pb.ServerGameplayEvent) {
	sge.NewRack = ""
	if sge.Event != nil {
		sge.Event.Rack = ""
		if sge.Event.Type == macondopb.GameEvent_EXCHANGE {
			sge.Event.Exchanged = ""
		}
	}
}

// CensorHistoryRacks strips all rack information from a GameHistoryRefresher.
func CensorHistoryRacks(ghr *pb.GameHistoryRefresher) {
	if ghr.History == nil {
		return
	}
	for _, evt := range ghr.History.Events {
		evt.Rack = ""
		if evt.Type == macondopb.GameEvent_EXCHANGE {
			evt.Exchanged = ""
		}
	}
	if len(ghr.History.LastKnownRacks) >= 2 {
		ghr.History.LastKnownRacks[0] = ""
		ghr.History.LastKnownRacks[1] = ""
	}
}
