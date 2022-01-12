package libcore

import (
	"bufio"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"
	"unsafe"

	"github.com/v2fly/v2ray-core/v5/common/net"
)

var (
	netIndexOfLocal = -1
	netIndexOfUid   = -1
	nativeEndian    binary.ByteOrder
)

func querySocketUidFromProcFs(source, _ net.Destination) int32 {
	if netIndexOfLocal < 0 || netIndexOfUid < 0 {
		return -1
	}

	path := "/proc/net/"

	if source.Network == net.Network_TCP {
		path += "tcp"
	} else {
		path += "udp"
	}

	if source.Address.Family().IsIPv6() {
		path += "6"
	}

	file, err := os.Open(path)
	if err != nil {
		return -1
	}

	defer file.Close()

	reader := bufio.NewReader(file)
	var bytes [2]byte
	binary.BigEndian.PutUint16(bytes[:], uint16(source.Port))
	local := fmt.Sprintf("%s:%s", hex.EncodeToString(nativeEndianIP(source.Address.IP())), hex.EncodeToString(bytes[:]))

	for {
		row, _, err := reader.ReadLine()
		if err != nil {
			return -1
		}

		fields := strings.Fields(string(row))

		if len(fields) <= netIndexOfLocal || len(fields) <= netIndexOfUid {
			continue
		}

		if strings.EqualFold(local, fields[netIndexOfLocal]) {
			uid, err := strconv.Atoi(fields[netIndexOfUid])
			if err != nil {
				return -1
			}

			return int32(uid)
		}
	}
}

func nativeEndianIP(ip []byte) []byte {
	result := make([]byte, len(ip))

	for i := 0; i < len(result); i += 4 {
		value := binary.BigEndian.Uint32(ip[:i])

		nativeEndian.PutUint32(result[i:], value)
	}

	return result
}

func init() {
	file, err := os.Open("/proc/net/tcp")
	if err != nil {
		return
	}

	defer file.Close()

	reader := bufio.NewReader(file)

	header, _, err := reader.ReadLine()
	if err != nil {
		return
	}

	columns := strings.Fields(string(header))

	var txQueue, rxQueue, tr, tmWhen bool

	for idx, col := range columns {
		offset := 0

		if txQueue && rxQueue {
			offset--
		}

		if tr && tmWhen {
			offset--
		}

		switch col {
		case "tx_queue":
			txQueue = true
		case "rx_queue":
			rxQueue = true
		case "tr":
			tr = true
		case "tm->when":
			tmWhen = true
		case "local_address":
			netIndexOfLocal = idx + offset
		case "uid":
			netIndexOfUid = idx + offset
		}
	}
}

func init() {
	var x uint32 = 0x01020304
	if *(*byte)(unsafe.Pointer(&x)) == 0x01 {
		nativeEndian = binary.BigEndian
	} else {
		nativeEndian = binary.LittleEndian
	}
}
