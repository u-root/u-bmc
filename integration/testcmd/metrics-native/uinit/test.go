// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/u-root/u-bmc/integration/utils"
)

func testMetrics(u string) error {
	resp, err := http.Get(u)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("non-200 HTTP status: %d", resp.StatusCode)
	}
	// TODO(bluecmd): Verify that it was an OpenMetrics page that was received
	var buf bytes.Buffer
	_, err = buf.ReadFrom(resp.Body)
	if err != nil {
		return err
	}

	if !strings.Contains(buf.String(), "ubmc_system_version") {
		return fmt.Errorf("Could not find metric ubmc_system_version among the metrics")
	}

	return nil
}

func uinit() error {
	if err := utils.AddIP("10.0.2.1/24", "eth0"); err != nil {
		return fmt.Errorf("Error adding IPv4 interface: %v", err)
	}
	if err := utils.SetLinkUp("eth0"); err != nil {
		return fmt.Errorf("Error setting link up on interface: %v", err)
	}

	for _, i := range []int{100, 500, 5000, 15000, -1} {
		if i == -1 {
			return fmt.Errorf("Timed out fetching metrics")
		}

		time.Sleep(time.Duration(i) * time.Millisecond)

		if err := testMetrics("http://10.0.2.15:9370/metrics"); err != nil {
			log.Printf("Error verifying metrics over IPv4: %v, retrying", err)
			continue
		}
		if err := testMetrics("http://[fe80::c00:00ff:fe00:0000%25eth0]:9370/metrics"); err != nil {
			log.Printf("Error verifying metrics over IPv6: %v, retrying", err)
			continue
		}
		break
	}
	return nil
}

func main() {
	if err := uinit(); err != nil {
		utils.FailTest(err)
	} else {
		utils.PassTest()
	}
}
