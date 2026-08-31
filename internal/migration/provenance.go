package migration

import (
	"fmt"
	"os"
	"reflect"
)

func developmentProvenance() DevelopmentProvenance {
	return DevelopmentProvenance{
		Schema:  DevelopmentProvenanceSchema,
		EventID: "development-provenance-20260831-guardian-v3-001",
		DevelopmentLocalGoCommands: map[string]int{
			"gofmt": 0,
			"build": 0,
			"test":  0,
			"vet":   0,
		},
		DevelopmentLocalVMHarnessExecutions: 1,
		Class:                               "NODE_VM_GUARDIAN_HARNESS",
		Purpose:                             "DEBUG_AUTH_MERGE_VS_BASE_COMMIT_MOCK",
		PolicyState:                         "RECORDED_DEVELOPMENT_POLICY_DEVIATION",
		EventMutationPolicy:                 "APPEND_ONLY_NO_RESET_DELETE_REWRITE",
		DevelopmentPolicyDeviationCount:     1,
		ProductRuntimeAuthority: Authority{
			RepositoryWrites:            0,
			LocalTestExecutions:         0,
			CrossProjectRequiredGates:   0,
			ProductGenerationAuthorized: false,
			RootReadmePolicy:            "EXCLUDED_FROM_REPOSITORY_INVENTORY",
		},
		RemainingValidationPolicy: "GITHUB_ACTIONS_ONLY_AFTER_THIS_RECORD",
	}
}

func unsignedDevelopmentProvenanceDigest(provenance DevelopmentProvenance) (string, error) {
	provenance.ProvenanceDigest = ""
	return DigestValue(provenance)
}

func ValidateDevelopmentProvenance(provenance DevelopmentProvenance) error {
	expected := developmentProvenance()
	expectedDigest, err := unsignedDevelopmentProvenanceDigest(expected)
	if err != nil {
		return err
	}
	if provenance.Schema != expected.Schema || provenance.EventID != expected.EventID || provenance.Class != expected.Class || provenance.Purpose != expected.Purpose || provenance.PolicyState != expected.PolicyState || provenance.EventMutationPolicy != expected.EventMutationPolicy || provenance.DevelopmentPolicyDeviationCount != 1 || provenance.DevelopmentLocalVMHarnessExecutions != 1 || provenance.RemainingValidationPolicy != expected.RemainingValidationPolicy || provenance.ProvenanceDigest != expectedDigest || provenance.ProductRuntimeAuthority != expected.ProductRuntimeAuthority || len(provenance.DevelopmentLocalGoCommands) != 4 {
		return fmt.Errorf("development provenance is not the recorded v3 event")
	}
	for _, command := range []string{"gofmt", "build", "test", "vet"} {
		if provenance.DevelopmentLocalGoCommands[command] != 0 {
			return fmt.Errorf("development provenance records local Go command %q", command)
		}
	}
	return nil
}

func WriteDevelopmentProvenance(path string) (DevelopmentProvenance, ArtifactRef, error) {
	provenance := developmentProvenance()
	digest, err := unsignedDevelopmentProvenanceDigest(provenance)
	if err != nil {
		return DevelopmentProvenance{}, ArtifactRef{}, err
	}
	provenance.ProvenanceDigest = digest
	if _, err := os.Stat(path); err == nil {
		var recorded DevelopmentProvenance
		if err := ReadJSON(path, &recorded); err != nil || !reflect.DeepEqual(recorded, provenance) {
			return DevelopmentProvenance{}, ArtifactRef{}, fmt.Errorf("development provenance already exists with a different immutable event")
		}
	} else if !os.IsNotExist(err) {
		return DevelopmentProvenance{}, ArtifactRef{}, err
	} else if err := WriteJSON(path, provenance); err != nil {
		return DevelopmentProvenance{}, ArtifactRef{}, err
	}
	if err := ValidateDevelopmentProvenance(provenance); err != nil {
		return DevelopmentProvenance{}, ArtifactRef{}, err
	}
	ref, err := artifactRef(path, "development-provenance.json")
	if err != nil {
		return DevelopmentProvenance{}, ArtifactRef{}, err
	}
	return provenance, ref, nil
}

func cloneDevelopmentProvenance(input *DevelopmentProvenance) *DevelopmentProvenance {
	if input == nil {
		return nil
	}
	output := *input
	output.DevelopmentLocalGoCommands = map[string]int{}
	for key, value := range input.DevelopmentLocalGoCommands {
		output.DevelopmentLocalGoCommands[key] = value
	}
	return &output
}
