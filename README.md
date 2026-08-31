# gooo-receipt-schema-migration

`gooo-receipt-schema-migration` is an independent, append-only migration
protocol for immutable receipt histories. It proves that a v2
`REGRESSION_REPAIR` parent can remain byte-for-byte unchanged while a v3
`CORRECTION_CHILD` owns the new outcome and causal fields, and v2 of this
migration protocol adds an executable audit of the real Guardian workflow.

The executable chain is fixed and source-bound:

```text
.gooo source → semantic IR → generated v2/v3 adapters + validator
            → generated executable scenarios → actual Guardian harness
            → report/proposal
```

Migration v1 remains fixed at exactly twelve cells: Foundation, Coherence, and
Regression are 4/4/4, and Driver, Outcome, and Guardrail are 4/4/4. Migration
v2 preserves those twelve cells and adds exactly four cells:
`BASE_CONTROLLED_GUARDIAN_EXECUTION`, `FEATURE_PR_VARIABLE_LIVENESS`,
`PASS_ARTIFACT_DIGEST_PROPAGATION`, and `REFERENCE_ERROR_FAIL_CLOSED`.
The v2 exact stage balance is 6/5/5 and role balance is 5/5/6. Its migration
record is ADD=4, RETIRE=0, SPLIT=0; no score or percentage is emitted.

Resolution precedence is `REFUTED > UNKNOWN > CLOSED`. UNKNOWN always carries
exactly `stage`, `step`, `reason`, `unknown_class`, `next_operation`, and
`blocked_by`. No denominator reduction, parent rewrite, self-attestation, or
promotion of UNKNOWN to a fixed point is accepted.

The command writes only to an absent or empty caller-owned output directory:

```sh
go run ./cmd/gooo-receipt-schema-migration run \
  --source examples/receipt-schema-migration-v1/migration.gooo \
  --contract contracts/receipt-migration-denominator-v1.json \
  --out /tmp/gooo-receipt-schema-migration-run
```

Migration v2 uses the parallel source and contract at
`examples/receipt-schema-migration-v2/migration.gooo` and
`contracts/receipt-migration-denominator-v2.json`. It emits
`generated/guardian-harness-cases.json`; CI executes
`scripts/run-guardian-harness.js` against the pinned
`meta-ontology-go@7f45792` workflow/Guardian blobs in caller-owned temporary
space, then attaches `guardian-harness-report.json` to the report and
proposal with `attach-harness`.

Every run emits `semantic-ir.json`, generated adapters and validator, versioned
scenario receipts, a machine-readable `adoption-proposal.json`,
`artifact-manifest.json`, and `human-report.md`. The v2 proposal includes the
exact variable lifetime/ownership rules, protected paths, executable cases,
rollback/refutation conditions, and upstream blob digests. It never writes
`meta-ontology-go`.
The proposal is descriptive only: it does not write `meta-ontology-go` and
requires no cross-project gate. An immutable external release and its digest
may be supplied together as an optional, immutable input.

CI is the validation authority. It uses Go 1.27 and reports schema versions,
parent and child counts, exact accepted/unknown/refuted counts, immutable
parent writes, adapter operations, artifact digests, and build/test/conformance
wall time and peak RSS in the CI summary. Without an exact comparable
before/after pair, improvement is
`UNKNOWN`.
