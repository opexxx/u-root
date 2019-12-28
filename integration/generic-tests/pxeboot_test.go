// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !race

package integration

import (
	"testing"
	"time"

	"github.com/u-root/u-root/pkg/qemu"
	"github.com/u-root/u-root/pkg/uroot"
	"github.com/u-root/u-root/pkg/vmtest"
)

// TestPxeboot runs a server and client to test pxebooting a node.
// TODO: FIX THIS TEST!
// Change the t.Logf below back to t.Errorf
func TestPxeboot(t *testing.T) {
	// TODO: support arm
	if vmtest.TestArch() != "amd64" {
		t.Skipf("test not supported on %s", vmtest.TestArch())
	}

	network := qemu.NewNetwork()
	dhcpServer, scleanup := vmtest.QEMUTest(t, &vmtest.Options{
		Name: "TestPxeboot_Server",
		BuildOpts: uroot.Opts{
			ExtraFiles: []string{
				"./testdata/pxe:pxeroot",
			},
		},
		TestCmds: []string{
			"ip addr add 192.168.0.1/24 dev eth0",
			"ip link set eth0 up",
			"ip route add 0.0.0.0/0 dev eth0",
			"ls -l /pxeroot",
			"pxeserver -tftp-dir=/pxeroot",
		},
		QEMUOpts: qemu.Options{
			SerialOutput: vmtest.TestLineWriter(t, "server"),
			Timeout:      30 * time.Second,
			Devices: []qemu.Device{
				network.NewVM(),
			},
		},
	})
	defer scleanup()

	dhcpClient, ccleanup := vmtest.QEMUTest(t, &vmtest.Options{
		Name: "TestPxeboot_Client",
		BuildOpts: uroot.Opts{
			Commands: uroot.BusyBoxCmds(
				"github.com/u-root/u-root/cmds/core/init",
				"github.com/u-root/u-root/cmds/core/elvish",
				"github.com/u-root/u-root/cmds/core/ip",
				"github.com/u-root/u-root/cmds/core/shutdown",
				"github.com/u-root/u-root/cmds/core/sleep",
				"github.com/u-root/u-root/cmds/boot/pxeboot",
			),
		},
		TestCmds: []string{
			"pxeboot --dry-run --no-load -v",
			// Sleep so serial console output gets flushed. The expect library is racy.
			"sleep 5",
			"shutdown -h",
		},
		QEMUOpts: qemu.Options{
			SerialOutput: vmtest.TestLineWriter(t, "client"),
			Timeout:      30 * time.Second,
			Devices: []qemu.Device{
				network.NewVM(),
			},
		},
	})
	defer ccleanup()

	if err := dhcpServer.Expect("starting file server"); err != nil {
		t.Logf("File server: %v", err)
	}
	if err := dhcpClient.Expect("Got DHCPv4 lease on eth0:"); err != nil {
		t.Logf("Lease %v:", err)
	}
	if err := dhcpClient.Expect("Boot URI: tftp://192.168.0.1/pxelinux.0"); err != nil {
		t.Logf("Boot: %v", err)
	}
}