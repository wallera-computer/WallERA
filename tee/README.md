# `tee`

This directory holds the ARM TrustZone-based TEE for wallera.

Build dependencies: `u-boot-tools`, `arm-none-eabi-gcc`, `qemu-system-arm`

## Architecture

This TEE implementation is largely inspired by the work of [F-Secure Foundry](https://github.com/f-secure-foundry/GoTEE-example).

Our implementation still uses GoTEE as its mean of existence, but implements strong separation between the Trusted OS and Trusted Applets.

WallERA's TEE employs the standard three-tier architecture described in ARM TrustZone literature:
 - non-secure OS running in non-secure world
 - secure OS running in secure world
 - trusted applets running in secure world

Comunication between layers happen by means of ARM's designated instructions:
 - an `smc` call triggers a context switch from non-secure OS supervisor mode to secure os supervisor mode
 - a `swi` call does a context switch from secure OS user mode to supervisor mode

The `trusted_os/nonsecuresyscall` package is an adaptation of the GoTEE's `syscall` package to work across the TrustZone boundaries, which then realizes communication between secure and non-secure world.

The build process embeds a copy of all the applets available, and a copy of the normal-world operating system.

Secure OS runs first, sets up execution contexts for all the applets and the normal-world operating system, then calls the latter for execution: secure OS effectively acts just as an execution monitor, routing data packets to/from the applets to/from the normal-world OS.

On top of that, secure OS sets up hardware firewalls to lock down memory and hardware as much as possible, to minimize the chances of forbidden access to sensible parts of the system from rogue normal-world OSes routines.

Communication happens through a simple mailbox design.

Normal world sends mail to a given applet, which will receive and process the data and then write a response to the same mailbox.

Each applet has a unique index, so that secure OS can easily route mail to the appropriate applet.

Applet execution is synchronous: once normal world sends mail, it will wait until an applet has processed the data and sent a reply, or until an error is produced.

In the future, applets will be subject to a processing timeout.
  
## Run the demo

```sh
make nonsecure_demo_os && make cryptography_applet && make trusted_os && make qemu
```
