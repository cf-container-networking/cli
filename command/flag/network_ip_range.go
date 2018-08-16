package flag

import (
	"fmt"
	"net"
	"strings"

	flags "github.com/jessevdk/go-flags"
)

type NetworkIPRange struct {
	Start string
	End   string
}

func (ir *NetworkIPRange) UnmarshalFlag(val string) error {

	ips := strings.Split(val, "-")
	if len(ips) == 1 {
		ir.Start = ips[0]
		ir.End = ips[0]
	} else if len(ips) == 2 {
		ir.Start = ips[0]
		ir.End = ips[1]
	} else {
		return &flags.Error{
			Type:    flags.ErrUnknown,
			Message: fmt.Sprintf("invalid ip range format: %s", val),
		}
	}

	for _, ipAddr := range ips {
		if net.ParseIP(ipAddr) == nil {
			return &flags.Error{
				Type:    flags.ErrUnknown,
				Message: fmt.Sprintf("invalid ip address: %s", ipAddr),
			}
		}
	}

	return nil
}
