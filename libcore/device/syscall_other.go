//go:build !windows

package device

import "syscall"

func Flock(fd int, how int) (err error) {
	return syscall.Flock(fd, how)
}
