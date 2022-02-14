package device

var DebugFunc func()

func GoDebug() {
	if DebugFunc != nil {
		go DebugFunc()
	}
}
