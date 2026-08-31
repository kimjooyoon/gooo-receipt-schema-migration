package migration

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

func ValidateDeclarations(source SourceDecl, contract Contract) error {
	if source.Version == "v2" {
		return validateV2Declarations(source, contract)
	}
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

func validateV2Declarations(source SourceDecl, contract Contract) error {
	if source.Schema != SourceSchemaV2 || source.Version != "v2" || contract.Schema != ContractSchemaV2 || contract.Version != "v2" || contract.ID != source.DenominatorID || !contract.Fixed {
		return fmt.Errorf("invalid v2 migration source or contract")
	}
	if source.CellCount != MigrationV2Cells || contract.CellCount != MigrationV2Cells || len(source.Cells) != MigrationV2Cells || len(contract.Cells) != MigrationV2Cells || len(source.Scenarios) != MigrationV2Cells || len(contract.Scenarios) != MigrationV2Cells {
		return fmt.Errorf("v2 denominator must contain exactly sixteen cells and scenarios")
	}
	if !sameCounts(source.StageCounts, contract.StageCounts) || !sameCounts(source.RoleCounts, contract.RoleCounts) || !sameCounts(source.StageCounts, map[string]int{"FOUNDATION": 6, "COHERENCE": 5, "REGRESSION": 5}) || !sameCounts(source.RoleCounts, map[string]int{"DRIVER": 5, "OUTCOME": 5, "GUARDRAIL": 6}) {
		return fmt.Errorf("v2 proof and indicator balance must be exact")
	}
	if !sameStrings(source.Precedence, []string{"REFUTED", "UNKNOWN", "CLOSED"}) || !sameStrings(contract.Precedence, source.Precedence) {
		return fmt.Errorf("resolution precedence must be REFUTED > UNKNOWN > CLOSED")
	}
	if !sameStrings(source.UnknownFields, []string{"stage", "step", "reason", "unknown_class", "next_operation", "blocked_by"}) || !sameStrings(contract.UnknownFields, source.UnknownFields) {
		return fmt.Errorf("UNKNOWN must carry the six required fields")
	}
	if source.Authority.RepositoryWrites != 0 || source.Authority.LocalTestExecutions != 0 || source.Authority.CrossProjectRequiredGates != 0 || source.Authority.ProductGenerationAuthorized || source.Authority.RootReadmePolicy != "EXCLUDED_FROM_REPOSITORY_INVENTORY" {
		return fmt.Errorf("authority declaration must remain zero and non-escalating")
	}
	if len(source.Schemas) != 2 || len(contract.Schemas) != 2 || !sameAdapters(source.Schemas, contract.Schemas) || source.Schemas[0].Version != "v2" || source.Schemas[0].Owner != "parent" || source.Schemas[1].Version != "v3" || source.Schemas[1].Owner != "child" {
		return fmt.Errorf("v2 must preserve the v2 parent and v3 child ownership adapters")
	}
	if contains(source.Schemas[0].OwnedFields, "outcome") || contains(source.Schemas[0].OwnedFields, "parent_outcome") || contains(source.Schemas[0].OwnedFields, "cause_code") || contains(source.Schemas[0].OwnedFields, "causal_chain") || contains(source.Schemas[0].OwnedFields, "next_operation") {
		return fmt.Errorf("v2 parent cannot own v3 outcome or causal fields")
	}
	for _, field := range []string{"parent_outcome", "outcome", "cause_code", "causal_chain", "next_operation"} {
		if !contains(source.Schemas[1].OwnedFields, field) {
			return fmt.Errorf("v3 child must own %s", field)
		}
	}
	if err := validateV2MigrationRecord(source.Migration); err != nil {
		return err
	}
	if err := validateGuardianFixture(source.GuardianFixture); err != nil {
		return err
	}
	if !sameMigrationRecord(source.Migration, contract.Migration) || !sameGuardianFixture(source.GuardianFixture, contract.GuardianFixture) {
		return fmt.Errorf("v2 migration and Guardian fixture differ between source and contract")
	}
	for i, cell := range source.Cells {
		if cell.Ordinal != i+1 || cell.ID == "" || cell.Stage == "" || cell.Role == "" || cell.SemanticEdge == "" || !sameCell(cell, contract.Cells[i]) {
			return fmt.Errorf("v2 cell %d has invalid identity or differs from contract", i+1)
		}
	}
	for i, id := range v1CellIDs() {
		if source.Cells[i].ID != id {
			return fmt.Errorf("v1 cell %d was not preserved", i+1)
		}
	}
	for i, scenario := range source.Scenarios {
		if scenario.Ordinal != i+1 || scenario.ID == "" || scenario.Cell == "" || scenario.Expected == "" || scenario.Operation == "" || !sameScenario(scenario, contract.Scenarios[i]) {
			return fmt.Errorf("v2 scenario %d has invalid identity or differs from contract", i+1)
		}
		if i < FixedCells && normalizeScenarioKind(scenario.Kind) != "RECEIPT" {
			return fmt.Errorf("v1 receipt scenario %q changed kind", scenario.ID)
		}
		if i >= FixedCells && normalizeScenarioKind(scenario.Kind) != "GUARDIAN" {
			return fmt.Errorf("v2 added scenario %q must be a Guardian scenario", scenario.ID)
		}
	}
	for i, id := range v1ScenarioIDs() {
		if source.Scenarios[i].ID != id {
			return fmt.Errorf("v1 scenario %d was not preserved", i+1)
		}
	}
	if !sameStrings([]string{source.Scenarios[12].ID, source.Scenarios[13].ID, source.Scenarios[14].ID, source.Scenarios[15].ID}, []string{"BASE_CONTROLLED_GUARDIAN_EXECUTION", "FEATURE_PR_VARIABLE_LIVENESS", "PASS_ARTIFACT_DIGEST_PROPAGATION", "REFERENCE_ERROR_FAIL_CLOSED"}) {
		return fmt.Errorf("v2 added scenario IDs are not exact")
	}
	if err := validateHarnessCases(source.HarnessCases, contract.HarnessCases, source.Scenarios); err != nil {
		return err
	}
	if len(source.Metrics) != 23 || !sameMetrics(source.Metrics, contract.Metrics) {
		return fmt.Errorf("v2 metrics must expose exact denominator and Guardian harness counts")
	}
	for _, metric := range source.Metrics {
		if metric.ID == "" || metric.MetaActivity == "" || metric.SourcePath == "" || metric.IRPath == "" || metric.GeneratedArtifact == "" || metric.Evaluator == "" || strings.Contains(strings.ToLower(metric.ID), "score") || strings.Contains(strings.ToLower(metric.ID), "percentage") {
			return fmt.Errorf("metric %q is not fully bound or uses a forbidden score/percentage", metric.ID)
		}
	}
	return nil
}

func validateV2MigrationRecord(record MigrationRecord) error {
	if record.FromVersion != "v1" || record.ToVersion != "v2" || record.Added != 4 || record.Retired != 0 || record.Split != 0 || !sameStrings(record.AddedCellIDs, []string{"BASE_CONTROLLED_GUARDIAN_EXECUTION", "FEATURE_PR_VARIABLE_LIVENESS", "PASS_ARTIFACT_DIGEST_PROPAGATION", "REFERENCE_ERROR_FAIL_CLOSED"}) {
		return fmt.Errorf("migration record must be ADD=4 RETIRE=0 SPLIT=0 with the exact four new cells")
	}
	if !sameCounts(record.StageCountsBefore, map[string]int{"FOUNDATION": 4, "COHERENCE": 4, "REGRESSION": 4}) || !sameCounts(record.StageCountsAfter, map[string]int{"FOUNDATION": 6, "COHERENCE": 5, "REGRESSION": 5}) || !sameCounts(record.StageDelta, map[string]int{"FOUNDATION": 2, "COHERENCE": 1, "REGRESSION": 1}) || !sameCounts(record.RoleCountsBefore, map[string]int{"DRIVER": 4, "OUTCOME": 4, "GUARDRAIL": 4}) || !sameCounts(record.RoleCountsAfter, map[string]int{"DRIVER": 5, "OUTCOME": 5, "GUARDRAIL": 6}) || !sameCounts(record.RoleDelta, map[string]int{"DRIVER": 1, "OUTCOME": 1, "GUARDRAIL": 2}) {
		return fmt.Errorf("migration record proof/indicator balance is not exact")
	}
	return nil
}

func validateGuardianFixture(fixture GuardianFixture) error {
	if fixture.Repository != "kimjooyoon/meta-ontology-go" || fixture.Ref != "dev" || fixture.Commit != "7f45792e3c23100cbb10cca8b229132060982a7b" || fixture.ManifestPath != "fixtures/meta-ontology-go/dev-7f45792.json" {
		return fmt.Errorf("Guardian fixture is not pinned to dev@7f45792")
	}
	return nil
}

func validateHarnessCases(source, contract []HarnessCaseDecl, scenarios []ScenarioDecl) error {
	if len(source) != 8 || len(contract) != 8 {
		return fmt.Errorf("Guardian harness must declare exactly eight executable cases")
	}
	seen := map[string]bool{}
	for i, item := range source {
		if item.Ordinal != i+1 || item.ID == "" || seen[item.ID] || item.ScenarioID == "" || item.Cell == "" || item.Expected == "" || item.Mode == "" || item.Operation == "" || !sameHarnessCase(item, contract[i]) {
			return fmt.Errorf("Guardian harness case %d has invalid identity or differs from contract", i+1)
		}
		seen[item.ID] = true
		if !scenarioIDExists(scenarios, item.ScenarioID) {
			return fmt.Errorf("Guardian harness case %q is not bound to a scenario", item.ID)
		}
	}
	expected := []string{"GUARDIAN_CURRENT_SNAPSHOT_REFERENCE_ERROR", "GUARDIAN_CORRECTED_SCOPE_CLOSED", "GUARDIAN_NULL_DIGEST_REFUTED", "GUARDIAN_DIGEST_MISMATCH_REFUTED", "GUARDIAN_BASE_WORKFLOW_CANDIDATE_FILES_SEPARATE", "GUARDIAN_PROTECTED_MIGRATION_PATH_FAIL_CLOSED", "GUARDIAN_UNSUPPORTED_FUTURE_SCHEMA_UNKNOWN", "GUARDIAN_DIGEST_MATCH_CLOSED"}
	ids := make([]string, 0, len(source))
	for _, item := range source {
		ids = append(ids, item.ID)
	}
	if !sameStrings(ids, expected) {
		return fmt.Errorf("Guardian harness case IDs are not exact")
	}
	return nil
}

func scenarioIDExists(scenarios []ScenarioDecl, target string) bool {
	for _, scenario := range scenarios {
		if scenario.ID == target {
			return true
		}
	}
	return false
}

func sameHarnessCase(a, b HarnessCaseDecl) bool {
	return a.Ordinal == b.Ordinal && a.ID == b.ID && a.ScenarioID == b.ScenarioID && a.Cell == b.Cell && a.Expected == b.Expected && a.Mode == b.Mode && a.Operation == b.Operation
}

func v1CellIDs() []string {
	return []string{"NORMAL_V2_PARENT_V3_CHILD", "NORMAL_REPLAY_DETERMINISTIC", "UNKNOWN_PARENT_DIGEST_MISSING", "UNKNOWN_CHILD_DIGEST_STALE", "UNKNOWN_UNSUPPORTED_FUTURE_SCHEMA", "REFUTED_CHILD_REWRITES_PARENT", "REFUTED_FUTURE_FIELD_IN_OLD_SCHEMA", "REFUTED_PARENT_DIGEST_MISMATCH", "REFUTED_SECOND_CHILD_CARDINALITY_ONE", "REFUTED_SELF_ATTESTATION", "REFUTED_DENOMINATOR_DOWNGRADE", "REFUTED_UNKNOWN_PROMOTED_FIXED_POINT"}
}

func v1ScenarioIDs() []string {
	return v1CellIDs()
}

func normalizeScenarioKind(kind string) string {
	if kind == "" {
		return "RECEIPT"
	}
	return kind
}

func sameMigrationRecord(a, b MigrationRecord) bool {
	return a.FromVersion == b.FromVersion && a.ToVersion == b.ToVersion && a.Added == b.Added && a.Retired == b.Retired && a.Split == b.Split && sameStrings(a.AddedCellIDs, b.AddedCellIDs) && sameCounts(a.StageCountsBefore, b.StageCountsBefore) && sameCounts(a.StageCountsAfter, b.StageCountsAfter) && sameCounts(a.StageDelta, b.StageDelta) && sameCounts(a.RoleCountsBefore, b.RoleCountsBefore) && sameCounts(a.RoleCountsAfter, b.RoleCountsAfter) && sameCounts(a.RoleDelta, b.RoleDelta)
}

func ValidateContract(contract Contract) error {
	if (contract.Schema != ContractSchema && contract.Schema != ContractSchemaV2) || !contract.Fixed || (contract.Version == "v1" && contract.CellCount != FixedCells) || (contract.Version == "v2" && contract.CellCount != MigrationV2Cells) {
		return fmt.Errorf("invalid fixed denominator contract")
	}
	return nil
}

func ValidateIR(ir IR) error {
	expectedSchema, expectedCells := IRSchema, FixedCells
	if ir.Version == "v2" {
		expectedSchema, expectedCells = IRSchemaV2, MigrationV2Cells
	}
	if ir.Schema != expectedSchema || (ir.Version != "v1" && ir.Version != "v2") || ir.DenominatorID == "" || ir.CellCount != expectedCells || len(ir.Cells) != expectedCells || len(ir.Scenarios) != expectedCells || len(ir.Adapters) != 2 {
		return fmt.Errorf("semantic IR shape does not match the declared denominator")
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
	return a.Ordinal == b.Ordinal && a.ID == b.ID && a.Cell == b.Cell && a.Class == b.Class && a.Expected == b.Expected && a.ParentMode == b.ParentMode && a.ChildMode == b.ChildMode && a.Operation == b.Operation && normalizeScenarioKind(a.Kind) == normalizeScenarioKind(b.Kind)
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
