package tcpip

var zeroChecksum = [2]byte{0x00, 0x00}

var sum = sumCompat

func Sum(b []byte) uint32 {
	return sum(b)
}

// Checksum for Internet Protocol family headers
func Checksum(sum uint32, b []byte) (answer [2]byte) {
	sum += Sum(b)
	sum = (sum >> 16) + (sum & 0xffff)
	sum += sum >> 16
	sum = ^sum
	answer[0] = byte(sum >> 8)
	answer[1] = byte(sum)
	return
}

func IsIPv4(packet []byte) bool {
	return len(packet) > 0 && (packet[0]>>4) == 4
}

func IsIPv6(packet []byte) bool {
	return len(packet) > 0 && (packet[0]>>4) == 6
}

func SetIPv4(packet []byte) {
	packet[0] = (packet[0] & 0x0f) | (4 << 4)
}

func SetIPv6(packet []byte) {
	packet[0] = (packet[0] & 0x0f) | (6 << 4)
}
