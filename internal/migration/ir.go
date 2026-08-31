package migration

import "fmt"

func BuildIR(source SourceDecl, contract Contract) (IR, error) {
	if err := ValidateDeclarations(source, contract); err != nil {
		return IR{}, err
	}
	contractDigest, err := ContractDigest(contract)
	if err != nil {
		return IR{}, err
	}
	irSchema := IRSchema
	if source.Version == "v2" {
		irSchema = IRSchemaV2
	}
	ir := IR{
		Schema: irSchema, Version: source.Version, SourceDigest: source.SourceDigest, ContractDigest: contractDigest,
		DenominatorID: source.DenominatorID, CellCount: source.CellCount,
		StageCounts: cloneCounts(source.StageCounts), RoleCounts: cloneCounts(source.RoleCounts),
		Authority: source.Authority, Precedence: append([]string(nil), source.Precedence...),
		UnknownFields: append([]string(nil), source.UnknownFields...), Adapters: cloneAdapters(source.Schemas),
		Cells: cloneCells(source.Cells), Scenarios: cloneScenarios(source.Scenarios), Metrics: cloneMetrics(source.Metrics),
		Migration: source.Migration, GuardianFixture: source.GuardianFixture, HarnessCases: append([]HarnessCaseDecl(nil), source.HarnessCases...),
	}
	ir.IRDigest, err = unsignedIRDigest(ir)
	if err != nil {
		return IR{}, err
	}
	return ir, nil
}

func cloneCounts(input map[string]int) map[string]int {
	output := make(map[string]int, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}

func cloneAdapters(input []AdapterDecl) []AdapterDecl {
	output := make([]AdapterDecl, len(input))
	for i, value := range input {
		output[i] = value
		output[i].Operations = append([]string(nil), value.Operations...)
		output[i].OwnedFields = append([]string(nil), value.OwnedFields...)
		output[i].ForbiddenFields = append([]string(nil), value.ForbiddenFields...)
	}
	return output
}

func cloneCells(input []Cell) []Cell {
	output := make([]Cell, len(input))
	for i, value := range input {
		output[i] = value
		output[i].DependsOn = append([]string(nil), value.DependsOn...)
	}
	return output
}

func cloneScenarios(input []ScenarioDecl) []ScenarioDecl {
	return append([]ScenarioDecl(nil), input...)
}

func cloneMetrics(input []MetricBinding) []MetricBinding {
	return append([]MetricBinding(nil), input...)
}

func cellByID(ir IR, id string) (Cell, error) {
	for _, cell := range ir.Cells {
		if cell.ID == id {
			return cell, nil
		}
	}
	return Cell{}, fmt.Errorf("cell %q not found", id)
}

func scenarioByID(ir IR, id string) (ScenarioDecl, error) {
	for _, scenario := range ir.Scenarios {
		if scenario.ID == id {
			return scenario, nil
		}
	}
	return ScenarioDecl{}, fmt.Errorf("scenario %q not found", id)
}
