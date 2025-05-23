# Local development

## Running the client locally

### Single instance

To start a single client instance locally please use the `make localnode-bin-start`
command. The script will compile the client, perform genesis, and start a local
client instance.

### Multiple client instances

Multiple clients can be run using locally built binaries. This method can be
useful when there are frequent changes to the code and rebuilds are needed.
However, it requires starting each node separately.

```bash
# Build the binaries and generate configuration for the clients.
$ make localnet-bin-init
# Start the sidecars instances.
$ make localnet-bin-sidecars-start
# Start a single instance of the client. Needs to be called for each of the
# the clients. At least 2/3 (so 3 out of 4) of clients are needed to produce
# blocks.
$ make localnet-bin-start
# Remove the `build` and `.localnet` directories.
$ make localnet-bin-clean
```

NOTE: The connect sidecar defaults to use the node0 grpc API, therefore in
order for it to work properly, node0 needs to be running.

For the Ethereum sidecar to startup properly, an Ethereum RPC provider
needs to be set. This can be done using a .env file located at the root of the
repository:
```
ETH_SIDECAR_RPC_PROVIDER=wss://eth-sepolia.g.alchemy.com/v2/<YOUR_ID>
```

Ethereum sidecar can be optimized against different RPC providers and accepts
the following flags:

- '--ethereum-sidecar.server.batch-size' - size of the block batch for fallback AssetsLocked events lookup
- '--ethereum-sidecar.server.requests-per-minute' - requests per minute for an Ethereum RPC provider

The network consists of four clients connected to each other. All the data
generated by the clients is stored in the `.localnet` directory.
