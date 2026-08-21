package contracts_test

import (
	"testing"

	"buf.build/go/protovalidate"
	commonpb "github.com/cineko-org/contracts/gen/go/cineko/common"
	executionpb "github.com/cineko-org/contracts/gen/go/cineko/execution"
	observationpb "github.com/cineko-org/contracts/gen/go/cineko/observation"
	probepb "github.com/cineko-org/contracts/gen/go/cineko/probe"
	"github.com/cineko-org/contracts/gen/go/cineko/seatmap"
)

func TestRequiredOneofRejectsUnsetState(t *testing.T) {
	t.Parallel()

	validator, err := protovalidate.New()
	if err != nil {
		t.Fatalf("create validator: %v", err)
	}
	if err := validator.Validate(&seatmap.Resolution{}); err == nil {
		t.Fatal("unset seat-map resolution passed contract validation")
	}

	valid := seatmap.Resolution_builder{
		Unverifiable: seatmap.Unverifiable_builder{ReasonCode: protoString("no-showtime")}.Build(),
	}.Build()
	if err := validator.Validate(valid); err != nil {
		t.Fatalf("valid seat-map resolution failed contract validation: %v", err)
	}
}

func TestInboundRequestValidationRejectsMissingIdentity(t *testing.T) {
	t.Parallel()

	validator, err := protovalidate.New()
	if err != nil {
		t.Fatalf("create validator: %v", err)
	}
	register := probepb.RegisterRequest_builder{
		Kind: probepb.ProbeKind_builder{
			Container: probepb.ContainerProbe_builder{}.Build(),
		}.Build(),
		Capabilities: []*observationpb.Capability{
			observationpb.Capability_builder{
				ScheduleCapture: observationpb.ScheduleCapture_builder{}.Build(),
			}.Build(),
		},
		MaxConcurrency: protoInt32(1),
		Runtime: commonpb.Runtime_builder{
			ComponentVersion: protoString("1.0.0"),
			BrowserRevision:  protoString("123"),
			Platform:         protoString("linux"),
			Architecture:     protoString("amd64"),
		}.Build(),
	}.Build()
	if err := validator.Validate(register); err == nil {
		t.Fatal("Probe registration without installation identity passed contract validation")
	}

	heartbeat := executionpb.HeartbeatRequest_builder{
		LeaseToken: protoString("lease"),
	}.Build()
	if err := validator.Validate(heartbeat); err == nil {
		t.Fatal("execution heartbeat without command identity passed contract validation")
	}
}

func protoString(value string) *string { return &value }

func protoInt32(value int32) *int32 { return &value }
