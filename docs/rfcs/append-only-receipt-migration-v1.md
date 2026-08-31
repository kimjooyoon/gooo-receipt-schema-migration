# Append-only receipt schema migration v1

## Problem

An immutable v2 `REGRESSION_REPAIR` parent may already have been consumed
without an `outcome`. A later v3 `CORRECTION_CHILD` must not make that old
parent invalid by requiring a field that v2 never owned.

## Ownership

The v2 parent owns its schema identity, repair declaration, fixed denominator,
immutability assertion, creator, and digest. It must not gain `outcome`,
`parent_outcome`, `cause_code`, `causal_chain`, `next_operation`, or an
attestation field after consumption.

The v3 child owns `parent_outcome=REFUTED_INCOMPLETE_PROPAGATION`, its own
`outcome`, `cause_code`, `causal_chain`, `next_operation`, child digest, and
the exact parent receipt ID/digest it extends. This is an append-only causal
edge, not a parent rewrite.

## Resolution

The validator resolves `REFUTED > UNKNOWN > CLOSED`. Missing parent digest,
stale child digest, and an unsupported future schema are evidence gaps and
remain UNKNOWN with the six required fields. A contradictory parent/child
lineage, future field in a v2 parent, second child under cardinality one,
self-attestation, denominator downgrade, or UNKNOWN promoted to fixed point is
REFUTED.

## Adoption boundary

The generated adoption proposal names the old/new field ownership, exact
semantic changes, expected protected paths, acceptance cases, and rollback or
refutation conditions. It has `repository_writes=0`,
`local_test_executions=0`, and `cross_project_required_gates=0`; it is not a
patch to the target project.
