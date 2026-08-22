package contracts_test

import (
	"strings"
	"testing"
	"time"

	"buf.build/go/protovalidate"
	catalogpb "github.com/cineko-org/contracts/gen/go/cineko/catalog"
	clientpb "github.com/cineko-org/contracts/gen/go/cineko/client"
	commonpb "github.com/cineko-org/contracts/gen/go/cineko/common"
	executionpb "github.com/cineko-org/contracts/gen/go/cineko/execution"
	observationpb "github.com/cineko-org/contracts/gen/go/cineko/observation"
	probepb "github.com/cineko-org/contracts/gen/go/cineko/probe"
	"github.com/cineko-org/contracts/gen/go/cineko/seatmap"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
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

func TestWebUIContractValidation(t *testing.T) {
	t.Parallel()

	validator, err := protovalidate.New()
	if err != nil {
		t.Fatalf("create validator: %v", err)
	}

	invalidRequests := []struct {
		name    string
		message proto.Message
	}{
		{name: "credentials", message: &clientpb.AccountCredentials{}},
		{name: "task state", message: &clientpb.WebUITaskState{}},
		{name: "account state", message: &clientpb.WebUIAccountState{}},
		{name: "action status", message: &clientpb.WebUIActionStatus{}},
		{name: "resource mutation", message: &clientpb.WebUIResourceMutation{}},
		{name: "resource deletion", message: &clientpb.WebUIResourceDeletion{}},
		{name: "monitor retry", message: &clientpb.WebUIMonitorRetryRequest{}},
		{name: "reservation cancellation", message: &clientpb.WebUIReservationCancellationRequest{}},
		{name: "event user", message: &clientpb.WebUIAppEventUserRequest{}},
		{name: "seat-map request", message: &clientpb.SeatMapRequest{}},
		{name: "auditorium request", message: &clientpb.AuditoriumRequest{}},
	}
	for _, test := range invalidRequests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			if err := validator.Validate(test.message); err == nil {
				t.Fatal("empty WebUI message passed contract validation")
			}
		})
	}

	userID, taskID := "user", "task"
	updatedAt := timestamppb.New(time.Unix(1, 0).UTC())
	validTask := clientpb.WebUITaskState_builder{
		Id:        &taskID,
		Running:   clientpb.WebUITaskRunning_builder{}.Build(),
		UpdatedAt: updatedAt,
	}.Build()
	if err := validator.Validate(validTask); err != nil {
		t.Fatalf("valid task state failed contract validation: %v", err)
	}

	validState := clientpb.WebUIState_builder{
		UserId:  &userID,
		Catalog: catalogpb.CatalogIndex_builder{}.Build(),
	}.Build()
	if err := validator.Validate(validState); err != nil {
		t.Fatalf("valid WebUI state failed contract validation: %v", err)
	}

	validAction := clientpb.WebUIActionStatus_builder{
		Completed: clientpb.WebUIActionCompleted_builder{}.Build(),
	}.Build()
	if err := validator.Validate(validAction); err != nil {
		t.Fatalf("valid WebUI action failed contract validation: %v", err)
	}
}

func TestAvailabilitySnapshotRequiresExactIdentity(t *testing.T) {
	t.Parallel()

	validator, err := protovalidate.New()
	if err != nil {
		t.Fatalf("create validator: %v", err)
	}
	if err := validator.Validate(&seatmap.AvailabilitySnapshot{}); err == nil {
		t.Fatal("empty availability snapshot passed contract validation")
	}

	valid := seatmap.AvailabilitySnapshot_builder{
		ShowtimeId:   protoString("showtime-1"),
		AuditoriumId: protoString("auditorium-1"),
		LayoutHash:   protoString(strings.Repeat("a", 64)),
		AvailableSeats: []*seatmap.AvailableSeat{
			seatmap.AvailableSeat_builder{SeatId: protoString("A-1")}.Build(),
		},
		ObservedAt: timestamppb.New(time.Unix(1, 0).UTC()),
	}.Build()
	if err := validator.Validate(valid); err != nil {
		t.Fatalf("valid availability snapshot failed contract validation: %v", err)
	}
}

func protoString(value string) *string { return &value }

func protoInt32(value int32) *int32 { return &value }
