# gooo-receipt-schema-migration

`gooo-receipt-schema-migration` is an independent, append-only migration
protocol for immutable receipt histories. It proves that a v2
`REGRESSION_REPAIR` parent can remain byte-for-byte unchanged while a v3
`CORRECTION_CHILD` owns the new outcome and causal fields.

The executable chain is fixed and source-bound:

```text
.gooo source → semantic IR → generated v2/v3 adapters + validator
            → scenario receipts → human report
```

The fixed denominator has exactly twelve cells. Foundation, Coherence, and
Regression each contain four cells. Driver, Outcome, and Guardrail each
contain four cells. The twelve cells are the two normal cases, three UNKNOWN
cases, and seven REFUTED cases declared in
`examples/receipt-schema-migration-v1/migration.gooo`.

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

The run emits `semantic-ir.json`, generated adapters and validator,
versioned scenario receipts, `scenario-receipts.json`, a machine-readable
`adoption-proposal.json`, `artifact-manifest.json`, and `human-report.md`.
The proposal is descriptive only: it does not write `meta-ontology-go` and
requires no cross-project gate. An immutable external release and its digest
may be supplied together as an optional, immutable input.

CI is the validation authority. It uses Go 1.27 and reports schema versions,
parent and child counts, exact accepted/unknown/refuted counts, immutable
parent writes, adapter operations, artifact digests, and build/test/conformance
wall time and peak RSS in the CI summary. Without an exact comparable
before/after pair, improvement is
`UNKNOWN`.
