package migration

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

func ValidateDeclarations(source SourceDecl, contract Contract) error {
	if source.Schema != SourceSchema || source.Version != "v1" {
		return fmt.Errorf("invalid .gooo source declaration")
	}
	if contract.Schema != ContractSchema || contract.Version != "v1" || contract.ID != source.DenominatorID || !contract.Fixed {
		return fmt.Errorf("fixed denominator declaration mismatch")
	}
	if source.CellCount != FixedCells || contract.CellCount != FixedCells || len(source.Cells) != FixedCells || len(contract.Cells) != FixedCells || len(source.Scenarios) != FixedCells || len(contract.Scenarios) != FixedCells {
		return fmt.Errorf("fixed denominator must contain exactly twelve cells and scenarios")
	}
	if !sameCounts(source.StageCounts, contract.StageCounts) || !sameCounts(source.RoleCounts, contract.RoleCounts) || !sameCounts(source.StageCounts, map[string]int{"FOUNDATION": 4, "COHERENCE": 4, "REGRESSION": 4}) || !sameCounts(source.RoleCounts, map[string]int{"DRIVER": 4, "OUTCOME": 4, "GUARDRAIL": 4}) {
		return fmt.Errorf("stage and role denominators must each be 4/4/4")
	}
	if !sameStrings(source.Precedence, []string{"REFUTED", "UNKNOWN", "CLOSED"}) || !sameStrings(contract.Precedence, source.Precedence) {
		return fmt.Errorf("resolution precedence must be REFUTED > UNKNOWN > CLOSED")
	}
	if !sameStrings(source.UnknownFields, []string{"stage", "step", "reason", "unknown_class", "next_operation", "blocked_by"}) || !sameStrings(contract.UnknownFields, source.UnknownFields) {
		return fmt.Errorf("UNKNOWN must carry the six required fields")
	}
	if source.Authority.RepositoryWrites != 0 || source.Authority.LocalTestExecutions != 0 || source.Authority.CrossProjectRequiredGates != 0 || source.Authority.ProductGenerationAuthorized || source.Authority.RootReadmePolicy != "EXCLUDED_FROM_REPOSITORY_INVENTORY" {
		return fmt.Errorf("authority declaration must be zero and non-escalating")
	}
	if len(source.Schemas) != 2 || len(contract.Schemas) != 2 || !sameAdapters(source.Schemas, contract.Schemas) {
		return fmt.Errorf("exactly the v2 parent and v3 child adapters are required")
	}
	if source.Schemas[0].Version != "v2" || source.Schemas[0].Owner != "parent" || source.Schemas[1].Version != "v3" || source.Schemas[1].Owner != "child" {
		return fmt.Errorf("schema ownership order must be v2 parent then v3 child")
	}
	if contains(source.Schemas[0].OwnedFields, "outcome") || contains(source.Schemas[0].OwnedFields, "parent_outcome") || contains(source.Schemas[0].OwnedFields, "cause_code") || contains(source.Schemas[0].OwnedFields, "causal_chain") || contains(source.Schemas[0].OwnedFields, "next_operation") {
		return fmt.Errorf("v2 parent cannot own v3 outcome or causal fields")
	}
	for _, field := range []string{"parent_outcome", "outcome", "cause_code", "causal_chain", "next_operation"} {
		if !contains(source.Schemas[1].OwnedFields, field) {
			return fmt.Errorf("v3 child must own %s", field)
		}
	}

	seenCells := map[string]bool{}
	stageCounts := map[string]int{}
	roleCounts := map[string]int{}
	for i, cell := range source.Cells {
		if cell.Ordinal != i+1 || cell.ID == "" || seenCells[cell.ID] || cell.Stage == "" || cell.Role == "" || cell.SemanticEdge == "" {
			return fmt.Errorf("cell %d has invalid identity", i+1)
		}
		if !sameCell(cell, contract.Cells[i]) {
			return fmt.Errorf("cell %d differs from fixed contract", i+1)
		}
		seenCells[cell.ID] = true
		stageCounts[cell.Stage]++
		roleCounts[cell.Role]++
	}
	if !sameCounts(stageCounts, source.StageCounts) || !sameCounts(roleCounts, source.RoleCounts) {
		return fmt.Errorf("cell stage or role counts do not match declared 4/4/4 axes")
	}

	seenScenarios := map[string]bool{}
	expectedCounts := map[string]int{"ACCEPTED": 0, "UNKNOWN": 0, "REFUTED": 0}
	for i, scenario := range source.Scenarios {
		if scenario.Ordinal != i+1 || scenario.ID == "" || seenScenarios[scenario.ID] || !seenCells[scenario.Cell] || scenario.Expected == "" || scenario.Operation == "" {
			return fmt.Errorf("scenario %d has invalid identity", i+1)
		}
		if !sameScenario(scenario, contract.Scenarios[i]) {
			return fmt.Errorf("scenario %d differs from fixed contract", i+1)
		}
		if scenario.Class == "NORMAL" && scenario.Expected != "ACCEPTED" || scenario.Class == "UNKNOWN" && scenario.Expected != "UNKNOWN" || scenario.Class == "REFUTED" && scenario.Expected != "REFUTED" {
			return fmt.Errorf("scenario %q has inconsistent class and expected state", scenario.ID)
		}
		seenScenarios[scenario.ID] = true
		expectedCounts[scenario.Expected]++
	}
	if expectedCounts["ACCEPTED"] != 2 || expectedCounts["UNKNOWN"] != 3 || expectedCounts["REFUTED"] != 7 {
		return fmt.Errorf("scenario denominator must contain exact accepted=2, unknown=3, refuted=7 cases")
	}
	if len(source.Metrics) != 15 || !sameMetrics(source.Metrics, contract.Metrics) {
		return fmt.Errorf("all fifteen declared metrics must be source, IR, artifact, and evaluator bound")
	}
	seenMetrics := map[string]bool{}
	for _, metric := range source.Metrics {
		if metric.ID == "" || seenMetrics[metric.ID] || metric.MetaActivity == "" || metric.SourcePath == "" || metric.IRPath == "" || metric.GeneratedArtifact == "" || metric.Evaluator == "" {
			return fmt.Errorf("metric %q is not fully bound to meta code", metric.ID)
		}
		seenMetrics[metric.ID] = true
	}
	return nil
}

func ValidateContract(contract Contract) error {
	if contract.Schema != ContractSchema || contract.CellCount != FixedCells || !contract.Fixed {
		return fmt.Errorf("invalid fixed denominator contract")
	}
	return nil
}

func ValidateIR(ir IR) error {
	if ir.Schema != IRScheme || ir.Version != "v1" || ir.DenominatorID == "" || ir.CellCount != FixedCells || len(ir.Cells) != FixedCells || len(ir.Scenarios) != FixedCells || len(ir.Adapters) != 2 {
		return fmt.Errorf("semantic IR shape is not fixed at twelve cells")
	}
	if ir.SourceDigest == "" || ir.ContractDigest == "" || ir.IRDigest == "" {
		return fmt.Errorf("semantic IR is missing a digest binding")
	}
	expected, err := unsignedIRDigest(ir)
	if err != nil {
		return err
	}
	if expected != ir.IRDigest {
		return fmt.Errorf("semantic IR digest mismatch")
	}
	return nil
}

func ValidateAdapter(artifact AdapterArtifact, ir IR) error {
	if artifact.Schema != AdapterSchema || artifact.IRDigest != ir.IRDigest || artifact.Version == "" || artifact.Owner == "" || len(artifact.Operations) == 0 {
		return fmt.Errorf("generated adapter is not bound to semantic IR")
	}
	digest, err := unsignedAdapterDigest(artifact)
	if err != nil || artifact.ArtifactDigest == "" || digest != artifact.ArtifactDigest {
		return fmt.Errorf("generated adapter digest mismatch")
	}
	return nil
}

func ValidateValidator(artifact ValidatorArtifact, ir IR) error {
	if artifact.Schema != ValidatorSchema || artifact.IRDigest != ir.IRDigest || len(artifact.SupportedSchemas) != 2 || len(artifact.Operations) == 0 || !sameStrings(artifact.Precedence, ir.Precedence) || !sameStrings(artifact.UnknownFields, ir.UnknownFields) {
		return fmt.Errorf("generated validator is not bound to semantic IR")
	}
	digest, err := unsignedValidatorDigest(artifact)
	if err != nil || artifact.ArtifactDigest == "" || digest != artifact.ArtifactDigest {
		return fmt.Errorf("generated validator digest mismatch")
	}
	return nil
}

func ValidateUnknownClaim(claim Claim) bool {
	return claim.HasUnknownTuple() && claim.State == "UNKNOWN"
}

func sameCounts(a, b map[string]int) bool {
	if len(a) != len(b) {
		return false
	}
	for key, value := range a {
		if b[key] != value {
			return false
		}
	}
	return true
}

func sameAdapters(a, b []AdapterDecl) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Version != b[i].Version || a[i].Owner != b[i].Owner || a[i].InputSchema != b[i].InputSchema || a[i].OutputSchema != b[i].OutputSchema || !sameStrings(a[i].Operations, b[i].Operations) || !sameStrings(a[i].OwnedFields, b[i].OwnedFields) || !sameStrings(a[i].ForbiddenFields, b[i].ForbiddenFields) {
			return false
		}
	}
	return true
}

func sameCell(a, b Cell) bool {
	return a.Ordinal == b.Ordinal && a.ID == b.ID && a.Stage == b.Stage && a.Role == b.Role && a.SemanticEdge == b.SemanticEdge && sameStrings(a.DependsOn, b.DependsOn)
}

func sameScenario(a, b ScenarioDecl) bool {
	return a.Ordinal == b.Ordinal && a.ID == b.ID && a.Cell == b.Cell && a.Class == b.Class && a.Expected == b.Expected && a.ParentMode == b.ParentMode && a.ChildMode == b.ChildMode && a.Operation == b.Operation
}

func sameMetrics(a, b []MetricBinding) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func sortedKeys(values map[string]int) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func validateJSONShape(raw json.RawMessage) (map[string]json.RawMessage, error) {
	var value map[string]json.RawMessage
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, err
	}
	return value, nil
}

func unknownClaim(stage, step, reason, class, next string, blocked []string) Claim {
	return Claim{State: "UNKNOWN", Stage: stage, Step: step, Reason: reason, UnknownClass: class, NextOperation: next, BlockedBy: blocked}
}

func refutedClaim(stage, step, reason, next string) Claim {
	return Claim{State: "REFUTED", Stage: stage, Step: step, Reason: reason, NextOperation: next, BlockedBy: []string{}}
}

func acceptedClaim(stage, step, reason string) Claim {
	return Claim{State: "CLOSED", Stage: stage, Step: step, Reason: reason, NextOperation: "NONE", BlockedBy: []string{}}
}

func validatePrecedence(state string, claim Claim) error {
	if state == "UNKNOWN" && !ValidateUnknownClaim(claim) {
		return fmt.Errorf("UNKNOWN claim does not carry the required six fields")
	}
	if state == "REFUTED" && claim.State != "REFUTED" {
		return fmt.Errorf("REFUTED result has non-refuted claim")
	}
	if state == "ACCEPTED" && claim.State != "CLOSED" {
		return fmt.Errorf("accepted result has non-closed claim")
	}
	return nil
}

func join(values []string) string {
	return strings.Join(values, ",")
}
