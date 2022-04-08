package main

import (
	"fmt"
	"os"
	_ "unsafe"

	"github.com/v2fly/v2ray-core/v5/main/commands"
)

// PC版 适配 qv2ray core & 插件

//go:linkname build github.com/v2fly/v2ray-core/v5.build
var build string

var version_v2ray string = "N/A"
var version_standalone string = "N/A"

func main() {
	fmt.Println("V2Ray:", version_v2ray, "Version:", version_standalone)
	fmt.Println()

	build = "Matsuridayo/Qv2ray"
	commands.CmdRun.Run(commands.CmdRun, os.Args[1:])
}
