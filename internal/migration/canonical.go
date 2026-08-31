package migration

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

func DigestBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func DigestValue(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return DigestBytes(data), nil
}

func DigestRawReceipt(raw json.RawMessage) (string, error) {
	var value map[string]json.RawMessage
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", fmt.Errorf("decode receipt for digest: %w", err)
	}
	delete(value, "digest")
	return DigestValue(value)
}

func unsignedIRDigest(ir IR) (string, error) {
	ir.IRDigest = ""
	return DigestValue(ir)
}

func unsignedAdapterDigest(artifact AdapterArtifact) (string, error) {
	artifact.ArtifactDigest = ""
	return DigestValue(artifact)
}

func unsignedValidatorDigest(artifact ValidatorArtifact) (string, error) {
	artifact.ArtifactDigest = ""
	return DigestValue(artifact)
}

func unsignedScenarioBundleDigest(bundle ScenarioBundle) (string, error) {
	bundle.ArtifactDigest = ""
	return DigestValue(bundle)
}

func unsignedManifestDigest(manifest ArtifactManifest) (string, error) {
	manifest.ArtifactDigest = ""
	return DigestValue(manifest)
}

func unsignedProposalDigest(proposal AdoptionProposal) (string, error) {
	proposal.ProposalDigest = ""
	return DigestValue(proposal)
}
