package migration

import (
	"fmt"
	"path/filepath"
	"strings"
)

func BuildAdoptionProposal(ir IR, outputDir string, external *ExternalRelease) (AdoptionProposal, ArtifactRef, error) {
	if err := ValidateIR(ir); err != nil {
		return AdoptionProposal{}, ArtifactRef{}, err
	}
	proposal := AdoptionProposal{
		Schema: ProposalSchema, ProposalID: "receipt-schema-migration-v1-adoption",
		TargetRepository: "github.com/kimjooyoon/meta-ontology-go", RepositoryWrites: 0, LocalTestExecutions: 0, CrossProjectRequiredGates: 0,
		OldSchemaOwnership: []SchemaOwnership{{SchemaVersion: "v2", Owner: "immutable parent", Owns: append([]string(nil), ir.Adapters[0].OwnedFields...), MustNotOwn: append([]string(nil), ir.Adapters[0].ForbiddenFields...)}},
		NewSchemaOwnership: []SchemaOwnership{{SchemaVersion: "v3", Owner: "append-only correction child", Owns: append([]string(nil), ir.Adapters[1].OwnedFields...), MustNotOwn: append([]string(nil), ir.Adapters[1].ForbiddenFields...)}},
		ExactRequiredSemanticChanges: []string{
			"Keep every consumed gooo/receipt/parent/v2 REGRESSION_REPAIR byte sequence and digest unchanged.",
			"Do not add outcome or causal fields to the v2 parent adapter or v2 parent validator requirements.",
			"Add gooo/receipt/child/v3 as an append-only child edge keyed by parent_receipt_id and exact parent_digest.",
			"Make the v3 child own parent_outcome=REFUTED_INCOMPLETE_PROPAGATION, outcome, cause_code, causal_chain, and next_operation.",
			"Reject a child write to an existing parent and enforce parent child cardinality one.",
			"Resolve REFUTED before UNKNOWN before CLOSED and preserve the six-field UNKNOWN frontier.",
			"Require the unchanged twelve-cell denominator with Foundation, Coherence, Regression and Driver, Outcome, Guardrail counts of 4/4/4.",
		},
		ExpectedProtectedPaths: []string{
			".github/ci-governance.json",
			".github/governance-denominator-v2-migration.json",
			".github/governance-denominator-v3-correction.json",
			".github/workflows/ci-guardian.yml",
			"internal/verify/scope_foundation_correction_child_20260831.go",
			"scripts/ci-proof/guardian.js",
			"scripts/meta-receipts/run.go",
		},
		RollbackConditions: []string{
			"Rollback the adoption proposal if any v2 parent digest or byte sequence changes.",
			"Rollback if v3 child validation makes an old v2 parent outcome field mandatory.",
			"Rollback if the fixed twelve-cell denominator or any 4/4/4 axis count changes.",
			"Rollback if required CI checks are absent or if the external release digest cannot be verified.",
		},
		RefutationConditions: []string{
			"REFUTED on any immutable parent write or child rewrite attempt.",
			"REFUTED on a future field in the old v2 schema, parent digest mismatch, or second child under cardinality one.",
			"REFUTED on self-attestation, denominator downgrade, or UNKNOWN promoted to FIXED_POINT.",
			"UNKNOWN, never CLOSED, for missing parent digest, stale child digest, or unsupported future schema.",
		},
		AcceptanceCases: make([]AcceptanceCase, 0, len(ir.Scenarios)),
	}
	if external != nil {
		if err := validateExternalRelease(*external); err != nil {
			return AdoptionProposal{}, ArtifactRef{}, err
		}
		proposal.OptionalExternalRelease = external
	}
	for _, scenario := range ir.Scenarios {
		invariant := ""
		switch scenario.ID {
		case "NORMAL_V2_PARENT_V3_CHILD":
			invariant = "v2 parent without outcome plus v3 child with parent_outcome REFUTED_INCOMPLETE_PROPAGATION validates"
		case "NORMAL_REPLAY_DETERMINISTIC":
			invariant = "replaying identical source and contract emits identical receipt bytes and digests"
		case "UNKNOWN_PARENT_DIGEST_MISSING":
			invariant = "missing parent digest remains UNKNOWN with the six required fields"
		case "UNKNOWN_CHILD_DIGEST_STALE":
			invariant = "stale child digest remains UNKNOWN with the six required fields"
		case "UNKNOWN_UNSUPPORTED_FUTURE_SCHEMA":
			invariant = "unsupported future schema remains UNKNOWN with the six required fields"
		default:
			invariant = "contradiction resolves REFUTED before any UNKNOWN evidence gap"
		}
		proposal.AcceptanceCases = append(proposal.AcceptanceCases, AcceptanceCase{ID: scenario.ID, ExpectedState: scenario.Expected, Invariant: invariant})
	}
	proposal.ProposalDigest, _ = unsignedProposalDigest(proposal)
	path := filepath.Join(outputDir, "adoption-proposal.json")
	if err := WriteJSON(path, proposal); err != nil {
		return AdoptionProposal{}, ArtifactRef{}, err
	}
	ref, err := artifactRef(path, "adoption-proposal.json")
	if err != nil {
		return AdoptionProposal{}, ArtifactRef{}, err
	}
	return proposal, ref, nil
}

func validateExternalRelease(release ExternalRelease) error {
	if release.URI == "" || !strings.Contains(release.URI, "://") || len(release.Digest) != len("sha256:")+64 || !strings.HasPrefix(release.Digest, "sha256:") {
		return fmt.Errorf("optional external release must include a URI and immutable sha256 digest")
	}
	return nil
}
