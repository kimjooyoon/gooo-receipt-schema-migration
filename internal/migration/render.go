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
	if report.Migration != nil {
		migration := report.Migration
		fmt.Fprintf(&b, "## Denominator migration\n\n- version: %s → %s\n- ADD: %d\n- RETIRE: %d\n- SPLIT: %d\n- stage counts before: %v\n- stage counts after: %v\n- stage delta: %v\n- role counts before: %v\n- role counts after: %v\n- role delta: %v\n\n", migration.FromVersion, migration.ToVersion, migration.Added, migration.Retired, migration.Split, migration.StageCountsBefore, migration.StageCountsAfter, migration.StageDelta, migration.RoleCountsBefore, migration.RoleCountsAfter, migration.RoleDelta)
	}
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
	if report.DevelopmentProvenance != nil {
		provenance := report.DevelopmentProvenance
		b.WriteString("\n## Development provenance\n\n")
		fmt.Fprintf(&b, "- event: `%s`\n- class: `%s`\n- purpose: `%s`\n- policy state: `%s`\n- local Go commands (gofmt/build/test/vet): `0/0/0/0`\n- local VM Guardian harness executions: `%d`\n- development policy deviations: `%d`\n- failed packaging attempts: `%d`\n- failed packaging reason: `%s`\n- release asset used for failed attempt: `%t`\n- final archive source: `%s`\n- final archive size_bytes: `%d`\n- final archive SHA-256: `%s`\n- reset/delete/rewrite: `%t`\n- mutation policy: `%s`\n- product/runtime authority repository writes: `%d`\n- product/runtime authority local test executions: `%d`\n- remaining validation: `%s`\n\nThis is a recorded development-policy deviation. The event is append-only; it is not reset, deleted, or rewritten. The local VM execution is intentionally separate from product/runtime authority, and all remaining validation is GitHub Actions only.\n\n", provenance.EventID, provenance.Class, provenance.Purpose, provenance.PolicyState, provenance.DevelopmentLocalVMHarnessExecutions, provenance.DevelopmentPolicyDeviationCount, provenance.FailedPackagingAttempts, provenance.FailedPackagingReason, provenance.ReleaseAssetUsed, provenance.FinalArchiveSource, provenance.FinalSizeBytes, provenance.FinalSHA256, provenance.ResetDeleteRewrite, provenance.EventMutationPolicy, provenance.ProductRuntimeAuthority.RepositoryWrites, provenance.ProductRuntimeAuthority.LocalTestExecutions, provenance.RemainingValidationPolicy)
	}
	if report.GuardianHarness != nil {
		b.WriteString("\n## Actual Guardian harness\n\n")
		fixtureRef := fmt.Sprintf("%s@%s", report.GuardianHarness.Fixture.Repository, report.GuardianHarness.Fixture.Commit)
		fixtureManifest := report.GuardianHarness.Fixture.ManifestPath
		if report.GuardianHarness.FixtureV3 != nil {
			fixtureRef = fmt.Sprintf("%s@%s..%s", report.GuardianHarness.FixtureV3.Repository, report.GuardianHarness.FixtureV3.BaseCommit, report.GuardianHarness.FixtureV3.HeadCommit)
			fixtureManifest = report.GuardianHarness.FixtureV3.ManifestPath
		}
		fmt.Fprintf(&b, "- fixture: `%s`\n- fixture manifest: `%s`\n- CLOSED: %d\n- UNKNOWN: %d\n- REFUTED: %d\n\n| case | expected | observed | Guardian | stage | step | reason | next_operation |\n|---|---|---|---|---|---|---|---|\n", fixtureRef, fixtureManifest, report.GuardianHarness.Summary.ClosedCount, report.GuardianHarness.Summary.UnknownCount, report.GuardianHarness.Summary.RefutedCount)
		for _, result := range report.GuardianHarness.Results {
			fmt.Fprintf(&b, "| %s | %s | %s | %s | %s | %s | %s | %s |\n", result.ID, result.ExpectedState, result.State, result.GuardianDecision, result.Stage, result.Step, result.Reason, result.NextOperation)
		}
	}
	b.WriteString("\n## Improvement\n\n`UNKNOWN`: exact comparable before/after pair was not provided.\n")
	return b.String()
}
