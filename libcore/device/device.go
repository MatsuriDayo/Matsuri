package device

import (
	"runtime"

	"go.uber.org/automaxprocs/maxprocs"
)

func AutoGoMaxProcs() {
	maxprocs.Set(maxprocs.Logger(func(string, ...interface{}) {}))
}

func NumUDPWorkers() int {
	numUDPWorkers := 4
	if num := runtime.GOMAXPROCS(0); num > numUDPWorkers {
		numUDPWorkers = num
	}
	return numUDPWorkers
}
