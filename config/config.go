// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config

import (
	"github.com/u-root/u-bmc/pkg/bmc/ttime"
)

type Version struct {
	Version string
	GitHash string
}

type ACME struct {
	Directory   string
	Contact     string
	Token       string
	TermsAgreed bool
	APICA       string
}

type Config struct {
	RoughtimeServers    []ttime.RoughtimeServer
	NtpServers          []ttime.NtpServer
	StartDebugSshServer bool
	DebugSshServerKeys  []string
	Version             Version
	ACME                ACME
}

var DefaultConfig = &Config{
	// The philosophy behind the time configuration is to use a fast, simple, and
	// authenticated time protocol for the initial time configuration followed
	// by a refined adjustment by (unauthenticated) NTP for precision.
	// This severely limits how much an NTP server can lie to u-bmc (+/- a few
	// seconds) which makes it a mild inconvenience when reading logs but that's
	// about it.
	//
	// Other BMCs choose to trust the host's clock but given that the host
	// could be compromised and trying to fool the BMC into accepting a
	// bad time source that's not what u-bmc does.
	//
	// Instead, u-bmc will refuse to execute any remote actions until an accurate
	// time source has been established.
	RoughtimeServers: []ttime.RoughtimeServer{
		{Protocol: "udp", Address: "roughtime.cloudflare.com:2002", PublicKeyType: ttime.KEY_TYPE_ED25519, PublicKey: "gD63hSj3ScS+wuOeGrubXlq35N1c5Lby/S+T7MNTjxo="},
		{Protocol: "udp", Address: "roughtime.sandbox.google.com:2002", PublicKeyType: ttime.KEY_TYPE_ED25519, PublicKey: "etPaaIxcBMY1oUeGpwvPMCJMwlRVNxv51KK/tktoJTQ="},
		{Protocol: "udp", Address: "time.0xt.ca:2002", PublicKeyType: ttime.KEY_TYPE_ED25519, PublicKey: "iBVjxg/1j7y1+kQUTBYdTabxCppesU/07D4PMDJk2WA="},
		{Protocol: "udp", Address: "roughtime.int80h.com:2002", PublicKeyType: ttime.KEY_TYPE_ED25519, PublicKey: "AW5uAoTSTDfG5NfY1bTh08GUnOqlRb+HVhbJ3ODJvsE="},
	},

	// While u-bmc has been granted 0.u-bmc.pool.ntp.org, they do not currently
	// offer AAAA/IPv6 records. In the mean time use Google's leap smeared
	// NTP servers that do have IPv6.
	NtpServers: []ttime.NtpServer{"time1.google.com", "time2.google.com", "time3.google.com", "time4.google.com"},

	// This is useful if you're debugging startup problems in u-bmc.
	// NOTE: The SSH server starts before trusted time has been acquired,
	// do not use in production environments.
	StartDebugSshServer: debugSSH,

	// authorizedKeys is being read by the compiler using go embed
	DebugSshServerKeys: authorizedKeys,

	Version: Version{
		Version: gitVersion,
		GitHash: gitHash,
	},

	ACME: ACME{
		// 10.0.2.2 is QEMUs address for the host and where pebble is running.
		Directory:   "https://10.0.2.2:14000/dir",
		Contact:     "nobody@localhost",
		Token:       "none",
		TermsAgreed: termsAgreed,
		APICA:       letsEncryptRootCA,
	},
}

const (
	letsEncryptRootCA = letsEncryptX1 + letsEncryptX2

	letsEncryptX1 = `
-----BEGIN CERTIFICATE-----
MIIFazCCA1OgAwIBAgIRAIIQz7DSQONZRGPgu2OCiwAwDQYJKoZIhvcNAQELBQAw
TzELMAkGA1UEBhMCVVMxKTAnBgNVBAoTIEludGVybmV0IFNlY3VyaXR5IFJlc2Vh
cmNoIEdyb3VwMRUwEwYDVQQDEwxJU1JHIFJvb3QgWDEwHhcNMTUwNjA0MTEwNDM4
WhcNMzUwNjA0MTEwNDM4WjBPMQswCQYDVQQGEwJVUzEpMCcGA1UEChMgSW50ZXJu
ZXQgU2VjdXJpdHkgUmVzZWFyY2ggR3JvdXAxFTATBgNVBAMTDElTUkcgUm9vdCBY
MTCCAiIwDQYJKoZIhvcNAQEBBQADggIPADCCAgoCggIBAK3oJHP0FDfzm54rVygc
h77ct984kIxuPOZXoHj3dcKi/vVqbvYATyjb3miGbESTtrFj/RQSa78f0uoxmyF+
0TM8ukj13Xnfs7j/EvEhmkvBioZxaUpmZmyPfjxwv60pIgbz5MDmgK7iS4+3mX6U
A5/TR5d8mUgjU+g4rk8Kb4Mu0UlXjIB0ttov0DiNewNwIRt18jA8+o+u3dpjq+sW
T8KOEUt+zwvo/7V3LvSye0rgTBIlDHCNAymg4VMk7BPZ7hm/ELNKjD+Jo2FR3qyH
B5T0Y3HsLuJvW5iB4YlcNHlsdu87kGJ55tukmi8mxdAQ4Q7e2RCOFvu396j3x+UC
B5iPNgiV5+I3lg02dZ77DnKxHZu8A/lJBdiB3QW0KtZB6awBdpUKD9jf1b0SHzUv
KBds0pjBqAlkd25HN7rOrFleaJ1/ctaJxQZBKT5ZPt0m9STJEadao0xAH0ahmbWn
OlFuhjuefXKnEgV4We0+UXgVCwOPjdAvBbI+e0ocS3MFEvzG6uBQE3xDk3SzynTn
jh8BCNAw1FtxNrQHusEwMFxIt4I7mKZ9YIqioymCzLq9gwQbooMDQaHWBfEbwrbw
qHyGO0aoSCqI3Haadr8faqU9GY/rOPNk3sgrDQoo//fb4hVC1CLQJ13hef4Y53CI
rU7m2Ys6xt0nUW7/vGT1M0NPAgMBAAGjQjBAMA4GA1UdDwEB/wQEAwIBBjAPBgNV
HRMBAf8EBTADAQH/MB0GA1UdDgQWBBR5tFnme7bl5AFzgAiIyBpY9umbbjANBgkq
hkiG9w0BAQsFAAOCAgEAVR9YqbyyqFDQDLHYGmkgJykIrGF1XIpu+ILlaS/V9lZL
ubhzEFnTIZd+50xx+7LSYK05qAvqFyFWhfFQDlnrzuBZ6brJFe+GnY+EgPbk6ZGQ
3BebYhtF8GaV0nxvwuo77x/Py9auJ/GpsMiu/X1+mvoiBOv/2X/qkSsisRcOj/KK
NFtY2PwByVS5uCbMiogziUwthDyC3+6WVwW6LLv3xLfHTjuCvjHIInNzktHCgKQ5
ORAzI4JMPJ+GslWYHb4phowim57iaztXOoJwTdwJx4nLCgdNbOhdjsnvzqvHu7Ur
TkXWStAmzOVyyghqpZXjFaH3pO3JLF+l+/+sKAIuvtd7u+Nxe5AW0wdeRlN8NwdC
jNPElpzVmbUq4JUagEiuTDkHzsxHpFKVK7q4+63SM1N95R1NbdWhscdCb+ZAJzVc
oyi3B43njTOQ5yOf+1CceWxG1bQVs5ZufpsMljq4Ui0/1lvh+wjChP4kqKOJ2qxq
4RgqsahDYVvTH9w7jXbyLeiNdd8XM2w9U/t7y0Ff/9yi0GE44Za4rF2LN9d11TPA
mRGunUHBcnWEvgJBQl9nJEiU0Zsnvgc/ubhPgXRR4Xq37Z0j4r7g1SgEEzwxA57d
emyPxgcYxn/eR44/KJ4EBs+lVDR3veyJm+kXQ99b21/+jh5Xos1AnX5iItreGCc=
-----END CERTIFICATE-----
`
	letsEncryptX2 = `
-----BEGIN CERTIFICATE-----
MIICGzCCAaGgAwIBAgIQQdKd0XLq7qeAwSxs6S+HUjAKBggqhkjOPQQDAzBPMQsw
CQYDVQQGEwJVUzEpMCcGA1UEChMgSW50ZXJuZXQgU2VjdXJpdHkgUmVzZWFyY2gg
R3JvdXAxFTATBgNVBAMTDElTUkcgUm9vdCBYMjAeFw0yMDA5MDQwMDAwMDBaFw00
MDA5MTcxNjAwMDBaME8xCzAJBgNVBAYTAlVTMSkwJwYDVQQKEyBJbnRlcm5ldCBT
ZWN1cml0eSBSZXNlYXJjaCBHcm91cDEVMBMGA1UEAxMMSVNSRyBSb290IFgyMHYw
EAYHKoZIzj0CAQYFK4EEACIDYgAEzZvVn4CDCuwJSvMWSj5cz3es3mcFDR0HttwW
+1qLFNvicWDEukWVEYmO6gbf9yoWHKS5xcUy4APgHoIYOIvXRdgKam7mAHf7AlF9
ItgKbppbd9/w+kHsOdx1ymgHDB/qo0IwQDAOBgNVHQ8BAf8EBAMCAQYwDwYDVR0T
AQH/BAUwAwEB/zAdBgNVHQ4EFgQUfEKWrt5LSDv6kviejM9ti6lyN5UwCgYIKoZI
zj0EAwMDaAAwZQIwe3lORlCEwkSHRhtFcP9Ymd70/aTSVaYgLXTWNLxBo1BfASdW
tL4ndQavEi51mI38AjEAi/V3bNTIZargCyzuFJ0nN6T5U6VR5CmD1/iQMVtCnwr1
/q4AaOeMSQ+2b1tbFfLn
-----END CERTIFICATE-----
`
)
