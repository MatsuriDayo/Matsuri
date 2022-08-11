//go:build !linux
package nat

func natAcceptControl(fd uintptr) {
}
