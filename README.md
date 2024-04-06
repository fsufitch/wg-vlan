# `wg-vlan` &ndash; Simple Wireguard VLAN Configuration

[Wireguard](https://www.wireguard.com/) is a fast, simple, powerful VPN. Its configuration/management for can be a little complicated to get going smoothly... Or at least they were for me.

Since my main use case is creating a VLAN organized around a central server, I made this tool for managing the requisite configurations.

## Build

Use Go 1.20+.

```
go build -o bin/wg-vlan ./
```

Pre-built binaries in Github and more structured versioning coming at a future point, maybe.

## Usage

> The `wg-vlan` command line includes `-h/--help` at any point. Use them to discover all the available options.

`wg-vlan` stores its own configuration separately from Wireguard's INI files, in a YAML file. Create a new configuration for your VLAN like this:

```bash
$ wg-vlan init -f my_vlan.yaml --endpoint my.vlan.example.com:51820
```

Then, add new clients to your VLAN (and generate their configuration) like this:

```bash
$ wg-vlan add -f my_vlan.yaml -n alice
$ wg-vlan add -f my_vlan.yaml -n bob
```

This results in a `my_vlan.yaml` which looks like:

```yaml
public_endpoint: my.vlan.example.com:51820
keep_alive: 25
server:
  peer_name: wg-vlan
  listen_port: 51820
  network: 10.20.30.1/24
  private_key: tcTUw/vk49fQ/XO361DzI3vc0yfmwdsizZL2QkjzxJM=
  public_key: rVY73e/8Z1LJk4cXdt9BabbobNJVd/nrEnjUka3v1kY=
clients:
- peer_name: alice
  network: 10.20.30.2
  private_key: dINRoLcey+mdrBIt0xHUoaNCjeMFl3ygahnL3RnNtX0=
  public_key: bsjOPLot8wTuF6BR+7gs6osK2KClyQgasp2LXbOX9TA=
  preshared_key: P6xB5nPjyqKwbEUrqOYrKiupBwOzDsqy1Zbjs4GT1u4=
- peer_name: bob
  network: 10.20.30.3
  private_key: FrCJAsWzwZaR4HVy37HDKBoLt4SCla+Y2EcjefKVHtU=
  public_key: dtQhdaRhD94079Wgv3vQ0QmZUVbjMl9pJfR9YF98FS8=
  preshared_key: W3fGT6adyOXed8nKx3ubx86AWIaTeOWrFg9Hxjrm98Y=
```

Clients can also be added using only a public key, if they generated their own.

Using this YAML configuration, `wg-vlan` can then export the relevant INI-format configuration for the VLAN "server" peer:

```ini
$ wg-vlan export -f my_vlan.yaml -s
# VLAN Server: wg-vlan
[Interface]
Address    = 10.20.30.1/24
ListenPort = 51820
PrivateKey = tcTUw/vk49fQ/XO361DzI3vc0yfmwdsizZL2QkjzxJM=

# VLAN Client: alice
[Peer]
AllowedIPs          = 10.20.30.2/32
PublicKey           = bsjOPLot8wTuF6BR+7gs6osK2KClyQgasp2LXbOX9TA=
PresharedKey        = P6xB5nPjyqKwbEUrqOYrKiupBwOzDsqy1Zbjs4GT1u4=
PersistentKeepalive = 25

# VLAN Client: bob
[Peer]
AllowedIPs          = 10.20.30.3/32
PublicKey           = dtQhdaRhD94079Wgv3vQ0QmZUVbjMl9pJfR9YF98FS8=
PresharedKey        = W3fGT6adyOXed8nKx3ubx86AWIaTeOWrFg9Hxjrm98Y=
PersistentKeepalive = 25
```

It can also generate the INI configuration for "client" peers:

```ini
$ wg-vlan export -f my_vlan.yaml -c alice
# VLAN Client: alice
[Interface]
Address    = 10.20.30.2/32
PrivateKey = dINRoLcey+mdrBIt0xHUoaNCjeMFl3ygahnL3RnNtX0=

# VLAN Server: wg-vlan
[Peer]
Endpoint            = my.vlan.example.com:51820
AllowedIPs          = 10.20.30.1/24
PublicKey           = rVY73e/8Z1LJk4cXdt9BabbobNJVd/nrEnjUka3v1kY=
PresharedKey        = P6xB5nPjyqKwbEUrqOYrKiupBwOzDsqy1Zbjs4GT1u4=
PersistentKeepalive = 25
```

You can also export these configurations as QR codes, using `--format qr`.

Use these files in the Wireguard configuration of the respective relevant computers, and you will have a VLAN-like network!

## YAML Configuration Schema

Some notes:

   * "peer_name" keys are purely for `wg-vlan` use; Wireguard itself uses no peer names.
   * "network" keys are IPv4 and infer a netmask of "/24" unless otherwise specified.
   * "public_key" is not required for the server; it can be inferred from the private key
   * "private_key" is not required for clients; however, `wg-vlan` cannot export configs for clients lacking a private key

```yaml
# The endpoint clients are told to connect to
public_endpoint: my.vlan.example.com:51820

# The keep-alive interval that peers use for checking in on each other
keep_alive: 25

server:
  peer_name: wg-vlan
  listen_port: 51820
  network: 10.20.30.1/24  # defines both the subnet o fthe VLAN and the IP of the server itself (within the subnet)
  private_key: tcTUw/vk49fQ/XO361DzI3vc0yfmwdsizZL2QkjzxJM=
  public_key: rVY73e/8Z1LJk4cXdt9BabbobNJVd/nrEnjUka3v1kY=
  extra:
    # Key/Value overrides for exported server configs; example:
    MTU: 1234

clients:
  - peer_name: alice
    network: 10.20.30.2  # must be within the subnet defined in "server"
    private_key: dINRoLcey+mdrBIt0xHUoaNCjeMFl3ygahnL3RnNtX0=
    public_key: bsjOPLot8wTuF6BR+7gs6osK2KClyQgasp2LXbOX9TA=
    preshared_key: P6xB5nPjyqKwbEUrqOYrKiupBwOzDsqy1Zbjs4GT1u4=
    extra: 
      # Key/Value overrides for exported server configs; example:
      MTU: 1234