# Acceptance matrix

The fixed denominator is the twelve rows below. The stage axis and role axis
are independent counts: Foundation/Coherence/Regression are 4/4/4, and
Driver/Outcome/Guardrail are 4/4/4.

| cell | stage | role | minimum case | expected state |
|---|---|---|---|---|
| NORMAL_V2_PARENT_V3_CHILD | FOUNDATION | DRIVER | v2 parent without outcome plus v3 child with `parent_outcome=REFUTED_INCOMPLETE_PROPAGATION` | ACCEPTED |
| NORMAL_REPLAY_DETERMINISTIC | FOUNDATION | OUTCOME | identical replay bytes and digests | ACCEPTED |
| UNKNOWN_PARENT_DIGEST_MISSING | FOUNDATION | GUARDRAIL | parent digest missing | UNKNOWN |
| UNKNOWN_CHILD_DIGEST_STALE | COHERENCE | OUTCOME | child digest stale | UNKNOWN |
| UNKNOWN_UNSUPPORTED_FUTURE_SCHEMA | FOUNDATION | DRIVER | unsupported future schema | UNKNOWN |
| REFUTED_CHILD_REWRITES_PARENT | COHERENCE | DRIVER | child attempts to rewrite parent | REFUTED |
| REFUTED_FUTURE_FIELD_IN_OLD_SCHEMA | COHERENCE | GUARDRAIL | v3-only field appears in v2 parent | REFUTED |
| REFUTED_PARENT_DIGEST_MISMATCH | COHERENCE | OUTCOME | child names a different parent digest | REFUTED |
| REFUTED_SECOND_CHILD_CARDINALITY_ONE | REGRESSION | DRIVER | second child under cardinality one | REFUTED |
| REFUTED_SELF_ATTESTATION | REGRESSION | GUARDRAIL | child attests itself | REFUTED |
| REFUTED_DENOMINATOR_DOWNGRADE | REGRESSION | GUARDRAIL | child lowers fixed denominator | REFUTED |
| REFUTED_UNKNOWN_PROMOTED_FIXED_POINT | REGRESSION | OUTCOME | UNKNOWN is promoted to FIXED_POINT | REFUTED |

The two normal cases are accepted, the three evidence gaps remain UNKNOWN,
and the seven contradictions are REFUTED. These are exact state counts, not a
single aggregate judgment. `REFUTED > UNKNOWN > CLOSED` is applied before
reporting a result.
