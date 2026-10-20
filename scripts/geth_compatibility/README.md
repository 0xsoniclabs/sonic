# geth_compatibility

When Sonic moves to a new go-ethereum version, the compiler catches changed
signatures but not changed behavior. `geth_compatibility` lists every go-ethereum
declaration that `sonicd` and `sonictool` can reach and whose source differs
between the go-ethereum version of the base revision and the one of the head.
It fails until every difference has a verdict in `review.md`, and every adopted
difference on the consensus path says whether it can alter consensus results.

If go-ethereum did not change, it exits successfully without loading anything,
so it is cheap to run on every pull request.

## Running a review

Run the tool from a Sonic checkout. It compares any two Sonic revisions,
branches, tags or commits: the head, and the merge base of the head with the
base. Both must build, but neither needs to contain the tool. For instance:

```sh
gc() {
  go run -C scripts/geth_compatibility . -head simon/amsterdam -base main \
    -review ~/amsterdam/scripts/geth_compatibility/review.md "$@"
}

gc -prepare /tmp/round-1          # sort the findings into groups for reviewers
gc -apply /tmp/round-1            # record what the reviewers filled in
gc -prepare /tmp/round-2 -verify  # a second look at critical rows and questions
gc -apply /tmp/round-2
```

With `-head`, the tool reads `scripts/geth_compatibility/review.md` of the head
revision, and `-write` and `-apply` need `-review` to name the file to write,
here the one of a worktree of the head, where you commit it.

After each `-prepare`, give every group it lists to its own reviewer, with the
prompt it prints: a subagent of any coding agent, or a person. Reviewers read
sources and fill in the rows of their group, nothing else. `-apply` checks the
rows and records those the reviewers changed in the review file. It skips a file
with an invalid row and says why; fix the file and apply again, which leaves the
rows recorded before as they are. It also skips a row that changed in the review
file since `-prepare`, for instance in a later round, and names it. If the
summary still counts findings `to review`, prepare a new directory for them. The
remaining `todo` rows are the open questions for humans.

Before you start, check that the facts about Sonic in `brief.md` still hold,
such as which upgrades are live. Each run takes about ten seconds and four GB
of memory, so run one at a time. Without `-head`, the tool checks the working
tree, and `-base` defaults to `origin/main`.

To look at findings yourself, `-only <regexp>` selects them by symbol, `-diff`
shows the change, and `-why` the path through which Sonic reaches it. You can
also write verdicts straight into the cells of the review file; `-write` adds a
`todo` row for every finding.

```sh
gc -only '^core func stateTransition\.' -diff -why
```

Commit the review file together with the go-ethereum upgrade. The tool passes
once no row is `todo` and no `adopted` row on the consensus path lacks a
classification. For calibration: `simon/amsterdam` against `main`, which moved
to go-ethereum 1.17.8, has 841 findings in 14 groups. Its reviewers took about
15 minutes, and the second look caught two consensus regressions.

## Findings

A finding is a difference in one of:

- a function or method reachable from the main packages, exported or not,
- a type, variable or constant referenced by reachable code, by the initializer
  of a package-level variable, or by a type, variable or constant these refer
  to, in any module,
- a package-level go-ethereum variable whose initializer calls a function, as it
  runs at startup; the variables named `_` of a package form one finding,
  `<package> var _`,
- a C, assembly or embedded file of a go-ethereum package with reachable code,
- the version of a module, other than go-ethereum, that provides reachable code.

Its status is `changed`, `added` (reached only through the new version) or
`removed` (reached only through the old one). Comments and blank lines outside
raw string literals are ignored, and constants are compared by value. A
declaration includes the imports it refers to, so that it changes with the
package an import name stands for. `SONIC` marks symbols Sonic refers to
directly.

## Verdicts

| Verdict    | Meaning |
|------------|---------|
| `todo`     | Not reviewed yet, or a question for a human: then the note starts with `OPEN:`. |
| `adopted`  | Behavior changes and Sonic accepts it. The note says why and what covers it. |
| `refactor` | No change in behavior. |
| `inactive` | The change only affects forks or features Sonic does not enable. |

`review.md` starts with the consensus critical findings, see below, followed by
a section per verdict, in this order, so the open findings come first. Each
holds a table with a row per finding. Every table has the columns Verdict,
Consensus, Symbol, Base, Head and Note, and no blank lines; the tool reads their
rows and ignores the text around them. `-write` and `-apply` regroup the rows.
A `|` in a note is written as `\|`.

Every verdict other than `todo` needs a note. A row is bound to the hashes of
both sides, so if the head changes again, for instance after a rebase of the
fork, the row is reopened as `todo` and its note is kept for the next reviewer,
prefixed with the verdict it had.

## Consensus

The consensus path is the code whose results all nodes have to agree on. Its
roots, listed in `consensus.go`, form and execute blocks, validate events and
write the genesis state. A finding is on the consensus path if code reachable
from these roots uses it, by the rules of the main analysis with three
restrictions, each documented in `consensus.go`:

- Skipped functions of Sonic are not entered, such as the import of genesis
  files that predate Sonic's networks.
- Cut go-ethereum symbols and packages are not entered: they are reachable,
  but cannot affect results, like go-ethereum's interpreter, which only runs
  for tracing, or logging and metrics.
- Types converted to an interface outside the consensus path, for instance
  while Sonic constructs its services, implement the interfaces of Sonic and
  go-ethereum, and those of other modules only if they are of the same module.
  They do not implement the interfaces of the standard library: code calls
  `fmt.Stringer` or `io.Closer` on all kinds of values.

The tool stops if the head lacks a root, skip or cut, for instance after
go-ethereum renamed a cut symbol.

The Consensus column records the reviewer's classification:

| Consensus | Meaning |
|-----------|---------|
| `live`    | The change can alter consensus results on hardforks that are already active. The note says how Sonic accepts or compensates for it. |
| hardfork  | The lowercase name of a hardfork, such as `canto`: the change can alter consensus results only once that hardfork is active. |
| `no`      | It is on the consensus path, but cannot alter results. |
| *empty*   | Not classified yet. Needed for `adopted` findings on the consensus path. |
| `-`       | Off the consensus path. The tool writes it and recomputes it on every run. |

A finding off the consensus path can be classified too, for instance if a call
missing on the path is the problem. Findings classified `live` or with a
hardfork name form the first section of `review.md`, whatever their verdict, so
that it lists what an upgrade changes in consensus: `live` first, then the
hardforks by name, which is the order in which Sonic names them (Allegro, Brio,
Canto, Dolce). The summary counts them per classification, so a misspelt
hardfork name shows up as a classification of its own. `-why` also shows how the consensus path
reaches a finding, and the listing marks findings on it with `path`.

## Limits

- The check compares source, not behavior. A verdict is a reviewer's judgement;
  replaying history with Bertha and the hook-liveness suite of the go-ethereum
  fork remain necessary.
- Reachability starts at the `main` and `init` functions of `./cmd/sonicd` and
  `./cmd/sonictool`; code reached only from tests is not included. A method
  called through an interface in reachable code is reachable for every type
  implementing the interface that is converted to an interface in reachable code,
  or that is the type of an exported field or element of such a type.
- A value handed to a package that looks up methods by reflection, such as the
  RPC server of go-ethereum, makes the exported methods of its type reachable,
  together with those of its exported fields, elements and method results. A
  value is handed to a package if it is passed to one of its functions or stored
  in one of its structs; values reaching such a package indirectly are not
  covered. `-why` shows which rule makes a finding reachable.
- `go:linkname`, changes of the Go toolchain, and the content of dependency
  modules are not covered; dependency modules are compared by version only.
- The consensus path over-approximates through interfaces, but does not follow
  function values that are created outside of it and called on it. The
  classification of a finding is a reviewer's judgement, like its verdict.
