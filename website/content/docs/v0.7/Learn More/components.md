---
title: "Components"
weight: 4
---

In this section we will discuss the various components of which Talos is comprised.

## Components

| Component                | Description                                                                                                                                                                                                                                                                                                   |
| ------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| [apid](apid)             | When interacting with Talos, the gRPC API endpoint you're interact with directly is provided by `apid`. `apid` acts as the gateway for all component interactions and forwards the requests to `routerd`.                                                                                                     |
| [containerd](containerd) | An industry-standard container runtime with an emphasis on simplicity, robustness and portability. To learn more see the [containerd website](https://containerd.io).                                                                                                                                         |
| [machined](machined)     | Talos replacement for the traditional Linux init-process. Specially designed to run Kubernetes and does not allow starting arbitrary user services.                                                                                                                                                           |
| [networkd](networkd)     | Handles all of the host level network configuration. Configuration is defined under the `networking` key                                                                                                                                                                                                      |
| [timed](timed)           | Handles the host time synchronization by acting as a NTP-client.                                                                                                                                                                                                                                              |
| [kernel](kernel)         | The Linux kernel included with Talos is configured according to the recommendations outlined in the [Kernel Self Protection Project](http://kernsec.org/wiki/index.php/Kernel_Self_Protection_Project).                                                                                                       |
| [routerd](routerd)       | Responsible for routing an incoming API request from `apid` to the appropriate backend (e.g. `networkd`, `machined` and `timed`).                                                                                                                                                                                  |
| [trustd](trustd)         | To run and operate a Kubernetes cluster a certain level of trust is required. Based on the concept of a 'Root of Trust', `trustd` is a simple daemon responsible for establishing trust within the system.                                                                                                    |
| [udevd](udevd)           | Implementation of `eudev` into `machined`. `eudev` is Gentoo's fork of udev, systemd's device file manager for the Linux kernel. It manages device nodes in /dev and handles all user space actions when adding or removing devices. To learn more see the [Gentoo Wiki](https://wiki.gentoo.org/wiki/Eudev). |

### apid

When interacting with Talos, the gRPC api endpoint you will interact with directly is `apid`.
Apid acts as the gateway for all component interactions.
Apid provides a mechanism to route requests to the appropriate destination when running on a control plane node.

We'll use some examples below to illustrate what `apid` is doing.

When a user wants to interact with a Talos component via `talosctl`, there are two flags that control the interaction with `apid`.
The `-e | --endpoints` flag is used to denote which Talos node ( via `apid` ) should handle the connection.
Typically this is a public facing server.
The `-n | --nodes` flag is used to denote which Talos node(s) should respond to the request.
If `--nodes` is not specified, the first endpoint will be used.

> Note: Typically there will be an `endpoint` already defined in the Talos config file.
> Optionally, `nodes` can be included here as well.

For example, if a user wants to interact with `machined`, a command like `talosctl -e cluster.talos.dev memory` may be used.

```bash
$ talosctl -e cluster.talos.dev memory
NODE                TOTAL   USED   FREE   SHARED   BUFFERS   CACHE   AVAILABLE
cluster.talos.dev   7938    1768   2390   145      53        3724    6571
```

In this case, `talosctl` is interacting with `apid` running on `cluster.talos.dev` and forwarding the request to the `machined` api.

If we wanted to extend our example to retrieve `memory` from another node in our cluster, we could use the command `talosctl -e cluster.talos.dev -n node02 memory`.

```bash
$ talosctl -e cluster.talos.dev -n node02 memory
NODE    TOTAL   USED   FREE   SHARED   BUFFERS   CACHE   AVAILABLE
node02  7938    1768   2390   145      53        3724    6571
```

The `apid` instance on `cluster.talos.dev` receives the request and forwards it to `apid` running on `node02` which forwards the request to the `machined` api.

We can further extend our example to retrieve `memory` for all nodes in our cluster by appending additional `-n node` flags or using a comma separated list of nodes ( `-n node01,node02,node03` ):

```bash
$ talosctl -e cluster.talos.dev -n node01 -n node02 -n node03 memory
NODE     TOTAL    USED    FREE     SHARED   BUFFERS   CACHE   AVAILABLE
node01   7938     871     4071     137      49        2945    7042
node02   257844   14408   190796   18138    49        52589   227492
node03   257844   1830    255186   125      49        777     254556
```

The `apid` instance on `cluster.talos.dev` receives the request and forwards is to `node01`, `node02`, and `node03` which then forwards the request to their local `machined` api.

### containerd

[Containerd](https://github.com/containerd/containerd) provides the container runtime to launch workloads on Talos as well as Kubernetes.

Talos services are namespaced under the `system` namespace in containerd whereas the Kubernetes services are namespaced under the `k8s.io` namespace.

### machined

A common theme throughout the design of Talos is minimalism.
We believe strongly in the UNIX philosophy that each program should do one job well.
The `init` included in Talos is one example of this, and we are calling it "`machined`".

We wanted to create a focused `init` that had one job - run Kubernetes.
To that extent, `machined` is relatively static in that it does not allow for arbitrary user defined services.
Only the services necessary to run Kubernetes and manage the node are available.
This includes:

- [containerd](containerd)
- [kubeadm](kubeadm)
- [kubelet](https://kubernetes.io/docs/concepts/overview/components/)
- [networkd](networkd)
- [timed](timed)
- [trustd](trustd)
- [udevd](udevd)

### networkd

Networkd handles all of the host level network configuration.
Configuration is defined under the `networking` key.

By default, we attempt to issue a DHCP request for every interface on the server.
This can be overridden by supplying one of the following kernel arguments:

- `talos.network.interface.ignore` - specify a list of interfaces to skip discovery on
- `ip` - `ip=<client-ip>:<server-ip>:<gw-ip>:<netmask>:<hostname>:<device>:<autoconf>:<dns0-ip>:<dns1-ip>:<ntp0-ip>` as documented in the [kernel here](https://www.kernel.org/doc/Documentation/filesystems/nfs/nfsroot.txt)
  - ex, `ip=10.0.0.99:::255.0.0.0:control-1:eth0:off:10.0.0.1`

### timed

Timed handles the host time synchronization.

### kernel

The Linux kernel included with Talos is configured according to the recommendations outlined in the Kernel Self Protection Project ([KSSP](http://kernsec.org/wiki/index.php/Kernel_Self_Protection_Project)).

### trustd

Security is one of the highest priorities within Talos.
To run a Kubernetes cluster a certain level of trust is required to operate a cluster.
For example, orchestrating the bootstrap of a highly available control plane requires the distribution of sensitive PKI data.

To that end, we created `trustd`.
Based on the concept of a Root of Trust, `trustd` is a simple daemon responsible for establishing trust within the system.
Once trust is established, various methods become available to the trustee.
It can, for example, accept a write request from another node to place a file on disk.

Additional methods and capability will be added to the `trustd` component in support of new functionality in the rest of the Talos environment.

### udevd

Udevd handles the kernel device notifications and sets up the necessary links in `/dev`.
