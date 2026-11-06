//go:build !linux

package network

import "errors"

func ConntrackAvailable() error { return errors.New("conntrack only available on Linux/Android") }
func PurgeDNS(int) error        { return errors.New("conntrack only available on Linux/Android") }
