# geth_compatibility

When Sonic moves to a new go-ethereum version, the compiler catches changed
signatures but not changed behavior. `geth_compatibility` writes a review file
with a row for every go-ethereum declaration that `sonicd` and `sonictool` can
reach and whose source differs between the go-ethereum version of a base
revision and the one of a head revision. Reviewers record a verdict in each row,
as [REVIEWING.md](REVIEWING.md) describes.

## Workflow

A review is a loop of three steps:

1. Run the tool. It writes a row for each finding to `review.md`.
2. Review the rows that need it. A coordinator splits them into groups by area,
   hands the groups to reviewers at once, merges the rows they return into
   `review.md`, and has the rows that claim a consensus impact or ask a
   question checked by second reviewers. [REVIEWING.md](REVIEWING.md)
   describes each step, for the coordinator and for the reviewers.
3. Run the tool again. It checks and regroups the rows.

The `todo` rows left are questions for humans. When the head changes, for
instance after a rebase or with a new go-ethereum version, run the loop again:
the tool keeps the reviewed rows whose findings are unchanged and reopens the
others, so only these need a reviewer.

## Running

```sh
cd scripts/geth_compatibility
go run . -head simon/amsterdam -base main
```

The head is any branch, tag or commit of the repository, and the base is
compared through its merge base with the head; `-base` defaults to
`origin/main`. Both must build from their committed files alone, so a module
replaced by a directory outside the repository cannot be loaded; neither needs
to contain the tool. The tool writes `review.md` in the current directory, or
the file `-review` names. A run takes about ten seconds and four GB of memory,
so run one at a time.

If the file exists, its rows are kept while the hashes of both sides still
match and the finding has not moved onto the consensus path. Other rows, for
instance after a rebase of the fork, are reopened as `todo` for the next
reviewer. A reopened row keeps a `live` or hardfork classification and its
note, prefixed with the verdict it had unless it was not reviewed yet. Rows
without a finding are dropped. Edit only the
Verdict, Consensus and Note cells; write a `|` in a note as `\|`.

## Findings

A finding is a difference in one of:

- a function or method reachable from the main packages, exported or not; the
  `init` functions of a package form one finding, `<package> func init`,
- a type, variable or constant referenced by reachable code, by the initializer
  of a package-level variable, or by a type, variable or constant these refer
  to, in any module,
- a package-level go-ethereum variable whose initializer calls a function, as it
  runs at startup; the variables named `_` of a package form one finding,
  `<package> var _`,
- a C, assembly or embedded file of a go-ethereum package that has a used
  declaration, or a file that its cgo preambles and C files include with
  quotes, directly or indirectly; other files below its directory, such as the
  tests of a bundled C library, are not compiled and do not count,
- the cgo preamble of a Go file of such a package, the comment above
  `import "C"` with its C code and `#cgo` directives,
- the version of a module, other than go-ethereum, that provides reachable code,
  including the initializers of package variables, or the value of a variable
  or constant that reachable code reads.

The Base and Head cells hold hashes of the two versions, `-` for a side that
lacks the declaration or does not import its package. In Go declarations,
blank lines outside raw string literals and comments other than `//go:embed`
directives are ignored, and constants are compared by value; files and cgo
preambles are compared as they are, comments included. A declaration includes
the imports it refers to, so that it changes with the package an import name
stands for.

## Verdicts

| Verdict    | Meaning |
|------------|---------|
| `todo`     | Not reviewed yet, or a question for a human: then the note starts with `OPEN:`. |
| `adopted`  | Behavior changes and Sonic accepts it. The note says why and what covers it. |
| `refactor` | No change in behavior. |
| `inactive` | The change only affects forks or features Sonic does not enable. |

## Consensus

The consensus path is the code whose results all nodes have to agree on. Its
roots, in `internal/analysis/consensus.go`, form and execute blocks, validate
events and write the genesis state. A finding is on the consensus path if code
reachable from these roots uses it, or the initializer of a package variable
this code reads or an init function writing that variable, also through the
map, slice or pointer it holds or in a function the init function calls,
by the rules of the main analysis with three restrictions, each documented
there:

- Skipped functions of Sonic are not entered, such as the import of genesis
  files that predate Sonic's networks.
- Cut go-ethereum symbols and packages, and variables of a cut type, are not
  entered: they are reachable, but cannot affect results, like go-ethereum's
  interpreter and its jump table, which only run for tracing, or logging and
  metrics.
- Types converted to an interface outside the consensus path implement the
  interfaces of Sonic and go-ethereum, and those of other modules only if they
  are of the same module, but not those of the standard library.

The tool stops if neither the base nor the head has a root, skip or cut, or if
the head lacks Sonic's upgrades type `opera.Upgrades`; update `analysis.Sonic`
then. List a symbol renamed between the two under both names.

The Consensus column records the reviewer's classification:

| Consensus | Meaning |
|-----------|---------|
| `live`    | The change can alter consensus results on hardforks that are already active. |
| hardfork  | The lowercase name of a field of `opera.Upgrades` on the head, such as `canto`: the change can alter consensus results only once that hardfork is active. |
| `no`      | It is on the consensus path, but cannot alter results. |
| *empty*   | On the consensus path, not classified yet. |
| `-`       | Off the consensus path. The tool writes it on every run. |

The tool stops on any other value.

Rows classified `live` or with a hardfork name form the first section of the
file, whatever their verdict: `live` first, then the hardforks by name. The
other rows follow in a section per verdict, `todo` first.

## Limits

- The tool compares source, not behavior. A verdict is a reviewer's judgement;
  replaying history with Bertha and the hook-liveness suite of the go-ethereum
  fork remain necessary.
- Reachability starts at the `main` and `init` functions of `./cmd/sonicd` and
  `./cmd/sonictool`; code reached only from tests is not included. A method
  called through an interface is reachable for every type implementing it that
  reachable code converts to an interface, or that is the type of an exported
  or embedded field or of an element of such a type.
- A value handed to a package that looks up methods by reflection, such as the
  RPC server of go-ethereum, makes the exported methods of its type reachable,
  together with those of its exported and embedded fields, elements and method
  results. Values reaching such a package indirectly are not covered.
- `go:linkname`, changes of the Go toolchain, and the content of dependency
  modules are not covered; dependency modules are compared by version only.
- Each side only knows the go-ethereum packages it imports. If only the head
  imports a package, its reachable declarations get the Base `-` even if they
  did not change.
- The consensus path over-approximates through interfaces, but does not follow
  function values that are created outside of it and called on it, unless the
  initializer of a package variable it reads, or an init function writing that
  variable, refers to them.
- Writes at startup are followed from init functions and the functions they
  call, but not from callbacks they hand to other functions, such as
  `sync.Once.Do`, nor from the initializers of other package variables.
