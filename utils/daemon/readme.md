# Sonic Daemon Systemd Support

This package adds support to daemonize `sonicd` using systemd tools. It provides configuration and scripts to run `sonicd` as a background service managed by systemd.

## How to Run as a User

To run `sonicd` you can use the provided Makefile to install the service files and scripts:

```sh
make install
```

The sonic database will be located at `~/.local/state/sonicd-test/`, it needs to
be initialized before first use of the daemon: 
```sh
~/.local/bin/sonictool -datadir ~/.local/state/sonicd-test/ genesis fake 1
```

## How to Read the Log

To view logs for the `sonicd` service, use `journalctl`:

```sh
journalctl --user-unit=sonicd -f
```

This will show real-time logs from the running service.

## Makefile Install

This will copy the necessary files to the appropriate locations for your system.
