package migration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSourceAndContractDeclareTheFixedTwelveCellProtocol(t *testing.T) {
	root := filepath.Join("..", "..")
	source, err := ParseSource(filepath.Join(root, "examples", "receipt-schema-migration-v1", "migration.gooo"))
	if err != nil {
		t.Fatal(err)
	}
	contract, err := LoadContract(filepath.Join(root, "contracts", "receipt-migration-denominator-v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateDeclarations(source, contract); err != nil {
		t.Fatal(err)
	}
	ir, err := BuildIR(source, contract)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateIR(ir); err != nil {
		t.Fatal(err)
	}
	if ir.CellCount != 12 || len(ir.Cells) != 12 || len(ir.Scenarios) != 12 {
		t.Fatalf("unexpected fixed denominator: %#v", ir)
	}
}

func TestRawReceiptDigestExcludesOnlyDigestField(t *testing.T) {
	raw, digest, err := rawWithDigest(ParentReceipt{Schema: ParentV2Schema, SchemaVersion: "v2", ReceiptID: "parent:test", Kind: "REGRESSION_REPAIR", Immutable: true}, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if digest == "" {
		t.Fatal("missing receipt digest")
	}
	actual, err := DigestRawReceipt(raw)
	if err != nil {
		t.Fatal(err)
	}
	if actual != digest {
		t.Fatalf("digest mismatch: got %s want %s", actual, digest)
	}
}

func TestUnknownClaimRequiresSixFields(t *testing.T) {
	claim := unknownClaim("LINEAGE", "resolve_parent_digest", "MISSING_PARENT_EVIDENCE", "MISSING_PARENT_EVIDENCE", "SUPPLY_IMMUTABLE_PARENT_DIGEST", []string{"parent-digest"})
	if !ValidateUnknownClaim(claim) {
		t.Fatalf("valid UNKNOWN claim rejected: %#v", claim)
	}
	claim.BlockedBy = nil
	if ValidateUnknownClaim(claim) {
		t.Fatal("UNKNOWN claim without blocked_by was accepted")
	}
}

func TestFutureFieldInV2ParentIsVisibleInRawJSON(t *testing.T) {
	parent, _, err := rawWithDigest(ParentReceipt{Schema: ParentV2Schema, SchemaVersion: "v2", ReceiptID: "parent:test", Kind: "REGRESSION_REPAIR", Immutable: true}, map[string]json.RawMessage{"outcome": json.RawMessage(`"future"`)}, "")
	if err != nil {
		t.Fatal(err)
	}
	value, err := validateJSONShape(parent)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := value["outcome"]; !ok {
		t.Fatal("future field was not retained in raw parent")
	}
}

func TestCallerOutputRejectsRepositoryPath(t *testing.T) {
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := ensureCallerOutput(root); err == nil {
		t.Fatal("repository output path was accepted")
	}
}
