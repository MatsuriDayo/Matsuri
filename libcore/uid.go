package libcore

var uidDumper UidDumper

type UidInfo struct {
	PackageName string
	Label       string
}

type UidDumper interface {
	DumpUid(ipv6 bool, udp bool, srcIp string, srcPort int32, destIp string, destPort int32) (int32, error)
	GetUidInfo(uid int32) (*UidInfo, error)
}

func SetUidDumper(dumper UidDumper) {
	uidDumper = dumper
}
