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
