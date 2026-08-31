package migration

import "encoding/json"

const (
	SourceSchema       = "gooo/receipt-schema-migration/source/v1"
	ContractSchema     = "gooo/receipt-schema-migration/denominator/v1"
	IRSchema           = "gooo/receipt-schema-migration/semantic-ir/v1"
	AdapterSchema      = "gooo/receipt-schema-migration/generated-adapter/v1"
	ValidatorSchema    = "gooo/receipt-schema-migration/generated-validator/v1"
	ScenarioSchema     = "gooo/receipt-schema-migration/scenario-receipt/v1"
	ReportSchema       = "gooo/receipt-schema-migration/report/v1"
	ProposalSchema     = "gooo/receipt-schema-migration/adoption-proposal/v1"
	ArtifactManifestSchema = "gooo/receipt-schema-migration/artifact-manifest/v1"
	CISummarySchema    = "gooo/receipt-schema-migration/ci-summary/v1"
	ParentV2Schema     = "gooo/receipt/parent/v2"
	ChildV3Schema      = "gooo/receipt/child/v3"
	FixedCells         = 12
)

type Authority struct {
	RepositoryWrites            int    `json:"repository_writes"`
	LocalTestExecutions         int    `json:"local_test_executions"`
	CrossProjectRequiredGates   int    `json:"cross_project_required_gates"`
	ProductGenerationAuthorized bool   `json:"product_generation_authorized"`
	RootReadmePolicy            string `json:"root_readme_policy"`
}

type Claim struct {
	State         string   `json:"state"`
	Stage         string   `json:"stage"`
	Step          string   `json:"step"`
	Reason        string   `json:"reason"`
	UnknownClass  string   `json:"unknown_class"`
	NextOperation string   `json:"next_operation"`
	BlockedBy     []string `json:"blocked_by"`
}

func (c Claim) HasUnknownTuple() bool {
	return c.State == "UNKNOWN" && c.Stage != "" && c.Step != "" && c.Reason != "" &&
		c.UnknownClass != "" && c.NextOperation != "" && c.BlockedBy != nil
}

type Cell struct {
	Ordinal      int      `json:"ordinal"`
	ID           string   `json:"id"`
	Stage        string   `json:"stage"`
	Role         string   `json:"role"`
	SemanticEdge string   `json:"semantic_edge"`
	DependsOn    []string `json:"depends_on"`
}

type AdapterDecl struct {
	Version        string   `json:"version"`
	Owner          string   `json:"owner"`
	InputSchema    string   `json:"input_schema"`
	OutputSchema   string   `json:"output_schema"`
	Operations     []string `json:"operations"`
	OwnedFields    []string `json:"owned_fields"`
	ForbiddenFields []string `json:"forbidden_fields"`
}

type ScenarioDecl struct {
	Ordinal    int    `json:"ordinal"`
	ID         string `json:"id"`
	Cell       string `json:"cell"`
	Class      string `json:"class"`
	Expected   string `json:"expected"`
	ParentMode string `json:"parent_mode"`
	ChildMode  string `json:"child_mode"`
	Operation  string `json:"operation"`
}

type MetricBinding struct {
	ID               string `json:"id"`
	MetaActivity     string `json:"meta_activity"`
	SourcePath       string `json:"source_path"`
	IRPath           string `json:"ir_path"`
	GeneratedArtifact string `json:"generated_artifact"`
	Evaluator        string `json:"evaluator"`
}

type SourceDecl struct {
	Schema        string            `json:"schema"`
	Version       string            `json:"version"`
	DenominatorID string            `json:"denominator_id"`
	CellCount     int               `json:"cell_count"`
	StageCounts   map[string]int    `json:"stage_counts"`
	RoleCounts    map[string]int    `json:"role_counts"`
	Authority     Authority         `json:"authority"`
	Precedence    []string          `json:"precedence"`
	UnknownFields []string          `json:"unknown_fields"`
	Schemas       []AdapterDecl     `json:"schemas"`
	Cells         []Cell            `json:"cells"`
	Scenarios     []ScenarioDecl    `json:"scenarios"`
	Metrics       []MetricBinding   `json:"metrics"`
	SourceDigest  string            `json:"source_digest"`
}

type Contract struct {
	Schema        string          `json:"schema"`
	ID            string          `json:"id"`
	Version       string          `json:"version"`
	CellCount     int             `json:"cell_count"`
	Fixed         bool            `json:"fixed"`
	StageCounts   map[string]int  `json:"stage_counts"`
	RoleCounts    map[string]int  `json:"role_counts"`
	Precedence    []string        `json:"precedence"`
	UnknownFields []string        `json:"unknown_fields"`
	Schemas       []AdapterDecl   `json:"schemas"`
	Cells         []Cell          `json:"cells"`
	Scenarios     []ScenarioDecl  `json:"scenarios"`
	Metrics       []MetricBinding `json:"metrics"`
}

type IR struct {
	Schema         string          `json:"schema"`
	Version        string          `json:"version"`
	SourceDigest   string          `json:"source_digest"`
	ContractDigest string          `json:"contract_digest"`
	DenominatorID  string          `json:"denominator_id"`
	CellCount      int             `json:"cell_count"`
	StageCounts    map[string]int  `json:"stage_counts"`
	RoleCounts     map[string]int  `json:"role_counts"`
	Authority      Authority       `json:"authority"`
	Precedence     []string        `json:"precedence"`
	UnknownFields  []string        `json:"unknown_fields"`
	Adapters       []AdapterDecl   `json:"adapters"`
	Cells          []Cell          `json:"cells"`
	Scenarios      []ScenarioDecl  `json:"scenarios"`
	Metrics        []MetricBinding `json:"metrics"`
	IRDigest       string          `json:"ir_digest,omitempty"`
}

type AdapterArtifact struct {
	Schema          string   `json:"schema"`
	Version         string   `json:"version"`
	Owner           string   `json:"owner"`
	InputSchema     string   `json:"input_schema"`
	OutputSchema    string   `json:"output_schema"`
	Operations      []string `json:"operations"`
	OwnedFields     []string `json:"owned_fields"`
	ForbiddenFields []string `json:"forbidden_fields"`
	IRDigest        string   `json:"ir_digest"`
	ArtifactDigest  string   `json:"artifact_digest,omitempty"`
}

type ValidatorArtifact struct {
	Schema         string   `json:"schema"`
	Version        string   `json:"version"`
	SupportedSchemas []string `json:"supported_schemas"`
	Precedence     []string `json:"precedence"`
	UnknownFields  []string `json:"unknown_fields"`
	Operations     []string `json:"operations"`
	IRDigest       string   `json:"ir_digest"`
	ArtifactDigest string   `json:"artifact_digest,omitempty"`
}

type ParentReceipt struct {
	Schema                  string         `json:"schema"`
	SchemaVersion           string         `json:"schema_version"`
	ReceiptID               string         `json:"receipt_id"`
	Kind                    string         `json:"kind"`
	DenominatorID           string         `json:"denominator_id"`
	DenominatorCellCount    int            `json:"denominator_cell_count"`
	DenominatorStageCounts  map[string]int `json:"denominator_stage_counts"`
	DenominatorRoleCounts   map[string]int `json:"denominator_role_counts"`
	Immutable               bool           `json:"immutable"`
	CreatedBy               string         `json:"created_by"`
	Digest                  string         `json:"digest"`
}

type ChildReceipt struct {
	Schema                 string         `json:"schema"`
	SchemaVersion          string         `json:"schema_version"`
	ReceiptID              string         `json:"receipt_id"`
	ParentReceiptID        string         `json:"parent_receipt_id"`
	ParentDigest           string         `json:"parent_digest"`
	ParentOutcome          string         `json:"parent_outcome"`
	Outcome                string         `json:"outcome"`
	CauseCode              string         `json:"cause_code"`
	CausalChain            []string       `json:"causal_chain"`
	NextOperation          string         `json:"next_operation"`
	DenominatorID          string         `json:"denominator_id"`
	DenominatorCellCount   int            `json:"denominator_cell_count"`
	DenominatorStageCounts map[string]int `json:"denominator_stage_counts"`
	DenominatorRoleCounts  map[string]int `json:"denominator_role_counts"`
	Attestation            string         `json:"attestation"`
	ChildOrdinal           int            `json:"child_ordinal"`
	Digest                 string         `json:"digest"`
}

type ScenarioArtifact struct {
	Schema              string            `json:"schema"`
	ScenarioID          string            `json:"scenario_id"`
	CellID              string            `json:"cell_id"`
	Class               string            `json:"class"`
	ExpectedState       string            `json:"expected_state"`
	Parent              json.RawMessage   `json:"parent"`
	Children            []json.RawMessage `json:"children"`
	ParentWriteAttempts int               `json:"parent_write_attempts"`
	ParentWrites        int               `json:"parent_writes"`
	ParentDigestBefore  string            `json:"parent_digest_before"`
	ParentDigestAfter   string            `json:"parent_digest_after"`
	ReplayDigestBefore  string            `json:"replay_digest_before,omitempty"`
	ReplayDigestAfter   string            `json:"replay_digest_after,omitempty"`
	ReplayDeterministic bool              `json:"replay_deterministic"`
}

type ScenarioResult struct {
	ScenarioID     string `json:"scenario_id"`
	CellID         string `json:"cell_id"`
	Stage          string `json:"stage"`
	Role           string `json:"role"`
	ExpectedState  string `json:"expected_state"`
	State          string `json:"state"`
	ObservedReason string `json:"observed_reason"`
	Claim          Claim  `json:"claim"`
	ParentDigest   string `json:"parent_digest"`
	ChildDigests   []string `json:"child_digests"`
}

type ScenarioBundle struct {
	Schema        string             `json:"schema"`
	IRDigest      string             `json:"ir_digest"`
	Scenarios     []ScenarioArtifact `json:"scenarios"`
	ArtifactDigest string            `json:"artifact_digest,omitempty"`
}

type ArtifactRef struct {
	Path      string `json:"path"`
	Digest    string `json:"digest"`
	SizeBytes int64  `json:"size_bytes"`
}

type ArtifactManifest struct {
	Schema         string       `json:"schema"`
	IRDigest       string       `json:"ir_digest"`
	Artifacts      []ArtifactRef `json:"artifacts"`
	ArtifactDigest string       `json:"artifact_digest,omitempty"`
}

type SchemaOwnership struct {
	SchemaVersion string   `json:"schema_version"`
	Owner         string   `json:"owner"`
	Owns          []string `json:"owns"`
	MustNotOwn    []string `json:"must_not_own"`
}

type AcceptanceCase struct {
	ID            string `json:"id"`
	ExpectedState string `json:"expected_state"`
	Invariant     string `json:"invariant"`
}

type ExternalRelease struct {
	URI    string `json:"uri"`
	Digest string `json:"digest"`
}

type AdoptionProposal struct {
	Schema                    string            `json:"schema"`
	ProposalID                string            `json:"proposal_id"`
	TargetRepository          string            `json:"target_repository"`
	RepositoryWrites          int               `json:"repository_writes"`
	LocalTestExecutions       int               `json:"local_test_executions"`
	CrossProjectRequiredGates int               `json:"cross_project_required_gates"`
	OldSchemaOwnership        []SchemaOwnership `json:"old_schema_ownership"`
	NewSchemaOwnership        []SchemaOwnership `json:"new_schema_ownership"`
	ExactRequiredSemanticChanges []string       `json:"exact_required_semantic_changes"`
	ExpectedProtectedPaths    []string          `json:"expected_protected_paths"`
	AcceptanceCases           []AcceptanceCase  `json:"acceptance_cases"`
	RollbackConditions        []string          `json:"rollback_conditions"`
	RefutationConditions      []string          `json:"refutation_conditions"`
	OptionalExternalRelease   *ExternalRelease  `json:"optional_external_release,omitempty"`
	ProposalDigest            string            `json:"proposal_digest,omitempty"`
}

type Report struct {
	Schema             string             `json:"schema"`
	Decision           string             `json:"decision"`
	Pipeline           map[string]string  `json:"pipeline"`
	SchemaVersions     []string           `json:"schema_versions"`
	FixedDenominator   int                `json:"fixed_denominator"`
	StageCounts        map[string]int     `json:"stage_counts"`
	RoleCounts         map[string]int     `json:"role_counts"`
	Precedence         []string           `json:"precedence"`
	UnknownFields      []string           `json:"unknown_fields"`
	Authority          Authority           `json:"authority"`
	Summary            Summary            `json:"summary"`
	AdapterOperations  []string           `json:"adapter_operations"`
	ArtifactDigests    []ArtifactRef      `json:"artifact_digests"`
	MetricBindings     []MetricBinding    `json:"metric_bindings"`
	Scenarios          []ScenarioResult   `json:"scenarios"`
	Improvement        Claim              `json:"improvement"`
	ReportDigest       string             `json:"report_digest,omitempty"`
}

type Summary struct {
	ParentReceiptCount  int `json:"parent_receipt_count"`
	ChildReceiptCount   int `json:"child_receipt_count"`
	AcceptedCount       int `json:"accepted_count"`
	UnknownCount        int `json:"unknown_count"`
	RefutedCount        int `json:"refuted_count"`
	ImmutableParentWrites int `json:"immutable_parent_writes"`
}

type MetricValue struct {
	ID    string `json:"id"`
	Value any    `json:"value"`
}

type CISummary struct {
	Schema            string          `json:"schema"`
	ReportDigest      string          `json:"report_digest"`
	SchemaVersions    []string        `json:"schema_versions"`
	ParentReceiptCount int             `json:"parent_receipt_count"`
	ChildReceiptCount int             `json:"child_receipt_count"`
	AcceptedCount     int             `json:"accepted_count"`
	UnknownCount      int             `json:"unknown_count"`
	RefutedCount      int             `json:"refuted_count"`
	ImmutableParentWrites int         `json:"immutable_parent_writes"`
	AdapterOperations []string        `json:"adapter_operations"`
	ArtifactDigests   []ArtifactRef   `json:"artifact_digests"`
	MetricBindings    []MetricBinding `json:"metric_bindings"`
	Metrics           []MetricValue   `json:"metrics"`
	Improvement       Claim           `json:"improvement"`
}
