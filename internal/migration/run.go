package migration

import (
	"fmt"
	"path/filepath"
)

func Run(sourcePath, contractPath, outputDir string, external *ExternalRelease) (Report, error) {
	if err := ensureCallerOutput(outputDir); err != nil {
		return Report{}, err
	}
	source, err := ParseSource(sourcePath)
	if err != nil {
		return Report{}, err
	}
	contract, err := LoadContract(contractPath)
	if err != nil {
		return Report{}, err
	}
	if err := ValidateContract(contract); err != nil {
		return Report{}, err
	}
	ir, err := BuildIR(source, contract)
	if err != nil {
		return Report{}, err
	}
	if err := ValidateIR(ir); err != nil {
		return Report{}, err
	}
	if err := WriteJSON(filepath.Join(outputDir, "semantic-ir.json"), ir); err != nil {
		return Report{}, err
	}
	refs, err := collectArtifactRefs(outputDir, []string{"semantic-ir.json"})
	if err != nil {
		return Report{}, err
	}
	adapterRefs, err := GenerateAdapters(ir, outputDir)
	if err != nil {
		return Report{}, err
	}
	refs = append(refs, adapterRefs...)
	bundle, scenarioRefs, err := GenerateScenarios(ir, outputDir)
	if err != nil {
		return Report{}, err
	}
	refs = append(refs, scenarioRefs...)
	if _, _, err := BuildAdoptionProposal(ir, outputDir, external); err != nil {
		return Report{}, err
	}
	proposalRef, err := artifactRef(filepath.Join(outputDir, "adoption-proposal.json"), "adoption-proposal.json")
	if err != nil {
		return Report{}, err
	}
	refs = append(refs, proposalRef)
	manifest, err := buildArtifactManifest(ir, outputDir, refs)
	if err != nil {
		return Report{}, err
	}
	if err := validateArtifactManifest(manifest, ir); err != nil {
		return Report{}, fmt.Errorf("artifact manifest: %w", err)
	}
	manifestRef, err := artifactRef(filepath.Join(outputDir, "artifact-manifest.json"), "artifact-manifest.json")
	if err != nil {
		return Report{}, err
	}
	refs = append(refs, manifestRef)
	if _, _, err := LoadGeneratedAdapters(ir, outputDir); err != nil {
		return Report{}, err
	}
	report, err := EvaluateScenarioBundle(ir, bundle, outputDir)
	if err != nil {
		return Report{}, err
	}
	report.ArtifactDigests = refs
	report.ReportDigest, err = DigestValue(report)
	if err != nil {
		return Report{}, err
	}
	if err := WriteJSON(filepath.Join(outputDir, "report.json"), report); err != nil {
		return Report{}, err
	}
	if err := WriteText(filepath.Join(outputDir, "human-report.md"), RenderReport(report)); err != nil {
		return Report{}, err
	}
	return report, nil
}
