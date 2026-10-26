# Golden files of the reference

The flags and TOML keys of the reference release, v2.2.5, against which the
O4 tests of `config_golden_test.go` check the node (rpc/PLAN.md, O4).

- `flags-2.2.5.txt`: the flag names of `sonicd --help`, one per line.
- `config-2.2.5.toml`: `sonicd --fakenet 1/1 --dump-config` on a fake genesis
  made by `sonictool genesis fake 1`, with `DataDir` emptied.

Both were produced by the v2.2.5 binaries. Regenerate them only for a new
reference release.
