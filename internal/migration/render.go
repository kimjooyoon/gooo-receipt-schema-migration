package migration

import (
	"fmt"
	"strings"
)

func WriteText(path, value string) error {
	return osWriteFile(path, []byte(value))
}

func RenderReport(report Report) string {
	var b strings.Builder
	b.WriteString("# Receipt schema migration conformance\n\n")
	b.WriteString("Pipeline: `.gooo source → semantic IR → generated version adapters/validator → scenario receipts → human report`\n\n")
	b.WriteString("## Fixed denominator\n\n")
	fmt.Fprintf(&b, "- cells: %d\n- FOUNDATION: %d\n- COHERENCE: %d\n- REGRESSION: %d\n- DRIVER: %d\n- OUTCOME: %d\n- GUARDRAIL: %d\n", report.FixedDenominator, report.StageCounts["FOUNDATION"], report.StageCounts["COHERENCE"], report.StageCounts["REGRESSION"], report.RoleCounts["DRIVER"], report.RoleCounts["OUTCOME"], report.RoleCounts["GUARDRAIL"])
	b.WriteString("\n\n## Exact state counts\n\n")
	fmt.Fprintf(&b, "- parent receipts: %d\n- child receipts: %d\n- accepted: %d\n- unknown: %d\n- refuted: %d\n- immutable parent writes: %d\n", report.Summary.ParentReceiptCount, report.Summary.ChildReceiptCount, report.Summary.AcceptedCount, report.Summary.UnknownCount, report.Summary.RefutedCount, report.Summary.ImmutableParentWrites)
	b.WriteString("\nResolution precedence: `REFUTED > UNKNOWN > CLOSED`.\n\n")
	b.WriteString("## Scenarios\n\n| scenario | stage | role | expected | observed | reason | parent digest | child digests |\n|---|---|---|---|---|---|---|---|\n")
	for _, scenario := range report.Scenarios {
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s | %s | %s | %s |\n", scenario.ScenarioID, scenario.Stage, scenario.Role, scenario.ExpectedState, scenario.State, scenario.ObservedReason, scenario.ParentDigest, join(scenario.ChildDigests))
	}
	b.WriteString("\n## Adapter operations\n\n")
	for _, operation := range report.AdapterOperations {
		fmt.Fprintf(&b, "- `%s`\n", operation)
	}
	b.WriteString("\n## Artifact digests\n\n| path | digest | size_bytes |\n|---|---|---|\n")
	for _, artifact := range report.ArtifactDigests {
		fmt.Fprintf(&b, "| %s | %s | %d |\n", artifact.Path, artifact.Digest, artifact.SizeBytes)
	}
	b.WriteString("\n## Improvement\n\n`UNKNOWN`: exact comparable before/after pair was not provided.\n")
	return b.String()
}

