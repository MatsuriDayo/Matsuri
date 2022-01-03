package libcore

import (
	"github.com/golang/protobuf/proto"
	"github.com/v2fly/v2ray-core/v5/app/observatory"
)

func (instance *V2RayInstance) GetObservatoryStatus() ([]byte, error) {
	if instance.observatory == nil {
		return nil, newError("observatory unavailable")
	}
	resp, err := instance.observatory.GetObservation(nil)
	if err != nil {
		return nil, err
	}
	return proto.Marshal(resp)
}

func (instance *V2RayInstance) UpdateStatus(status []byte) error {
	if instance.observatory == nil {
		return newError("observatory unavailable")
	}
	s := new(observatory.OutboundStatus)
	err := proto.Unmarshal(status, s)
	if err != nil {
		return err
	}
	instance.observatory.UpdateStatus(s)
	return err
}

type StatusUpdateListener interface {
	OnUpdate(status []byte)
}

func (instance *V2RayInstance) SetStatusUpdateListener(listener StatusUpdateListener) {
	if listener == nil {
		instance.observatory.StatusUpdate = nil
	} else {
		instance.observatory.StatusUpdate = func(result *observatory.OutboundStatus) {
			status, _ := proto.Marshal(result)
			listener.OnUpdate(status)
		}
	}
}
