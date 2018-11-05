Welcome to u-bmc's documentation!
=================================

.. toctree::
   :maxdepth: 2
   :caption: Contents:

u-bmc is a Linux OS distribution that is fully open-source for
baseboard management controllers, or BMCs.

Why u-bmc?
----------

Take a step back and ask yourself, why do we have BMCs?

You're most likely going to come up with an answer that has something to do
with managing servers remotely, or maybe debugging them. Both are valid
use-cases for BMCs. In addition, some servers are manufactured to rely on the
BMC for tasks such as fan control.

This creates a problem, since the two previous stated use-cases
(managing and debugging) requires high level of access to the host system.
As a debugger the BMC has supreme access to critical resources, and as a
system manager its function is critical for system function.

A system that is both critical and highly privileged should be easy to audit,
and employ modern security. Those are the goals of u-bmc.

u-bmc sacrifices classical industry compatibility in order to
offer a solution that is genuinely tailored for the mission. This usually
results in a more secure implementation but also better integration with other
systems in general. Example: IPMI is replaced with gRPC, and SNMP with
OpenMetrics.

To ease adoption for the users that require classical interfaces there are
protocol adapters being planned that run off-BMC which converts from protocols
like Redfish to gRPC.

How?
----

u-bmc uses u-root as a framework to create a minimal Linux distribution
including only the bare essentials you need from a BMC. If you do not need
a particular function, you can turn it off from the configuration file and it
will not be compiled into the binary.

Relation to LinuxBoot
~~~~~~~~~~~~~~~~~~~~~

Historically BMCs have been considered insecure by nature and have received
little to no attention - not unlike BIOSes. While LinuxBoot's mission is to
uplift BIOS firmware for existing servers, u-bmc's is to uplift BMC firmware.
The implementations differ, some things are shared, but the goal is the same.
u-bmc, like LinuxBoot, sacrifices classical industry compatibility in order to
offer a solution that is genuinely tailored for the mission.

By close collaboration between LinuxBoot and u-bmc, the hope is to one day
have servers that ship with both free and open BMC as well as BIOS.
