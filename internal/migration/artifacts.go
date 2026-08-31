package migration

import (
	"os"
	"path/filepath"
)

func artifactRef(path, reportPath string) (ArtifactRef, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ArtifactRef{}, err
	}
	return ArtifactRef{Path: reportPath, Digest: DigestBytes(data), SizeBytes: int64(len(data))}, nil
}

func buildArtifactManifest(ir IR, outputDir string, refs []ArtifactRef) (ArtifactManifest, error) {
	manifest := ArtifactManifest{Schema: ArtifactManifestSchema, IRDigest: ir.IRDigest, Artifacts: append([]ArtifactRef(nil), refs...)}
	manifest.ArtifactDigest, _ = unsignedManifestDigest(manifest)
	path := filepath.Join(outputDir, "artifact-manifest.json")
	if err := WriteJSON(path, manifest); err != nil {
		return ArtifactManifest{}, err
	}
	return manifest, nil
}

func validateArtifactManifest(manifest ArtifactManifest, ir IR) error {
	if manifest.Schema != ArtifactManifestSchema || manifest.IRDigest != ir.IRDigest || len(manifest.Artifacts) == 0 || manifest.ArtifactDigest == "" {
		return os.ErrInvalid
	}
	digest, err := unsignedManifestDigest(manifest)
	if err != nil || digest != manifest.ArtifactDigest {
		return os.ErrInvalid
	}
	return nil
}
