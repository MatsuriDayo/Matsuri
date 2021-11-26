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

var foregroundUid uint16

func SetForegroundUid(uid int32) {
	foregroundUid = uint16(uid)
}

var foregroundImeUid uint16

func SetForegroundImeUid(uid int32) {
	foregroundImeUid = uint16(uid)
}
