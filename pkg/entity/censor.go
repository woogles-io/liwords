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
