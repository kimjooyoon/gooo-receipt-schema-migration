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
	if ir.Version == "v2" {
		proposal.Schema = ProposalSchemaV2
		proposal.ProposalID = "receipt-schema-migration-v2-adoption"
		proposal.MigrationVersion = "v2"
		migration := ir.Migration
		proposal.Migration = &migration
		fixture := ir.GuardianFixture
		proposal.GuardianFixture = &fixture
		proposal.VariableLifetimeOwnership = []string{
			"Declare beforeDigest and afterDigest in the workflow-script scope that consumes them, before the policy branch.",
			"The base-controlled workflow is read from github.workflow_sha; candidate-controlled changed paths come only from the live pull request API.",
			"A PASS artifact with protected kernel paths must carry non-null before and after digests computed from the exact base and head.",
			"A null, stale, mismatched, or unavailable digest is REFUTED and never promoted to CLOSED.",
		}
		proposal.ExactRequiredSemanticChanges = append(proposal.ExactRequiredSemanticChanges,
			"Preserve the v1 twelve-cell denominator and publish v2 as exactly sixteen cells.",
			"Record ADD=4, RETIRE=0, SPLIT=0 with the four exact Guardian cells and exact proof/indicator deltas.",
			"Pin the actual dev@7f45792 ci-guardian workflow, Guardian validator, dependencies, and related validator scope by commit, path, and blob SHA.",
			"Execute the real pull_request_target feature-PR script in caller-owned temporary space; do not close the audit with static string search.",
			"Resolve REFUTED before UNKNOWN before CLOSED for Guardian evidence and preserve UNKNOWN's six required fields.",
		)
		proposal.ExpectedProtectedPaths = []string{
			".github/ci-governance.json",
			".github/agent-scope-table.md",
			".github/branch-policy.md",
			".github/conformance-plan.md",
			".github/foundation-authorization.json",
			"go.mod",
			"go.sum",
		}
		proposal.RollbackConditions = append(proposal.RollbackConditions,
			"Rollback if the upstream commit, protected path, or verified blob SHA differs from the pinned fixture.",
			"Rollback if the current workflow still raises an uncaught beforeDigest/afterDigest ReferenceError before policy classification.",
			"Rollback if a PASS artifact has null or mismatched kernel digests, or if protected migration paths do not fail closed.",
			"Rollback if the v1 twelve-cell denominator changes or v2 does not report exact ADD=4/RETIRE=0/SPLIT=0.",
		)
		proposal.RefutationConditions = append(proposal.RefutationConditions,
			"REFUTED on a workflow-scope ReferenceError, null digest, digest mismatch, or protected migration path bypass.",
			"UNKNOWN, never CLOSED, for an unsupported future schema with exactly stage, step, reason, unknown_class, next_operation, and blocked_by.",
		)
		proposal.HarnessAcceptanceCases = []AcceptanceCase{
			{ID: "GUARDIAN_CURRENT_SNAPSHOT_REFERENCE_ERROR", ExpectedState: "REFUTED", Invariant: "the actual current workflow scope raises ReferenceError before policy classification and is converted to REFUTED"},
			{ID: "GUARDIAN_CORRECTED_SCOPE_CLOSED", ExpectedState: "CLOSED", Invariant: "beforeDigest and afterDigest are initialized before the branch and a non-null PASS artifact closes"},
			{ID: "GUARDIAN_NULL_DIGEST_REFUTED", ExpectedState: "REFUTED", Invariant: "null digest evidence is REFUTED"},
			{ID: "GUARDIAN_DIGEST_MISMATCH_REFUTED", ExpectedState: "REFUTED", Invariant: "different computed and artifact digests are REFUTED"},
			{ID: "GUARDIAN_BASE_WORKFLOW_CANDIDATE_FILES_SEPARATE", ExpectedState: "CLOSED", Invariant: "base-controlled workflow authority and candidate-controlled changed paths remain separate"},
			{ID: "GUARDIAN_PROTECTED_MIGRATION_PATH_FAIL_CLOSED", ExpectedState: "REFUTED", Invariant: "a protected migration path is rejected by the actual Guardian changed-file validator"},
			{ID: "GUARDIAN_UNSUPPORTED_FUTURE_SCHEMA_UNKNOWN", ExpectedState: "UNKNOWN", Invariant: "unsupported future schema remains UNKNOWN with the exact six-field claim"},
			{ID: "GUARDIAN_DIGEST_MATCH_CLOSED", ExpectedState: "CLOSED", Invariant: "matching non-null computed and PASS artifact digests close through the actual attestation validator"},
		}
	}
	if external != nil {
		if err := validateExternalRelease(*external); err != nil {
			return AdoptionProposal{}, ArtifactRef{}, err
		}
		proposal.OptionalExternalRelease = external
	}
	for _, scenario := range ir.Scenarios {
		if normalizeScenarioKind(scenario.Kind) != "RECEIPT" {
			continue
		}
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
