package contracts

import (
	"errors"
	"testing"
)

func TestProtocol(t *testing.T) {
	if ProtocolHeaderValue() != "3" {
		t.Fatalf("protocol header = %q", ProtocolHeaderValue())
	}
	if err := RequireProtocol(ProtocolVersion); err != nil {
		t.Fatalf("current protocol: %v", err)
	}
	if err := RequireProtocol(1); err == nil {
		t.Fatal("legacy protocol unexpectedly accepted")
	}
	if err := RequireProtocol(ProtocolVersion + 1); err == nil {
		t.Fatal("future protocol unexpectedly accepted")
	}
	if ProbeBootstrapAlgorithm != "ES256" || ProbeBootstrapTokenType != "Cineko-Probe-Bootstrap" || // #nosec G101 -- public JOSE type marker.
		ProbeBootstrapIssuer == "" || ProbeBootstrapAudience == "" || ProbeBootstrapMaxClockSkew <= 0 ||
		ProbeBootstrapMaxConcurrent != 1 {
		t.Fatal("invalid Probe bootstrap contract constants")
	}
}

func TestSupportedCapabilities(t *testing.T) {
	for _, value := range []string{
		CapabilityCGVScheduleCapture,
		CapabilityCGVCatalogCapture,
		CapabilityCGVSeatMapCapture,
	} {
		if !IsSupportedCapability(value) {
			t.Fatalf("canonical CGV capability %q is not supported", value)
		}
	}
	for _, value := range []string{"", " cgv.schedule.capture.v2 ", "cgv.schedule.capture.v1", "arbitrary.v1"} {
		if IsSupportedCapability(value) {
			t.Fatalf("unsupported capability %q accepted", value)
		}
	}
}

func TestRequireEgressPolicy(t *testing.T) {
	t.Parallel()
	if err := RequireEgressPolicy(EgressPolicyScanDefault); err != nil {
		t.Fatalf("RequireEgressPolicy(scan_default) error = %v", err)
	}
	for _, policy := range []EgressPolicyID{"", "unknown"} {
		if err := RequireEgressPolicy(policy); !errors.Is(err, ErrUnsupportedEgressPolicy) {
			t.Fatalf("RequireEgressPolicy(%q) error = %v", policy, err)
		}
	}
}
