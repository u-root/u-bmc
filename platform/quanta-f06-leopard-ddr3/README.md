# Quanta F06 Leopard DDR3

This is a platform implementing OCP FB Server Intel Motherboard v3.
Read more at [this blog post](https://blog.mainframe.dev/2018/08/open-datacenter-hardware-leopard-server.html).

The platform uses 57600 baud rate for all serial ports as that is what it shipped with.
Rumor has it that this is due to EMC issues running at higher speed introducing phantom
input.

## Quirks

These are some platform quirks that have not been worked around yet.

### Serial port mux
The serial port mux does not work until the main board has been powered on. In addition
to this the default serial port output is the main serial console, not the BMC. This means
that the default power-on state is pretty much useless until the first boot has occured.

This would be fixable with access to the CPLD code most likely, but alas that's not
possible at this time.

The serial port mux has two leds next to it. The leds represent the selected console.

Seen from the front of the machine, this is the decoding table.

| Leds        | UART                    |
| ----------- |-------------------------|
| 00          | Main board              |
| 01          | BMC                     |
| 10          | Backplane (always dead) |

Use a console board to toggle and attach to the serial port.
