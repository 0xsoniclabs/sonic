This tool is a tool provided by [golang/stress](https://pkg.go.dev/golang.org/x/tools/cmd/stress#section-documentation) with a slight modification that it runs with `-race`.

1) Build the tool run `go build flaky_stress.go`  - alternatively use the official tool installed with 

    `go install golang.org/x/tools/cmd/stress@latest`

2) Build the tests binaries:

     `go test ./my/tests/folder/ -o /path/to/test/dump/folder`

      This command will run the tests and dump binaries at the given path.

      Please do not pollute the repo, write build files in the build folder.


3) Run 

    `flaky_stress /path/to/test/dump/folder/* -test.run=specific_test_name`

     or 

    `flaky_stress /path/to/test/dump/folder/my_package.test -test.r un=specific_test_name` 

    and expect an output similar to:

    ```
    5s: 0 runs so far, 0 failures, 8 active
    10s: 1 runs so far, 0 failures, 8 active
    15s: 5 runs so far, 0 failures, 8 active
    20s: 7 runs so far, 0 failures, 8 active
    ```
    Note: 8 active is the number of tests running in parallel.

    As time moves on more test and iterations will be executed. As this tool will repeatedly execute the tests as long as it runs, it should replay tests in a CPU intensive scenario stressing waits/delays and reporting on tests that fail.

