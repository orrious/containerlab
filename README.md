<p align=center><a href="https://containerlab.dev"><img src=docs/images/containerlab_export_white_ink.svg?sanitize=true/></a></p>

[![github release](https://img.shields.io/github/release/srl-labs/containerlab.svg?style=flat-square&color=00c9ff&labelColor=bec8d2)](https://github.com/srl-labs/containerlab/releases/)
[![Github all releases](https://img.shields.io/github/downloads/srl-labs/containerlab/total.svg?style=flat-square&color=00c9ff&labelColor=bec8d2)](https://github.com/srl-labs/containerlab/releases/)
[![Doc](https://img.shields.io/badge/Docs-containerlab.dev-blue?style=flat-square&color=00c9ff&labelColor=bec8d2)](https://containerlab.dev)
[![DeepWiki](https://img.shields.io/badge/deepwiki-1DA1F2?logo=wikipedia&style=flat-square&color=00c9ff&labelColor=bec8d2&logoColor=black)](https://deepwiki.com/srl-labs/containerlab)
[![Bluesky](https://img.shields.io/badge/follow-containerlab-1DA1F2?logo=bluesky&style=flat-square&color=00c9ff&labelColor=bec8d2)](https://bsky.app/profile/containerlab.dev)
[![Discord](https://img.shields.io/discord/860500297297821756?style=flat-square&label=discord&logo=discord&color=00c9ff&labelColor=bec8d2)](https://discord.gg/vAyddtaEV9)

---

With the growing number of containerized Network Operating Systems grows the demand to easily run them in the user-defined, versatile lab topologies.

Unfortunately, container orchestration tools like docker-compose are not a good fit for that purpose, as they do not allow a user to easily create connections between the containers which define a topology.

Containerlab provides a CLI for orchestrating and managing container-based networking labs. It starts the containers, builds a virtual wiring between them to create lab topologies of users choice and manages labs lifecycle.

![pic](https://gitlab.com/rdodin/pics/-/wikis/uploads/01fcdc212ee1c7de70ef5d2a8d109044/image.png)

Containerlab focuses on the containerized Network Operating Systems which are typically used to test network features and designs, such as:

* [Nokia SR Linux](https://containerlab.dev/manual/kinds/srl/)
* [Nokia virtual SR OS (SR-SIM)](https://containerlab.dev/manual/kinds/sros/)
* [Arista cEOS](https://containerlab.dev/manual/kinds/ceos/)
* [Cisco XRd](https://containerlab.dev/manual/kinds/xrd/)
* [SONiC](https://containerlab.dev/manual/kinds/sonic-vs/)
* [Juniper cRPD](https://containerlab.dev/manual/kinds/crpd/)
* [Juniper cSRX](https://containerlab.dev/manual/kinds/csrx/)
* [Cumulus VX](https://containerlab.dev/manual/kinds/cvx/)
* [Keysight IXIA-C](https://containerlab.dev/manual/kinds/keysight_ixia-c-one/)
* [RARE/FreeRtr](https://containerlab.dev/manual/kinds/rare-freertr/)
* [Ostinato](https://containerlab.dev/manual/kinds/ostinato/)
* [Spirent TestCenter](https://containerlab.dev/manual/kinds/spirent_stc/)
* [veesix osvbng](https://containerlab.dev/manual/kinds/veesix_osvbng/)
* [6WIND VSR](https://containerlab.dev/manual/kinds/6wind_vsr/)
* [FD.io VPP](https://containerlab.dev/manual/kinds/fdio_vpp/)
* [VyOS Networks VyOS](https://containerlab.dev/manual/kinds/vyosnetworks_vyos/)
* [Arrcus ArcOS](https://containerlab.dev/manual/kinds/arrcus_arcos/)

In addition to native containerized NOSes, containerlab can launch traditional virtual machine based routers using [vrnetlab or boxen integration](https://containerlab.dev/manual/vrnetlab/):

* [Nokia virtual SR OS (vSim)](https://containerlab.dev/manual/kinds/vr-sros/)
* [Juniper vMX](https://containerlab.dev/manual/kinds/vr-vmx/)
* [Juniper vQFX](https://containerlab.dev/manual/kinds/vr-vqfx/)
* [Juniper vSRX](https://containerlab.dev/manual/kinds/vr-vsrx/)
* [Juniper vJunos-router](https://containerlab.dev/manual/kinds/vr-vjunosrouter/)
* [Juniper vJunos-switch](https://containerlab.dev/manual/kinds/vr-vjunosswitch/)
* [Juniper vJunosEvolved](https://containerlab.dev/manual/kinds/vr-vjunosevolved/)
* [Cisco IOS XRv9k](https://containerlab.dev/manual/kinds/vr-xrv9k/)
* [Cisco XRd vRouter](https://containerlab.dev/manual/kinds/cisco_xrd_vrouter/)
* [Cisco Nexus 9000v](https://containerlab.dev/manual/kinds/vr-n9kv)
* [Cisco c8000v](https://containerlab.dev/manual/kinds/vr-c8000v/)
* [Cisco SD-WAN](https://containerlab.dev/manual/kinds/cisco_sdwan/)
* [Cisco CSR 1000v](https://containerlab.dev/manual/kinds/vr-csr)
* [Cisco vIOS](https://containerlab.dev/manual/kinds/cisco_vios/)
* [Cisco ASAv](https://containerlab.dev/manual/kinds/cisco_asav/)
* [Cisco FTDv](https://containerlab.dev/manual/kinds/vr-ftdv/)
* [Dell FTOS10v](https://containerlab.dev/manual/kinds/vr-ftosv)
* [Arista vEOS](https://containerlab.dev/manual/kinds/vr-veos)
* [Palo Alto PAN](https://containerlab.dev/manual/kinds/vr-pan)
* [IPInfusion OcNOS](https://containerlab.dev/manual/kinds/ipinfusion-ocnos)
* [Check Point Cloudguard](https://containerlab.dev/manual/kinds/checkpoint_cloudguard/)
* [Fortinet Fortigate](https://containerlab.dev/manual/kinds/fortinet_fortigate/)
* [Aruba AOS-CX](https://containerlab.dev/manual/kinds/vr-aoscx)
* [Huawei VRP](https://containerlab.dev/manual/kinds/huawei_vrp)
* [F5 BIG-IP VE](https://containerlab.dev/manual/kinds/f5_bigipve/)
* [OpenBSD](https://containerlab.dev/manual/kinds/openbsd)
* [FreeBSD](https://containerlab.dev/manual/kinds/freebsd)
* [OpenWRT](https://containerlab.dev/manual/kinds/openwrt/)

And, of course, containerlab is perfectly capable of wiring up arbitrary linux containers which can host your network applications, virtual functions or simply be a test client. With all that, containerlab provides a single IaaC interface to manage labs which can span contain all the needed variants of nodes:

<p align="center">
<img src="https://gitlab.com/rdodin/pics/-/wikis/uploads/bb8d9163f265dc827428097e6726d949/image.png" width="80%">
</p>

### VM-like container clusters

When a lab needs to model a VM made of multiple cooperating processes but the
implementation must stay container-only, a useful pattern is to create a
namespace-owner container and let application containers join its network
namespace.

In this model, the namespace-owner container acts as the VM boundary. It owns
the topology-facing interfaces, routes, NAT and port-forwarding rules. The
application containers run with `network-mode: container:<namespace-owner>` and
therefore share the same interfaces, loopback addresses, routes and port space,
similar to processes running inside one VM.

```yaml
topology:
  nodes:
    vm-ns:
      kind: linux
      image: localhost/vm-router:latest
      cmd: /usr/local/bin/vm-router-init
      sysctls:
        net.ipv4.ip_forward: "1"

    app-a:
      kind: linux
      image: localhost/app-a:latest
      network-mode: container:vm-ns

    app-b:
      kind: linux
      image: localhost/app-b:latest
      network-mode: container:vm-ns

    ceos:
      kind: ceos
      image: localhost/ceos:4.35.2F

  links:
    - endpoints: ["ceos:eth1", "vm-ns:eth1"]
```

The `vm-router-init` process can configure the shared namespace like a small VM
router or firewall:

```bash
ip addr add 10.0.0.2/31 dev eth1
ip route add default via 10.0.0.1

# outbound NAT for traffic leaving the VM-like namespace
iptables -t nat -A POSTROUTING -o eth1 -j MASQUERADE

# inbound PAT from the topology-facing interface to a service in the namespace
iptables -t nat -A PREROUTING -i eth1 -p tcp --dport 8080 \
  -j REDIRECT --to-ports 80
```

Because the application containers share one network namespace, they also share
one port space. Two applications cannot both bind `0.0.0.0:80` unless they are
configured with distinct addresses or ports. This is intentional for the VM-like
model: containerlab wires the namespace owner into the topology, while the
joined containers behave like processes inside that namespace.

This short clip briefly demonstrates containerlab features and explains its purpose:

[![vid](https://gitlab.com/rdodin/pics/-/wikis/uploads/35d954fd81d9594ffa5b6110cbc950f5/clab-clip-stillshot.png)](https://youtu.be/xdi7rwdJgkg)

## Features

* **IaaC approach**  
    Declarative way of defining the labs by means of the topology definition [`clab` files](https://containerlab.dev/manual/topo-def-file/).
* **Network Operating Systems centric**  
    Focus on containerized Network Operating Systems. The sophisticated startup requirements of various NOS containers are abstracted with [kinds](https://containerlab.dev/manual/kinds/) which allows the user to focus on the use cases, rather than infrastructure hurdles.
* **VM based nodes friendly**  
    With the [vrnetlab integration](https://containerlab.dev/manual/vrnetlab) it is possible to get the best of two worlds - running virtualized and containerized nodes alike with the same IaaC approach and workflows.
* **Multi-vendor and open**  
    Although being kick-started by Nokia engineers, containerlab doesn't take sides and supports NOSes from other vendors and opensource projects.
* **Lab orchestration**  
    Starting the containers and interconnecting them alone is already good, but containerlab packages even more features like managing lab lifecycle: [deploy](https://containerlab.dev/cmd/deploy), [destroy](https://containerlab.dev/cmd/destroy), [save](https://containerlab.dev/cmd/save), [inspect](https://containerlab.dev/cmd/inspect), [graph](https://containerlab.dev/cmd/graph) operations.
* **Systemd-capable container runtime controls**
    Container nodes can opt into init-friendly runtime settings such as cgroup namespace mode, PID namespace mode, tmpfs mounts, security options, shared memory sizing, and privileged mode. Docker and Podman runtimes honor these settings consistently for containers that need to run systemd or other PID 1 supervisors.
* **Scaled labs generator**  
    With [`generate`](https://containerlab.dev/cmd/generate) capabilities of containerlab it possible to define/launch CLOS-based topologies of arbitrary scale. Just say how many tiers you need and how big each tier is, the rest will be done in a split second.
* **Simplicity and convenience**  
    Starting from frictionless [installation](https://containerlab.dev/install/) and [upgrade](https://containerlab.dev/install#upgrade) capabilities and ranging to the behind-the-scenes [link wiring machinery](https://containerlab.dev/manual/network), containerlab does its best for you to enjoy the tool.
* **Fast**  
    Blazing fast way to create container based labs on any Linux system with Docker.
* **Automated TLS certificates provisioning**  
    The nodes which require TLS certs will get them automatically on boot.
* **Documentation is a first-class citizen**  
    We do not let our users guess by making a complete, concise and clean [documentation](https://containerlab.dev).
* **Lab catalog**  
   The "most-wanted" lab topologies are [documented and included](https://containerlab.dev/lab-examples/lab-examples/) with containerlab installation. Based on this cherry-picked selection you can start crafting the labs answering your needs.

## Use cases

* **Labs and demos**  
    Containerlab was meant to be a tool for provisioning networking labs built with containers. It is free, open and ubiquitous. No software apart from Docker is required!  
    As with any lab environment it allows the users to validate features, topologies, perform interop testing, datapath testing, etc.  
    It is also a perfect companion for your next demo. Deploy the lab fast, with all its configuration stored as a code -> destroy when done.
* **Testing and CI**  
    Because of the containerlab's single-binary packaging and code-based lab definition files, it was never that easy to spin up a test bed for CI. Gitlab CI, Github Actions and virtually any CI system will be able to spin up containerlab topologies in a single simple command.
* **Telemetry validation**  
    Coupling modern telemetry stacks with containerlab labs make a perfect fit for Telemetry use cases validation. Spin up a lab with containerized network functions with a telemetry on the side, and run comprehensive telemetry use cases.

Containerlab documentation is provided at <https://containerlab.dev>.
