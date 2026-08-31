package migration

import (
	"encoding/json"
	"fmt"
	"path/filepath"
)

func GenerateScenarios(ir IR, outputDir string) (ScenarioBundle, []ArtifactRef, error) {
	if err := ValidateIR(ir); err != nil {
		return ScenarioBundle{}, nil, err
	}
	scenarios := make([]ScenarioArtifact, 0, FixedCells)
	refs := make([]ArtifactRef, 0, FixedCells+1)
	for _, declaration := range ir.Scenarios {
		if normalizeScenarioKind(declaration.Kind) != "RECEIPT" {
			continue
		}
		cell, err := cellByID(ir, declaration.Cell)
		if err != nil {
			return ScenarioBundle{}, nil, err
		}
		parentRaw, parentDigest, err := makeParent(declaration, ir)
		if err != nil {
			return ScenarioBundle{}, nil, err
		}
		children, childDigests, err := makeChildren(declaration, ir, parentDigest)
		if err != nil {
			return ScenarioBundle{}, nil, err
		}
		artifact := ScenarioArtifact{
			Schema: ScenarioSchema, ScenarioID: declaration.ID, CellID: cell.ID, Class: declaration.Class, ExpectedState: declaration.Expected,
			Parent: parentRaw, Children: children, ParentWriteAttempts: 0, ParentWrites: 0,
			ParentDigestBefore: parentDigest, ParentDigestAfter: parentDigest, ReplayDeterministic: true,
		}
		if declaration.ChildMode == "V3_REWRITE_PARENT" {
			artifact.ParentWriteAttempts = 1
			artifact.ParentDigestAfter = DigestBytes([]byte("attempted-parent-rewrite:" + parentDigest))
		}
		if declaration.ID == "NORMAL_REPLAY_DETERMINISTIC" {
			artifact.ReplayDigestBefore, artifact.ReplayDigestAfter = childDigests[0], childDigests[0]
		}
		path := filepath.Join(outputDir, "scenarios", declaration.ID+".json")
		if err := WriteJSON(path, artifact); err != nil {
			return ScenarioBundle{}, nil, err
		}
		ref, err := artifactRef(path, filepath.Join("scenarios", declaration.ID+".json"))
		if err != nil {
			return ScenarioBundle{}, nil, err
		}
		refs = append(refs, ref)
		scenarios = append(scenarios, artifact)
	}
	bundle := ScenarioBundle{Schema: ScenarioSchema, IRDigest: ir.IRDigest, Scenarios: scenarios}
	if len(bundle.Scenarios) != FixedCells {
		return ScenarioBundle{}, nil, fmt.Errorf("receipt scenario bundle must preserve exactly twelve v1 scenarios")
	}
	bundle.ArtifactDigest, _ = unsignedScenarioBundleDigest(bundle)
	bundlePath := filepath.Join(outputDir, "scenario-receipts.json")
	if err := WriteJSON(bundlePath, bundle); err != nil {
		return ScenarioBundle{}, nil, err
	}
	ref, err := artifactRef(bundlePath, "scenario-receipts.json")
	if err != nil {
		return ScenarioBundle{}, nil, err
	}
	refs = append(refs, ref)
	return bundle, refs, nil
}

func makeParent(declaration ScenarioDecl, ir IR) (json.RawMessage, string, error) {
	parent := ParentReceipt{
		Schema: ParentV2Schema, SchemaVersion: "v2", ReceiptID: "parent:" + declaration.ID, Kind: "REGRESSION_REPAIR",
		DenominatorID: ir.DenominatorID, DenominatorCellCount: ir.CellCount, DenominatorStageCounts: cloneCounts(ir.StageCounts), DenominatorRoleCounts: cloneCounts(ir.RoleCounts),
		Immutable: true, CreatedBy: "gooo-receipt-schema-migration/" + ir.Version,
	}
	var extras map[string]json.RawMessage
	if declaration.ParentMode == "V2_FUTURE_FIELD" {
		extras = map[string]json.RawMessage{"outcome": json.RawMessage(`"REFUTED_INCOMPLETE_PROPAGATION"`)}
	}
	return rawWithDigest(parent, extras, "")
}

func makeChildren(declaration ScenarioDecl, ir IR, parentDigest string) ([]json.RawMessage, []string, error) {
	count := 1
	if declaration.ChildMode == "V3_SECOND_CHILD" {
		count = 2
	}
	children := make([]json.RawMessage, 0, count)
	digests := make([]string, 0, count)
	for ordinal := 1; ordinal <= count; ordinal++ {
		child := ChildReceipt{
			Schema: ChildV3Schema, SchemaVersion: "v3", ReceiptID: fmt.Sprintf("child:%s:%d", declaration.ID, ordinal),
			ParentReceiptID: "parent:" + declaration.ID, ParentDigest: parentDigest, ParentOutcome: "REFUTED_INCOMPLETE_PROPAGATION",
			Outcome: "CORRECTION_APPLIED", CauseCode: "OUTCOME_NOT_PROPAGATED", CausalChain: []string{"REGRESSION_REPAIR", "CORRECTION_CHILD"},
			NextOperation: "CONSUME_CHILD_OUTCOME", DenominatorID: ir.DenominatorID, DenominatorCellCount: ir.CellCount,
			DenominatorStageCounts: cloneCounts(ir.StageCounts), DenominatorRoleCounts: cloneCounts(ir.RoleCounts), Attestation: "INDEPENDENT_EVALUATOR", ChildOrdinal: ordinal,
		}
		extras := map[string]json.RawMessage{}
		forcedDigest := ""
		switch declaration.ChildMode {
		case "V3_PARENT_DIGEST_MISSING":
			child.ParentDigest = ""
		case "V3_CHILD_DIGEST_STALE":
			forcedDigest = "sha256:stale-child-digest"
		case "V4_FUTURE":
			child.Schema, child.SchemaVersion = "gooo/receipt/child/v4", "v4"
		case "V3_PARENT_DIGEST_MISMATCH":
			child.ParentDigest = "sha256:" + repeated("b", 64)
		case "V3_SELF_ATTESTED":
			child.Attestation = "SELF"
		case "V3_DENOMINATOR_DOWNGRADE":
			child.DenominatorCellCount = 11
			child.DenominatorStageCounts = map[string]int{"FOUNDATION": 4, "COHERENCE": 4, "REGRESSION": 3}
		case "V3_UNKNOWN_FIXED_POINT":
			child.ParentOutcome = "UNKNOWN"
			child.Outcome = "FIXED_POINT"
			child.CauseCode = "UNKNOWN_PROMOTED_WITHOUT_EVIDENCE"
		case "V3_REPLAY", "V3_REWRITE_PARENT", "V3", "V3_SECOND_CHILD":
		default:
			return nil, nil, fmt.Errorf("unsupported child mode %q", declaration.ChildMode)
		}
		if declaration.ChildMode == "V3_REWRITE_PARENT" {
			extras["rewrite_parent"] = json.RawMessage(`true`)
		}
		raw, digest, err := rawWithDigest(child, extras, forcedDigest)
		if err != nil {
			return nil, nil, err
		}
		children = append(children, raw)
		digests = append(digests, digest)
	}
	return children, digests, nil
}

func rawWithDigest(value any, extras map[string]json.RawMessage, forcedDigest string) (json.RawMessage, string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, "", err
	}
	object, err := validateJSONShape(data)
	if err != nil {
		return nil, "", err
	}
	for key, raw := range extras {
		object[key] = raw
	}
	delete(object, "digest")
	digest, err := DigestValue(object)
	if err != nil {
		return nil, "", err
	}
	if forcedDigest != "" {
		digest = forcedDigest
	}
	encodedDigest, _ := json.Marshal(digest)
	object["digest"] = encodedDigest
	final, err := json.Marshal(object)
	if err != nil {
		return nil, "", err
	}
	return json.RawMessage(final), digest, nil
}

func repeated(value string, count int) string {
	output := ""
	for i := 0; i < count; i++ {
		output += value
	}
	return output
}

func EvaluateScenarioBundle(ir IR, bundle ScenarioBundle, outputDir string) (Report, error) {
	if err := ValidateIR(ir); err != nil {
		return Report{}, err
	}
	if bundle.Schema != ScenarioSchema || bundle.IRDigest != ir.IRDigest || len(bundle.Scenarios) != FixedCells {
		return Report{}, fmt.Errorf("scenario bundle is not bound to fixed semantic IR")
	}
	if digest, err := unsignedScenarioBundleDigest(bundle); err != nil || digest != bundle.ArtifactDigest {
		return Report{}, fmt.Errorf("scenario bundle digest mismatch")
	}
	cellIndex := make(map[string]Cell, len(ir.Cells))
	for _, cell := range ir.Cells {
		cellIndex[cell.ID] = cell
	}
	reportSchema := ReportSchema
	if ir.Version == "v2" {
		reportSchema = ReportSchemaV2
	} else if ir.Version == "v3" {
		reportSchema = ReportSchemaV3
	}
	pipelineSource := "examples/receipt-schema-migration-v1/migration.gooo"
	if ir.Version == "v2" {
		pipelineSource = "examples/receipt-schema-migration-v2/migration.gooo"
	}
	report := Report{
		Schema: reportSchema, Decision: "RECEIPT_SCHEMA_MIGRATION_CONFORMANCE_REPORTED",
		Pipeline:       map[string]string{"source": pipelineSource, "semantic_ir": "semantic-ir.json", "generated_adapters": "generated/adapters/v2.json,generated/adapters/v3.json", "generated_validator": "generated/validator.json", "scenario_receipts": "scenario-receipts.json", "guardian_harness_cases": "generated/guardian-harness-cases.json", "guardian_harness_report": "guardian-harness-report.json", "development_provenance": "development-provenance.json", "human_report": "human-report.md"},
		SchemaVersions: []string{"v2", "v3"}, FixedDenominator: ir.CellCount, StageCounts: cloneCounts(ir.StageCounts), RoleCounts: cloneCounts(ir.RoleCounts),
		Precedence: append([]string(nil), ir.Precedence...), UnknownFields: append([]string(nil), ir.UnknownFields...), Authority: ir.Authority,
		AdapterOperations: adapterOperations(ir), MetricBindings: append([]MetricBinding(nil), ir.Metrics...), Scenarios: make([]ScenarioResult, 0, FixedCells),
		Improvement: unknownClaim("IMPROVEMENT", "compare_before_after", "EXACT_COMPARABLE_PAIR_NOT_PROVIDED", "MISSING_EXACT_PAIR", "PROVIDE_EXACT_COMPARABLE_PAIR", []string{"before-after-evidence"}),
	}
	if ir.Version == "v3" {
		provenance := developmentProvenance()
		provenance.ProvenanceDigest, _ = unsignedDevelopmentProvenanceDigest(provenance)
		report.DevelopmentProvenance = &provenance
	}
	if ir.Version == "v2" || ir.Version == "v3" {
		record := ir.Migration
		report.MigrationVersion = ir.Version
		report.Migration = &record
		report.GuardianFixtureV3 = cloneGuardianV3Fixture(ir.GuardianFixtureV3)
	}
	for _, scenario := range bundle.Scenarios {
		cell, ok := cellIndex[scenario.CellID]
		if !ok {
			return Report{}, fmt.Errorf("scenario %q is bound to unknown cell", scenario.ScenarioID)
		}
		result := evaluateScenario(ir, scenario, cell)
		report.Scenarios = append(report.Scenarios, result)
		report.Summary.ParentReceiptCount++
		report.Summary.ChildReceiptCount += len(scenario.Children)
		switch result.State {
		case "ACCEPTED":
			report.Summary.AcceptedCount++
		case "UNKNOWN":
			report.Summary.UnknownCount++
		case "REFUTED":
			report.Summary.RefutedCount++
		}
		report.Summary.ImmutableParentWrites += scenario.ParentWrites
		if result.State != scenario.ExpectedState {
			return Report{}, fmt.Errorf("scenario %s expected %s, observed %s", scenario.ScenarioID, scenario.ExpectedState, result.State)
		}
	}
	if report.Summary.ImmutableParentWrites != 0 {
		return Report{}, fmt.Errorf("immutable parent writes must be zero")
	}
	refs, err := collectArtifactRefs(outputDir, []string{"semantic-ir.json", "generated/adapters/v2.json", "generated/adapters/v3.json", "generated/validator.json", "scenario-receipts.json", "adoption-proposal.json"})
	if err != nil {
		return Report{}, err
	}
	if ir.Version == "v2" || ir.Version == "v3" {
		ref, err := artifactRef(filepath.Join(outputDir, "generated/guardian-harness-cases.json"), "generated/guardian-harness-cases.json")
		if err != nil {
			return Report{}, err
		}
		refs = append(refs, ref)
	}
	report.ArtifactDigests = refs
	return report, nil
}

func evaluateScenario(ir IR, artifact ScenarioArtifact, cell Cell) ScenarioResult {
	result := ScenarioResult{ScenarioID: artifact.ScenarioID, CellID: artifact.CellID, Stage: cell.Stage, Role: cell.Role, ExpectedState: artifact.ExpectedState, State: "REFUTED", ChildDigests: []string{}}
	parent, parentDigest, err := decodeParent(artifact.Parent)
	if err != nil {
		result.ObservedReason = "PARENT_RECEIPT_MALFORMED"
		result.Claim = refutedClaim("PARENT", "decode_parent", result.ObservedReason, "REGENERATE_PARENT_RECEIPT")
		return result
	}
	result.ParentDigest = parentDigest
	for _, child := range artifact.Children {
		if digest, err := childDigestField(child); err == nil {
			result.ChildDigests = append(result.ChildDigests, digest)
		}
	}
	if artifact.ParentWriteAttempts > 0 || artifact.ParentWrites != 0 {
		result.ObservedReason = "IMMUTABLE_PARENT_REWRITE_ATTEMPT"
		result.Claim = refutedClaim("LINEAGE", "reject_parent_write", result.ObservedReason, "APPEND_NEW_CHILD_WITHOUT_PARENT_WRITE")
		return result
	}
	if reason := validateParentV2(artifact.Parent, parent, ir, parentDigest); reason != "" {
		result.ObservedReason = reason
		result.Claim = refutedClaim("PARENT", "validate_v2_parent", reason, "PRESERVE_IMMUTABLE_V2_PARENT")
		return result
	}
	if len(artifact.Children) != 1 {
		result.ObservedReason = "CHILD_CARDINALITY_EXCEEDS_ONE"
		result.Claim = refutedClaim("LINEAGE", "validate_child_cardinality", result.ObservedReason, "RETAIN_SINGLE_CHILD_CARDINALITY")
		return result
	}
	childRaw := artifact.Children[0]
	child, err := decodeChild(childRaw)
	if err != nil {
		result.ObservedReason = "CHILD_RECEIPT_MALFORMED"
		result.Claim = refutedClaim("CHILD", "decode_child", result.ObservedReason, "REGENERATE_VERSIONED_CHILD")
		return result
	}
	if child.Schema != ChildV3Schema || child.SchemaVersion != "v3" {
		result.State = "UNKNOWN"
		result.ObservedReason = "UNSUPPORTED_FUTURE_SCHEMA"
		result.Claim = unknownClaim("ADAPTER", "dispatch_schema_version", result.ObservedReason, "UNSUPPORTED_SCHEMA", "ADD_VERIFIED_VERSION_ADAPTER", []string{"schema-adapter:v4"})
		return result
	}
	actualChildDigest, err := DigestRawReceipt(childRaw)
	if err != nil || actualChildDigest != child.Digest {
		result.State = "UNKNOWN"
		result.ObservedReason = "STALE_CHILD_DIGEST"
		result.Claim = unknownClaim("EVIDENCE", "verify_child_digest", result.ObservedReason, "STALE_CHILD_EVIDENCE", "OBTAIN_CURRENT_CHILD_BYTES", []string{"child-digest"})
		return result
	}
	if child.ParentDigest == "" {
		result.State = "UNKNOWN"
		result.ObservedReason = "MISSING_PARENT_DIGEST"
		result.Claim = unknownClaim("LINEAGE", "resolve_parent_digest", result.ObservedReason, "MISSING_PARENT_EVIDENCE", "SUPPLY_IMMUTABLE_PARENT_DIGEST", []string{"parent-digest"})
		return result
	}
	if child.ParentReceiptID != parent.ReceiptID || child.ParentDigest != parentDigest {
		result.ObservedReason = "PARENT_DIGEST_MISMATCH"
		result.Claim = refutedClaim("LINEAGE", "compare_parent_digest", result.ObservedReason, "REISSUE_CHILD_WITH_EXACT_PARENT_DIGEST")
		return result
	}
	if child.Attestation != "INDEPENDENT_EVALUATOR" {
		result.ObservedReason = "SELF_ATTESTATION_NOT_AUTHORITY"
		result.Claim = refutedClaim("AUTHORITY", "validate_attestation", result.ObservedReason, "OBTAIN_INDEPENDENT_EVALUATOR_ATTESTATION")
		return result
	}
	if child.DenominatorCellCount != ir.CellCount || !sameCounts(child.DenominatorStageCounts, ir.StageCounts) || !sameCounts(child.DenominatorRoleCounts, ir.RoleCounts) {
		result.ObservedReason = "DENOMINATOR_DOWNGRADE"
		result.Claim = refutedClaim("DENOMINATOR", "validate_child_denominator", result.ObservedReason, "RESTORE_FIXED_TWELVE_CELL_DENOMINATOR")
		return result
	}
	if child.ParentOutcome == "UNKNOWN" && child.Outcome == "FIXED_POINT" {
		result.ObservedReason = "UNKNOWN_PROMOTED_TO_FIXED_POINT"
		result.Claim = refutedClaim("RESOLUTION", "reject_unknown_promotion", result.ObservedReason, "PRESERVE_UNKNOWN_FRONTIER")
		return result
	}
	if child.ParentOutcome != "REFUTED_INCOMPLETE_PROPAGATION" || child.Outcome == "" || child.CauseCode == "" || len(child.CausalChain) < 2 || child.NextOperation == "" {
		result.ObservedReason = "CHILD_OUTCOME_CAUSAL_FIELDS_INCOMPLETE"
		result.Claim = refutedClaim("CHILD", "validate_outcome_causality", result.ObservedReason, "REGENERATE_COMPLETE_V3_CHILD")
		return result
	}
	if artifact.ScenarioID == "NORMAL_REPLAY_DETERMINISTIC" && (!artifact.ReplayDeterministic || artifact.ReplayDigestBefore == "" || artifact.ReplayDigestBefore != artifact.ReplayDigestAfter) {
		result.State = "UNKNOWN"
		result.ObservedReason = "REPLAY_BYTES_NOT_DETERMINISTIC"
		result.Claim = unknownClaim("REPLAY", "compare_replay_bytes", result.ObservedReason, "NONDETERMINISTIC_REPLAY", "REPEAT_REPLAY_WITH_IDENTICAL_INPUT", []string{"replay-digest"})
		return result
	}
	result.State = "ACCEPTED"
	result.ObservedReason = "APPEND_ONLY_V3_CHILD_VALIDATED"
	result.Claim = acceptedClaim("CHILD", "validate_append_only_migration", result.ObservedReason)
	return result
}

func decodeParent(raw json.RawMessage) (ParentReceipt, string, error) {
	var parent ParentReceipt
	if err := json.Unmarshal(raw, &parent); err != nil {
		return ParentReceipt{}, "", err
	}
	digest, err := DigestRawReceipt(raw)
	return parent, digest, err
}

func decodeChild(raw json.RawMessage) (ChildReceipt, error) {
	var child ChildReceipt
	if err := json.Unmarshal(raw, &child); err != nil {
		return ChildReceipt{}, err
	}
	return child, nil
}

func childDigestField(raw json.RawMessage) (string, error) {
	value, err := validateJSONShape(raw)
	if err != nil {
		return "", err
	}
	var digest string
	if err := json.Unmarshal(value["digest"], &digest); err != nil {
		return "", err
	}
	return digest, nil
}

func validateParentV2(raw json.RawMessage, parent ParentReceipt, ir IR, digest string) string {
	value, err := validateJSONShape(raw)
	if err != nil {
		return "PARENT_RECEIPT_MALFORMED"
	}
	for _, forbidden := range []string{"outcome", "parent_outcome", "cause_code", "causal_chain", "next_operation", "attestation"} {
		if _, ok := value[forbidden]; ok {
			return "FUTURE_FIELD_IN_OLD_V2_PARENT"
		}
	}
	if parent.Schema != ParentV2Schema || parent.SchemaVersion != "v2" || parent.Kind != "REGRESSION_REPAIR" || parent.ReceiptID == "" || !parent.Immutable {
		return "V2_PARENT_IDENTITY_INVALID"
	}
	if parent.Digest == "" || parent.Digest != digest {
		return "V2_PARENT_DIGEST_INVALID"
	}
	if parent.DenominatorID != ir.DenominatorID || parent.DenominatorCellCount != ir.CellCount || !sameCounts(parent.DenominatorStageCounts, ir.StageCounts) || !sameCounts(parent.DenominatorRoleCounts, ir.RoleCounts) {
		return "V2_PARENT_DENOMINATOR_INVALID"
	}
	return ""
}

func collectArtifactRefs(outputDir string, paths []string) ([]ArtifactRef, error) {
	refs := make([]ArtifactRef, 0, len(paths))
	for _, path := range paths {
		ref, err := artifactRef(filepath.Join(outputDir, path), path)
		if err != nil {
			return nil, err
		}
		refs = append(refs, ref)
	}
	return refs, nil
}
