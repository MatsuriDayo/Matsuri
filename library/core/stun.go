package libcore

import (
	"fmt"

	"libcore/stun"
)

type StunResult struct {
	Text    string
	Success bool
}

func StunTest(server string) *StunResult {
	//note: this library doesn't support stun1.l.google.com:19302
	ret := &StunResult{}

	client := stun.NewClient()
	client.SetServerAddr(server)
	nat, host, err := client.Discover()
	if err != nil {
		ret.Success = false
		ret.Text = err.Error()
		return ret
	}

	text := fmt.Sprintln("NAT Type:", nat)
	if host != nil {
		text += fmt.Sprintln("External IP Family:", host.Family())
		text += fmt.Sprintln("External IP:", host.IP())
		text += fmt.Sprintln("External Port:", host.Port())
	}
	ret.Success = true
	ret.Text = text
	return ret
}
