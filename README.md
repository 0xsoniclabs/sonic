# Sonic

EVM-compatible chain secured by the Lachesis consensus algorithm.

## Building the source

Building Sonic requires both a Go (version 1.21 or later) and a C compiler. You can install
them using your favourite package manager. Once the dependencies are installed, run:

```sh
make all
```
The build outputs are ```build/sonicd``` and ```build/sonictool``` executables.

## Initialization of the Sonic Database

You will need a genesis file to join a network. See [lachesis_launch](https://github.com/Fantom-foundation/lachesis_launch) for details on obtaining one. Once you have a genesis file, initialize the DB:

```sh
sonictool --datadir=<target DB path> genesis <path to the genesis file>
```

## Running `sonicd`

Going through all the possible command line flags is out of scope here,
but we've enumerated a few common parameter combos to get you up to speed quickly
on how you can run your own `sonicd` instance.

### Launching a network

To launch a `sonicd` read-only (non-validator) node for network specified by the genesis file:

```sh
sonicd --datadir=<DB path>
```

### Configuration

As an alternative to passing the numerous flags to the `sonicd` binary, you can also pass a
configuration file via:

```sh
sonicd --datadir=<DB path> --config /path/to/your/config.toml
```

To get an idea of what the file should look like you can use the `dumpconfig` subcommand to
export the default configuration:

```sh
sonictool --datadir=<DB path> dumpconfig
```

### Validator

To create a new validator private key:

```sh
sonictool --datadir=<DB path> validator new
```

To launch a validator, use the `--validator.id` and `--validator.pubkey` flags. See the [Fantom Documentation](https://docs.fantom.foundation) for details on obtaining a validator ID and registering your initial stake.

```sh
sonicd --datadir=<DB path> --validator.id=YOUR_ID --validator.pubkey=0xYOUR_PUBKEY
```

`sonicd` will prompt for a password to decrypt your validator private key. Optionally, use `--validator.password` to specify a password file.

#### Participation in discovery

Optionally, specify your public IP to improve connectivity. Ensure your TCP/UDP p2p port (5050 by default) is open:

```sh
sonicd --datadir=<DB path> --nat=extip:1.2.3.4
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines. Please also review our [Code of Conduct](CODE_OF_CONDUCT.md).
