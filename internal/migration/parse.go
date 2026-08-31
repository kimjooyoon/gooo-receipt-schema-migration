package migration

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func ParseSource(path string) (SourceDecl, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return SourceDecl{}, err
	}
	decl := SourceDecl{SourceDigest: DigestBytes(data)}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(strings.SplitN(scanner.Text(), "#", 2)[0])
		line = strings.TrimSpace(strings.SplitN(line, "//", 2)[0])
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		switch fields[0] {
		case "gooo":
			if len(fields) != 3 || fields[1] != "receipt_schema_migration" || fields[2] != "v1" {
				return SourceDecl{}, fmt.Errorf("line %d: invalid gooo header", lineNumber)
			}
			decl.Schema, decl.Version = SourceSchema, fields[2]
		case "denominator":
			values, err := parseKeyValues(fields[1:])
			if err != nil {
				return SourceDecl{}, fmt.Errorf("line %d: %w", lineNumber, err)
			}
			decl.DenominatorID = values["id"]
			decl.CellCount, err = parseInt(values, "cell_count")
			if err != nil {
				return SourceDecl{}, fmt.Errorf("line %d: %w", lineNumber, err)
			}
		case "axis":
			if len(fields) != 3 || (fields[1] != "stage" && fields[1] != "role") {
				return SourceDecl{}, fmt.Errorf("line %d: invalid axis declaration", lineNumber)
			}
			counts, err := parseCounts(fields[2])
			if err != nil {
				return SourceDecl{}, fmt.Errorf("line %d: %w", lineNumber, err)
			}
			if fields[1] == "stage" {
				decl.StageCounts = counts
			} else {
				decl.RoleCounts = counts
			}
		case "authority":
			values, err := parseKeyValues(fields[1:])
			if err != nil {
				return SourceDecl{}, fmt.Errorf("line %d: %w", lineNumber, err)
			}
			if decl.Authority.RepositoryWrites, err = parseInt(values, "repository_writes"); err != nil {
				return SourceDecl{}, fmt.Errorf("line %d: %w", lineNumber, err)
			}
			if decl.Authority.LocalTestExecutions, err = parseInt(values, "local_test_executions"); err != nil {
				return SourceDecl{}, fmt.Errorf("line %d: %w", lineNumber, err)
			}
			if decl.Authority.CrossProjectRequiredGates, err = parseInt(values, "cross_project_required_gates"); err != nil {
				return SourceDecl{}, fmt.Errorf("line %d: %w", lineNumber, err)
			}
			decl.Authority.ProductGenerationAuthorized = values["product_generation_authorized"] == "true"
			decl.Authority.RootReadmePolicy = values["root_readme_policy"]
		case "precedence":
			if len(fields) != 2 {
				return SourceDecl{}, fmt.Errorf("line %d: invalid precedence", lineNumber)
			}
			decl.Precedence = strings.Split(fields[1], ">")
		case "unknown_fields":
			if len(fields) != 2 {
				return SourceDecl{}, fmt.Errorf("line %d: invalid unknown_fields", lineNumber)
			}
			decl.UnknownFields = strings.Split(fields[1], ",")
		case "schema":
			values, err := parseKeyValues(fields[1:])
			if err != nil {
				return SourceDecl{}, fmt.Errorf("line %d: %w", lineNumber, err)
			}
			decl.Schemas = append(decl.Schemas, AdapterDecl{
				Version: values["version"], Owner: values["owner"], InputSchema: values["input_schema"], OutputSchema: values["output_schema"],
				Operations: splitList(values["operations"]), OwnedFields: splitList(values["owned_fields"]), ForbiddenFields: splitList(values["forbidden_fields"]),
			})
		case "cell":
			values, err := parseKeyValues(fields[1:])
			if err != nil {
				return SourceDecl{}, fmt.Errorf("line %d: %w", lineNumber, err)
			}
			ordinal, err := parseInt(values, "ordinal")
			if err != nil {
				return SourceDecl{}, fmt.Errorf("line %d: %w", lineNumber, err)
			}
			decl.Cells = append(decl.Cells, Cell{Ordinal: ordinal, ID: values["id"], Stage: values["stage"], Role: values["role"], SemanticEdge: values["edge"], DependsOn: splitDepends(values["depends_on"])})
		case "scenario":
			values, err := parseKeyValues(fields[1:])
			if err != nil {
				return SourceDecl{}, fmt.Errorf("line %d: %w", lineNumber, err)
			}
			ordinal, err := parseInt(values, "ordinal")
			if err != nil {
				return SourceDecl{}, fmt.Errorf("line %d: %w", lineNumber, err)
			}
			decl.Scenarios = append(decl.Scenarios, ScenarioDecl{Ordinal: ordinal, ID: values["id"], Cell: values["cell"], Class: values["class"], Expected: values["expected"], ParentMode: values["parent_mode"], ChildMode: values["child_mode"], Operation: values["operation"]})
		case "metric":
			values, err := parseKeyValues(fields[1:])
			if err != nil {
				return SourceDecl{}, fmt.Errorf("line %d: %w", lineNumber, err)
			}
			decl.Metrics = append(decl.Metrics, MetricBinding{ID: values["id"], MetaActivity: values["activity"], SourcePath: values["source"], IRPath: values["ir"], GeneratedArtifact: values["artifact"], Evaluator: values["evaluator"]})
		default:
			return SourceDecl{}, fmt.Errorf("line %d: unknown declaration %q", lineNumber, fields[0])
		}
	}
	if err := scanner.Err(); err != nil {
		return SourceDecl{}, err
	}
	return decl, nil
}

func parseKeyValues(fields []string) (map[string]string, error) {
	values := make(map[string]string, len(fields))
	for _, field := range fields {
		parts := strings.SplitN(field, "=", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return nil, fmt.Errorf("invalid key/value %q", field)
		}
		values[parts[0]] = strings.Trim(parts[1], "\"")
	}
	return values, nil
}

func parseInt(values map[string]string, key string) (int, error) {
	value, ok := values[key]
	if !ok {
		return 0, fmt.Errorf("missing %s", key)
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid %s %q", key, value)
	}
	return n, nil
}

func parseCounts(value string) (map[string]int, error) {
	counts := map[string]int{}
	for _, item := range strings.Split(value, ",") {
		parts := strings.SplitN(item, "=", 2)
		if len(parts) != 2 || parts[0] == "" {
			return nil, fmt.Errorf("invalid axis count %q", item)
		}
		n, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, fmt.Errorf("invalid axis count %q", item)
		}
		counts[parts[0]] = n
	}
	return counts, nil
}

func splitList(value string) []string {
	if value == "" || value == "-" {
		return []string{}
	}
	return strings.Split(value, ",")
}

func splitDepends(value string) []string {
	return splitList(value)
}

func LoadContract(path string) (Contract, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Contract{}, err
	}
	var contract Contract
	if err := json.Unmarshal(data, &contract); err != nil {
		return Contract{}, fmt.Errorf("decode contract: %w", err)
	}
	if contract.Schema != ContractSchema {
		return Contract{}, fmt.Errorf("unexpected contract schema %q", contract.Schema)
	}
	return contract, nil
}

func ContractDigest(contract Contract) (string, error) {
	return DigestValue(contract)
}

func sameStrings(a, b []string) bool {
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
