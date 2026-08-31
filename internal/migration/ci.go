package migration

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

func BuildCISummary(reportPath, buildMetricsPath, testMetricsPath, conformanceMetricsPath, outputPath string) (CISummary, error) {
	var report Report
	if err := ReadJSON(reportPath, &report); err != nil {
		return CISummary{}, err
	}
	build, err := readProcessMetrics(buildMetricsPath)
	if err != nil {
		return CISummary{}, err
	}
	test, err := readProcessMetrics(testMetricsPath)
	if err != nil {
		return CISummary{}, err
	}
	conformance, err := readProcessMetrics(conformanceMetricsPath)
	if err != nil {
		return CISummary{}, err
	}
	metrics := []MetricValue{
		{ID: "schema_versions", Value: report.SchemaVersions},
		{ID: "parent_receipt_count", Value: report.Summary.ParentReceiptCount},
		{ID: "child_receipt_count", Value: report.Summary.ChildReceiptCount},
		{ID: "accepted_count", Value: report.Summary.AcceptedCount},
		{ID: "unknown_count", Value: report.Summary.UnknownCount},
		{ID: "refuted_count", Value: report.Summary.RefutedCount},
		{ID: "immutable_parent_writes", Value: report.Summary.ImmutableParentWrites},
		{ID: "adapter_operations", Value: report.AdapterOperations},
		{ID: "artifact_digests", Value: report.ArtifactDigests},
		{ID: "build_wall_ms", Value: build.WallMS},
		{ID: "build_peak_rss_kib", Value: build.PeakRSSKiB},
		{ID: "test_wall_ms", Value: test.WallMS},
		{ID: "test_peak_rss_kib", Value: test.PeakRSSKiB},
		{ID: "conformance_wall_ms", Value: conformance.WallMS},
		{ID: "conformance_peak_rss_kib", Value: conformance.PeakRSSKiB},
	}
	if err := validateMetricBindings(report.MetricBindings, metrics); err != nil {
		return CISummary{}, err
	}
	summary := CISummary{
		Schema: CISummarySchema, ReportDigest: report.ReportDigest, SchemaVersions: append([]string(nil), report.SchemaVersions...),
		ParentReceiptCount: report.Summary.ParentReceiptCount, ChildReceiptCount: report.Summary.ChildReceiptCount, AcceptedCount: report.Summary.AcceptedCount,
		UnknownCount: report.Summary.UnknownCount, RefutedCount: report.Summary.RefutedCount, ImmutableParentWrites: report.Summary.ImmutableParentWrites,
		AdapterOperations: append([]string(nil), report.AdapterOperations...), ArtifactDigests: append([]ArtifactRef(nil), report.ArtifactDigests...), MetricBindings: append([]MetricBinding(nil), report.MetricBindings...), Metrics: metrics, Improvement: report.Improvement,
	}
	if summary.ReportDigest == "" {
		return CISummary{}, fmt.Errorf("report digest is required")
	}
	if err := WriteJSON(outputPath, summary); err != nil {
		return CISummary{}, err
	}
	return summary, nil
}

type processMetrics struct {
	WallMS      int64
	PeakRSSKiB  int64
}

func readProcessMetrics(path string) (processMetrics, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return processMetrics{}, err
	}
	values := strings.Fields(string(data))
	if len(values) != 2 {
		return processMetrics{}, fmt.Errorf("metrics %s must contain wall_ms and peak_rss_kib", path)
	}
	wall, err := parseMetric(values[0], "wall_ms")
	if err != nil {
		return processMetrics{}, err
	}
	rss, err := parseMetric(values[1], "peak_rss_kib")
	if err != nil {
		return processMetrics{}, err
	}
	return processMetrics{WallMS: wall, PeakRSSKiB: rss}, nil
}

func parseMetric(value, key string) (int64, error) {
	parts := strings.SplitN(value, "=", 2)
	if len(parts) != 2 || parts[0] != key {
		return 0, fmt.Errorf("expected %s metric, got %q", key, value)
	}
	n, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || n < 0 {
		return 0, fmt.Errorf("invalid %s metric %q", key, value)
	}
	return n, nil
}

func validateMetricBindings(bindings []MetricBinding, values []MetricValue) error {
	bound := make(map[string]MetricBinding, len(bindings))
	for _, binding := range bindings {
		bound[binding.ID] = binding
	}
	for _, value := range values {
		binding, ok := bound[value.ID]
		if !ok || binding.MetaActivity == "" || binding.SourcePath == "" || binding.IRPath == "" || binding.GeneratedArtifact == "" || binding.Evaluator == "" {
			return fmt.Errorf("metric %s has no complete source/IR/artifact/evaluator binding", value.ID)
		}
	}
	if len(values) != len(bindings) {
		return fmt.Errorf("CI metric values and bindings are not one-to-one")
	}
	return nil
}
