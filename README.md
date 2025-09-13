# netshed

A command-line tool to manage network interfaces and NAT masquerading on Linux.

I had been looking for a tool to set up temporary network interfaces for testing environments that would be automatically cleared on reboot. Instead of executing commands or shell scripts each time, I wanted to declaratively describe test networks using configuration files.


## Features

- Create and delete bridge interfaces
- Create and delete dummy interfaces
- Set up NAT masquerading similar to libvirt NAT network
- Configure network interfaces through YAML files

## Requirements

- Linux
- Root privileges (sudo)
- nftables

## Installation

Using go install:
```bash
$ go install github.com/zinrai/netshed@latest
```

Build from source:
```bash
$ go build -o netshed cmd/netshed/main.go
```

## Usage

Create network interfaces:
```bash
$ sudo netshed create -config network.yaml
```

Delete network interfaces:
```bash
$ sudo netshed delete -config network.yaml
```

## Configuration Examples

Bridge interface with NAT masquerade:
```yaml
networks:
  - name: "vm0"
    type: "bridge"
    subnet: "192.168.100.0/24"
    gateway: "192.168.100.1/24"
    masquerade: true
```

Internal bridge interface:
```yaml
networks:
  - name: "internal0"
    type: "bridge"
    subnet: "192.168.200.0/24"
    gateway: "192.168.200.1/24"
    masquerade: false
```

Dummy interface:
```yaml
networks:
  - name: "dummy0"
    type: "dummy"
    address: "10.0.0.1/24"
```

## License

This project is licensed under the [MIT License](./LICENSE).
