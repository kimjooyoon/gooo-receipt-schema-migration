package migration

import (
	"fmt"
	"path/filepath"
)

func GenerateGuardianHarnessCases(ir IR, outputDir string) (GuardianHarnessCasesArtifact, ArtifactRef, error) {
	if (ir.Version != "v2" && ir.Version != "v3") || len(ir.HarnessCases) == 0 {
		return GuardianHarnessCasesArtifact{}, ArtifactRef{}, fmt.Errorf("Guardian harness cases require migration v2 or v3")
	}
	schema := GuardianHarnessSchema
	if ir.Version == "v3" {
		schema = GuardianHarnessSchemaV2
	}
	artifact := GuardianHarnessCasesArtifact{
		Schema: schema, MigrationVersion: ir.Version, IRDigest: ir.IRDigest,
		Fixture: ir.GuardianFixture, FixtureV3: cloneGuardianV3Fixture(ir.GuardianFixtureV3), Cases: append([]HarnessCaseDecl(nil), ir.HarnessCases...),
	}
	var err error
	artifact.ArtifactDigest, err = unsignedGuardianHarnessCasesDigest(artifact)
	if err != nil {
		return GuardianHarnessCasesArtifact{}, ArtifactRef{}, err
	}
	path := filepath.Join(outputDir, "generated", "guardian-harness-cases.json")
	if err := WriteJSON(path, artifact); err != nil {
		return GuardianHarnessCasesArtifact{}, ArtifactRef{}, err
	}
	ref, err := artifactRef(path, "generated/guardian-harness-cases.json")
	if err != nil {
		return GuardianHarnessCasesArtifact{}, ArtifactRef{}, err
	}
	return artifact, ref, nil
}

func ValidateGuardianHarnessCases(artifact GuardianHarnessCasesArtifact, ir IR) error {
	expectedSchema := GuardianHarnessSchema
	if ir.Version == "v3" {
		expectedSchema = GuardianHarnessSchemaV2
	}
	if artifact.Schema != expectedSchema || artifact.MigrationVersion != ir.Version || artifact.IRDigest != ir.IRDigest || !sameGuardianFixture(artifact.Fixture, ir.GuardianFixture) || !sameGuardianV3Fixture(artifact.FixtureV3, ir.GuardianFixtureV3) || !sameHarnessCases(artifact.Cases, ir.HarnessCases) {
		return fmt.Errorf("generated Guardian harness cases are not bound to %s IR", ir.Version)
	}
	digest, err := unsignedGuardianHarnessCasesDigest(artifact)
	if err != nil || digest != artifact.ArtifactDigest {
		return fmt.Errorf("generated Guardian harness cases digest mismatch")
	}
	return nil
}

func ValidateGuardianHarnessReport(report GuardianHarnessReport, cases GuardianHarnessCasesArtifact, ir IR) error {
	expectedSchema := GuardianHarnessSchema
	if ir.Version == "v3" {
		expectedSchema = GuardianHarnessSchemaV2
	}
	if report.Schema != expectedSchema || report.MigrationVersion != ir.Version || report.IRDigest != ir.IRDigest || !sameGuardianFixture(report.Fixture, cases.Fixture) || !sameGuardianV3Fixture(report.FixtureV3, cases.FixtureV3) || report.FixtureFileCount < 1 || len(report.Results) != len(cases.Cases) {
		return fmt.Errorf("Guardian harness report is not bound to generated cases")
	}
	caseByID := make(map[string]HarnessCaseDecl, len(cases.Cases))
	for _, item := range cases.Cases {
		caseByID[item.ID] = item
	}
	seen := make(map[string]bool, len(report.Results))
	var summary GuardianHarnessSummary
	for _, result := range report.Results {
		item, ok := caseByID[result.ID]
		if !ok || seen[result.ID] || result.ScenarioID != item.ScenarioID || result.CellID != item.Cell || result.ExpectedState != item.Expected || result.State != item.Expected || result.Claim.State != result.State || result.Stage == "" || result.Step == "" || result.Reason == "" || result.NextOperation == "" || result.BlockedBy == nil {
			return fmt.Errorf("Guardian harness result %q is missing exact case binding", result.ID)
		}
		seen[result.ID] = true
		switch result.State {
		case "CLOSED":
			if result.GuardianDecision != "PASS" || result.Claim.State != "CLOSED" {
				return fmt.Errorf("Guardian case %q did not close from a PASS artifact", result.ID)
			}
			summary.ClosedCount++
		case "UNKNOWN":
			if !ValidateUnknownClaim(result.Claim) || result.Claim.Stage != result.Stage || result.Claim.Step != result.Step || result.Claim.Reason != result.Reason || result.Claim.UnknownClass != result.UnknownClass || result.Claim.NextOperation != result.NextOperation {
				return fmt.Errorf("Guardian case %q does not carry the exact six-field UNKNOWN claim", result.ID)
			}
			summary.UnknownCount++
		case "REFUTED":
			if (result.GuardianDecision != "REFUTED" && result.GuardianDecision != "REFERENCE_ERROR") || result.Claim.State != "REFUTED" {
				return fmt.Errorf("Guardian case %q did not fail closed as REFUTED", result.ID)
			}
			summary.RefutedCount++
		default:
			return fmt.Errorf("Guardian case %q has unsupported state %q", result.ID, result.State)
		}
	}
	if ir.Version == "v3" {
		summary.FoundationAuthorizationCount = report.Summary.FoundationAuthorizationCount
		summary.FoundationReceiptCount = report.Summary.FoundationReceiptCount
	}
	if len(seen) != len(cases.Cases) || report.Summary != summary {
		return fmt.Errorf("Guardian harness summary is not exact")
	}
	if ir.Version == "v3" && (report.Summary.FoundationAuthorizationCount < 1 || report.Summary.FoundationReceiptCount < 1) {
		return fmt.Errorf("v3 Guardian harness must count Foundation authorization and receipt evaluations")
	}
	digest, err := unsignedGuardianHarnessReportDigest(report)
	if err != nil || digest != report.ArtifactDigest {
		return fmt.Errorf("Guardian harness report digest mismatch")
	}
	return nil
}

func AttachGuardianHarness(reportPath, harnessPath, proposalPath, manifestPath, humanPath string) (Report, error) {
	var report Report
	if err := ReadJSON(reportPath, &report); err != nil {
		return Report{}, err
	}
	var harness GuardianHarnessReport
	if err := ReadJSON(harnessPath, &harness); err != nil {
		return Report{}, err
	}
	var cases GuardianHarnessCasesArtifact
	if err := ReadJSON(filepath.Join(filepath.Dir(harnessPath), "generated", "guardian-harness-cases.json"), &cases); err != nil {
		return Report{}, err
	}
	var ir IR
	if err := ReadJSON(filepath.Join(filepath.Dir(harnessPath), "semantic-ir.json"), &ir); err != nil {
		return Report{}, err
	}
	if err := ValidateIR(ir); err != nil {
		return Report{}, err
	}
	if err := ValidateGuardianHarnessCases(cases, ir); err != nil {
		return Report{}, err
	}
	if err := ValidateGuardianHarnessReport(harness, cases, ir); err != nil {
		return Report{}, err
	}
	var proposal AdoptionProposal
	if err := ReadJSON(proposalPath, &proposal); err != nil {
		return Report{}, err
	}
	harnessRef, err := artifactRef(harnessPath, "guardian-harness-report.json")
	if err != nil {
		return Report{}, err
	}
	proposal.GuardianHarnessArtifact = &harnessRef
	proposal.ProposalDigest, err = unsignedProposalDigest(proposal)
	if err != nil {
		return Report{}, err
	}
	if err := WriteJSON(proposalPath, proposal); err != nil {
		return Report{}, err
	}
	proposalRef, err := artifactRef(proposalPath, "adoption-proposal.json")
	if err != nil {
		return Report{}, err
	}
	var manifest ArtifactManifest
	if err := ReadJSON(manifestPath, &manifest); err != nil {
		return Report{}, err
	}
	manifest.Artifacts = replaceArtifactRef(manifest.Artifacts, proposalRef)
	manifest.Artifacts = replaceArtifactRef(manifest.Artifacts, harnessRef)
	manifest.ArtifactDigest, err = unsignedManifestDigest(manifest)
	if err != nil {
		return Report{}, err
	}
	if err := WriteJSON(manifestPath, manifest); err != nil {
		return Report{}, err
	}
	manifestRef, err := artifactRef(manifestPath, "artifact-manifest.json")
	if err != nil {
		return Report{}, err
	}
	report.GuardianHarness = &harness
	report.ArtifactDigests = append(append([]ArtifactRef(nil), manifest.Artifacts...), manifestRef)
	report.ReportDigest, err = DigestValue(report)
	if err != nil {
		return Report{}, err
	}
	if err := WriteJSON(reportPath, report); err != nil {
		return Report{}, err
	}
	if humanPath != "" {
		if err := WriteText(humanPath, RenderReport(report)); err != nil {
			return Report{}, err
		}
	}
	return report, nil
}

func replaceArtifactRef(refs []ArtifactRef, replacement ArtifactRef) []ArtifactRef {
	for i, ref := range refs {
		if ref.Path == replacement.Path {
			refs[i] = replacement
			return refs
		}
	}
	return append(refs, replacement)
}

func sameGuardianFixture(a, b GuardianFixture) bool {
	return a.Repository == b.Repository && a.Ref == b.Ref && a.Commit == b.Commit && a.ManifestPath == b.ManifestPath
}

func sameGuardianV3Fixture(a, b *GuardianV3Fixture) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func sameHarnessCases(a, b []HarnessCaseDecl) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !sameHarnessCase(a[i], b[i]) {
			return false
		}
	}
	return true
}
