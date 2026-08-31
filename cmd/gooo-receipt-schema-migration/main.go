package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/kimjooyoon/gooo-receipt-schema-migration/internal/migration"
)

func main() {
	if len(os.Args) < 2 {
		fatal("command is required: run, attach-harness, or ci-summary")
	}
	switch os.Args[1] {
	case "run":
		run(os.Args[2:])
	case "ci-summary":
		ciSummary(os.Args[2:])
	case "attach-harness":
		attachHarness(os.Args[2:])
	default:
		fatal(fmt.Sprintf("unknown command %q", os.Args[1]))
	}
}

func attachHarness(args []string) {
	set := flag.NewFlagSet("attach-harness", flag.ExitOnError)
	report := set.String("report", "", "report.json path")
	harness := set.String("harness", "", "guardian-harness-report.json path")
	proposal := set.String("proposal", "", "adoption-proposal.json path")
	manifest := set.String("manifest", "", "artifact-manifest.json path")
	human := set.String("human", "", "human-report.md path")
	set.Parse(args)
	if *report == "" || *harness == "" || *proposal == "" || *manifest == "" || *human == "" {
		fatal("attach-harness requires --report, --harness, --proposal, --manifest, and --human")
	}
	result, err := migration.AttachGuardianHarness(*report, *harness, *proposal, *manifest, *human)
	if err != nil {
		fatal(err.Error())
	}
	data, err := json.Marshal(result.GuardianHarness.Summary)
	if err != nil {
		fatal(err.Error())
	}
	fmt.Println(string(data))
}

func run(args []string) {
	set := flag.NewFlagSet("run", flag.ExitOnError)
	source := set.String("source", "", "path to declared .gooo source")
	contract := set.String("contract", "", "fixed denominator contract")
	out := set.String("out", "", "caller-owned temporary output directory")
	externalURI := set.String("external-release-uri", "", "optional immutable external release URI")
	externalDigest := set.String("external-release-digest", "", "optional immutable external release sha256 digest")
	set.Parse(args)
	if *source == "" || *contract == "" || *out == "" {
		fatal("run requires --source, --contract, and --out")
	}
	var external *migration.ExternalRelease
	if *externalURI != "" || *externalDigest != "" {
		if *externalURI == "" || *externalDigest == "" {
			fatal("external release URI and digest must be supplied together")
		}
		external = &migration.ExternalRelease{URI: *externalURI, Digest: *externalDigest}
	}
	report, err := migration.Run(*source, *contract, *out, external)
	if err != nil {
		fatal(err.Error())
	}
	data, err := json.Marshal(report.Summary)
	if err != nil {
		fatal(err.Error())
	}
	fmt.Println(string(data))
}

func ciSummary(args []string) {
	set := flag.NewFlagSet("ci-summary", flag.ExitOnError)
	report := set.String("report", "", "report.json path")
	build := set.String("build-metrics", "", "measured build metrics")
	test := set.String("test-metrics", "", "measured test metrics")
	conformance := set.String("conformance-metrics", "", "measured conformance metrics")
	out := set.String("out", "", "CI summary output path")
	set.Parse(args)
	if *report == "" || *build == "" || *test == "" || *conformance == "" || *out == "" {
		fatal("ci-summary requires --report, --build-metrics, --test-metrics, --conformance-metrics, and --out")
	}
	summary, err := migration.BuildCISummary(*report, *build, *test, *conformance, *out)
	if err != nil {
		fatal(err.Error())
	}
	data, err := json.Marshal(summary)
	if err != nil {
		fatal(err.Error())
	}
	fmt.Println(string(data))
}

func fatal(message string) {
	fmt.Fprintln(os.Stderr, message)
	os.Exit(1)
}
