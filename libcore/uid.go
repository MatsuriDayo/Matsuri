package libcore

import (
	"syscall"

	"github.com/v2fly/v2ray-core/v5/common/net"
)

var (
	uidDumper UidDumper
	useProcfs bool
)

type UidInfo struct {
	PackageName string
	Label       string
}

type UidDumper interface {
	DumpUid(ipProto int32, srcIp string, srcPort int32, destIp string, destPort int32) (int32, error)
	GetUidInfo(uid int32) (*UidInfo, error)
}

func SetUidDumper(dumper UidDumper, procfs bool) {
	uidDumper = dumper
	useProcfs = procfs
}

func dumpUid(source net.Destination, destination net.Destination) (int32, error) {
	if useProcfs {
		return querySocketUidFromProcFs(source, destination), nil
	} else {
		var ipProto int32
		if destination.Network == net.Network_TCP {
			ipProto = syscall.IPPROTO_TCP
		} else {
			ipProto = syscall.IPPROTO_UDP
		}
		return uidDumper.DumpUid(ipProto, source.Address.IP().String(), int32(source.Port), destination.Address.IP().String(), int32(destination.Port))
	}
}
