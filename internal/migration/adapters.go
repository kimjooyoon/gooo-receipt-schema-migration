package migration

import (
	"fmt"
	"path/filepath"
)

func GenerateAdapters(ir IR, outputDir string) ([]ArtifactRef, error) {
	if err := ValidateIR(ir); err != nil {
		return nil, err
	}
	refs := make([]ArtifactRef, 0, len(ir.Adapters)+1)
	for _, declaration := range ir.Adapters {
		artifact := AdapterArtifact{
			Schema: AdapterSchema, Version: declaration.Version, Owner: declaration.Owner,
			InputSchema: declaration.InputSchema, OutputSchema: declaration.OutputSchema,
			Operations: append([]string(nil), declaration.Operations...), OwnedFields: append([]string(nil), declaration.OwnedFields...),
			ForbiddenFields: append([]string(nil), declaration.ForbiddenFields...), IRDigest: ir.IRDigest,
		}
		digest, err := unsignedAdapterDigest(artifact)
		if err != nil {
			return nil, err
		}
		artifact.ArtifactDigest = digest
		path := filepath.Join(outputDir, "generated", "adapters", declaration.Version+".json")
		if err := WriteJSON(path, artifact); err != nil {
			return nil, err
		}
		ref, err := artifactRef(path, filepath.Join("generated", "adapters", declaration.Version+".json"))
		if err != nil {
			return nil, err
		}
		refs = append(refs, ref)
	}
	validator := ValidatorArtifact{
		Schema: ValidatorSchema, Version: "v1", SupportedSchemas: []string{ParentV2Schema, ChildV3Schema},
		Precedence: append([]string(nil), ir.Precedence...), UnknownFields: append([]string(nil), ir.UnknownFields...),
		Operations: []string{"VALIDATE_VERSION_DISPATCH", "VALIDATE_V2_OWNERSHIP", "VALIDATE_V3_OWNERSHIP", "VALIDATE_APPEND_ONLY_LINEAGE", "VALIDATE_DIGESTS", "VALIDATE_CARDINALITY", "VALIDATE_DENOMINATOR", "RESOLVE_PRECEDENCE"},
		IRDigest:   ir.IRDigest,
	}
	if ir.Version == "v2" {
		validator.Operations = append(validator.Operations, "VALIDATE_BASE_CONTROLLED_WORKFLOW", "VALIDATE_CANDIDATE_CHANGED_PATHS", "VALIDATE_VARIABLE_LIFETIME", "VALIDATE_GUARDIAN_ARTIFACT_DIGEST", "VALIDATE_FAIL_CLOSED_REFERENCE_ERROR")
	}
	validator.ArtifactDigest, _ = unsignedValidatorDigest(validator)
	validatorPath := filepath.Join(outputDir, "generated", "validator.json")
	if err := WriteJSON(validatorPath, validator); err != nil {
		return nil, err
	}
	ref, err := artifactRef(validatorPath, "generated/validator.json")
	if err != nil {
		return nil, err
	}
	refs = append(refs, ref)
	return refs, nil
}

func LoadGeneratedAdapters(ir IR, outputDir string) ([]AdapterArtifact, ValidatorArtifact, error) {
	adapters := make([]AdapterArtifact, 0, len(ir.Adapters))
	for _, declaration := range ir.Adapters {
		var artifact AdapterArtifact
		path := filepath.Join(outputDir, "generated", "adapters", declaration.Version+".json")
		if err := ReadJSON(path, &artifact); err != nil {
			return nil, ValidatorArtifact{}, err
		}
		if err := ValidateAdapter(artifact, ir); err != nil {
			return nil, ValidatorArtifact{}, fmt.Errorf("adapter %s: %w", declaration.Version, err)
		}
		adapters = append(adapters, artifact)
	}
	var validator ValidatorArtifact
	if err := ReadJSON(filepath.Join(outputDir, "generated", "validator.json"), &validator); err != nil {
		return nil, ValidatorArtifact{}, err
	}
	if err := ValidateValidator(validator, ir); err != nil {
		return nil, ValidatorArtifact{}, err
	}
	return adapters, validator, nil
}

func adapterOperations(ir IR) []string {
	operations := make([]string, 0, 21)
	for _, adapter := range ir.Adapters {
		operations = append(operations, adapter.Operations...)
	}
	operations = append(operations, "VALIDATE_VERSION_DISPATCH", "VALIDATE_V2_OWNERSHIP", "VALIDATE_V3_OWNERSHIP", "VALIDATE_APPEND_ONLY_LINEAGE", "VALIDATE_DIGESTS", "VALIDATE_CARDINALITY", "VALIDATE_DENOMINATOR", "RESOLVE_PRECEDENCE")
	if ir.Version == "v2" {
		operations = append(operations, "VALIDATE_BASE_CONTROLLED_WORKFLOW", "VALIDATE_CANDIDATE_CHANGED_PATHS", "VALIDATE_VARIABLE_LIFETIME", "VALIDATE_GUARDIAN_ARTIFACT_DIGEST", "VALIDATE_FAIL_CLOSED_REFERENCE_ERROR")
	}
	return operations
}
