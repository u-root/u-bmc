# ubmcctl

u-bmc control. Client CLI.

Usage:

Set fan speed when logged into u-bmc over SSH:

```
ubmcctl SetFan fan: 0, mode: FAN_MODE_PERCENTAGE, percentage: 50
```

Get current fan settings:

```
ubmcctl --host 10.0.10.20 GetFans
```

Stream the host console:

```
ubmcctl StreamConsole -
# Console output will now stream in protobuf ascii form
# You can send data by writing e.g. 'data: "hello\n"'
```
