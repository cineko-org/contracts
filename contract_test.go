package contracts_test

import (
	"strings"
	"testing"
	"time"

	"buf.build/go/protovalidate"
	catalogpb "github.com/cineko-org/contracts/v3/gen/go/cineko/catalog"
	clientpb "github.com/cineko-org/contracts/v3/gen/go/cineko/client"
	collectionpb "github.com/cineko-org/contracts/v3/gen/go/cineko/collection"
	commonpb "github.com/cineko-org/contracts/v3/gen/go/cineko/common"
	executionpb "github.com/cineko-org/contracts/v3/gen/go/cineko/execution"
	observationpb "github.com/cineko-org/contracts/v3/gen/go/cineko/observation"
	probepb "github.com/cineko-org/contracts/v3/gen/go/cineko/probe"
	"github.com/cineko-org/contracts/v3/gen/go/cineko/seatmap"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestGlobalCatalogAssignmentValidates(t *testing.T) {
	t.Parallel()

	validator, err := protovalidate.New()
	if err != nil {
		t.Fatalf("create validator: %v", err)
	}
	assignment := observationpb.AssignmentTask_builder{
		Egress: commonpb.EgressPolicy_builder{
			ManagedScan: commonpb.ManagedScanEgress_builder{}.Build(),
		}.Build(),
		Catalog: observationpb.CatalogTask_builder{
			ProviderId: protoString("cgv"),
			Locale:     protoString("ko-KR"),
			TimeZone:   protoString("Asia/Seoul"),
		}.Build(),
	}.Build()
	if err := validator.Validate(assignment); err != nil {
		t.Fatalf("global catalog assignment failed validation: %v", err)
	}

	assignment.GetCatalog().ClearProviderId()
	if err := validator.Validate(assignment); err == nil {
		t.Fatal("global catalog assignment without provider ID passed validation")
	}
}

func TestCatalogTaskRejectsStaleJSONFields(t *testing.T) {
	t.Parallel()

	for _, field := range []struct {
		name  string
		value string
	}{
		{name: "theater", value: `{"identity":{"cgv":{"siteNo":"0056"}}}`},
		{name: "targetDates", value: `[{"year":2026,"month":8,"day":23}]`},
	} {
		field := field
		t.Run(field.name, func(t *testing.T) {
			var task observationpb.CatalogTask
			payload := `{"providerId":"cgv","locale":"ko-KR","timeZone":"Asia/Seoul","` +
				field.name + `":` + field.value + `}`
			err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal([]byte(payload), &task)
			if err == nil {
				t.Fatalf("stale %s JSON passed latest CatalogTask decoding", field.name)
			}
			if !strings.Contains(err.Error(), field.name) {
				t.Fatalf("stale CatalogTask failed for an unexpected reason: %v", err)
			}
		})
	}
}

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
		Snapshot: validSeatMapSnapshot("auditorium-1", strings.Repeat("a", 64)),
		State: collectionpb.State_builder{
			Idle: collectionpb.Idle_builder{}.Build(),
		}.Build(),
	}.Build()
	if err := validator.Validate(valid); err != nil {
		t.Fatalf("valid seat-map resolution failed contract validation: %v", err)
	}

	if err := validator.Validate(seatmap.Resolution_builder{
		State: collectionpb.State_builder{
			Idle: collectionpb.Idle_builder{}.Build(),
		}.Build(),
	}.Build()); err == nil {
		t.Fatal("idle seat-map resolution without a cached snapshot passed validation")
	}
}

func TestProviderIdentityRejectsDisplayText(t *testing.T) {
	t.Parallel()

	validator, err := protovalidate.New()
	if err != nil {
		t.Fatalf("create validator: %v", err)
	}
	tests := []struct {
		name     string
		identity proto.Message
	}{
		{
			name: "theater display text",
			identity: catalogpb.TheaterIdentity_builder{
				Cgv: catalogpb.CgvTheaterIdentity_builder{SiteNo: protoString("서울/용산아이파크몰")}.Build(),
			}.Build(),
		},
		{
			name: "auditorium display text",
			identity: catalogpb.AuditoriumIdentity_builder{
				Cgv: catalogpb.CgvAuditoriumIdentity_builder{
					SiteNo:   protoString("0056"),
					ScreenNo: protoString("IMAX관"),
				}.Build(),
			}.Build(),
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			if err := validator.Validate(test.identity); err == nil {
				t.Fatal("display text passed provider identity validation")
			}
		})
	}

	valid := catalogpb.ShowtimeIdentity_builder{
		Cgv: catalogpb.CgvShowtimeIdentity_builder{
			SiteNo:       protoString("0056"),
			ScheduleDate: commonpb.LocalDate_builder{Year: protoInt32(2026), Month: protoInt32(8), Day: protoInt32(22)}.Build(),
			ScreenNo:     protoString("0007"),
			Sequence:     protoString("0003"),
		}.Build(),
	}.Build()
	if err := validator.Validate(valid); err != nil {
		t.Fatalf("valid typed showtime identity failed validation: %v", err)
	}
}

func TestLiveSeatObservationRequiresMatchingIdentity(t *testing.T) {
	t.Parallel()

	validator, err := protovalidate.New()
	if err != nil {
		t.Fatalf("create validator: %v", err)
	}
	live := seatmap.LiveSeatObservation_builder{
		Layout: validSeatMapSnapshot("auditorium-1", strings.Repeat("a", 64)),
		Availability: seatmap.AvailabilitySnapshot_builder{
			ShowtimeId:   protoString("showtime-1"),
			AuditoriumId: protoString("auditorium-2"),
			LayoutHash:   protoString(strings.Repeat("a", 64)),
			ObservedAt:   timestamppb.New(time.Unix(1, 0).UTC()),
		}.Build(),
	}.Build()
	if err := validator.Validate(live); err == nil {
		t.Fatal("live seat observation with mismatched auditorium passed validation")
	}

	unknownSeat := seatmap.LiveSeatObservation_builder{
		Layout: validSeatMapSnapshot("auditorium-1", strings.Repeat("a", 64)),
		Availability: seatmap.AvailabilitySnapshot_builder{
			ShowtimeId:   protoString("showtime-1"),
			AuditoriumId: protoString("auditorium-1"),
			LayoutHash:   protoString(strings.Repeat("a", 64)),
			AvailableSeats: []*seatmap.AvailableSeat{
				seatmap.AvailableSeat_builder{SeatId: protoString("unknown")}.Build(),
			},
			ObservedAt: timestamppb.New(time.Unix(1, 0).UTC()),
		}.Build(),
	}.Build()
	if err := validator.Validate(unknownSeat); err == nil {
		t.Fatal("live seat observation with unknown available seat passed validation")
	}

	mismatchedHash := seatmap.LiveSeatObservation_builder{
		Layout: validSeatMapSnapshot("auditorium-1", strings.Repeat("a", 64)),
		Availability: seatmap.AvailabilitySnapshot_builder{
			ShowtimeId:   protoString("showtime-1"),
			AuditoriumId: protoString("auditorium-1"),
			LayoutHash:   protoString(strings.Repeat("b", 64)),
			ObservedAt:   timestamppb.New(time.Unix(1, 0).UTC()),
		}.Build(),
	}.Build()
	if err := validator.Validate(mismatchedHash); err == nil {
		t.Fatal("live seat observation with mismatched layout hash passed validation")
	}

	mismatchedSeatAuditorium := validSeatMapSnapshot("auditorium-1", strings.Repeat("a", 64))
	mismatchedSeatAuditorium.GetLayout().GetSeats()[0].SetAuditoriumId("auditorium-2")
	if err := validator.Validate(mismatchedSeatAuditorium); err == nil {
		t.Fatal("snapshot with a seat from another auditorium passed validation")
	}
}

func TestCompletedRequiresTypedPayload(t *testing.T) {
	t.Parallel()

	validator, err := protovalidate.New()
	if err != nil {
		t.Fatalf("create validator: %v", err)
	}
	if err := validator.Validate(&observationpb.Completed{}); err == nil {
		t.Fatal("empty assignment completion passed validation")
	}
	missingRunMetadata := observationpb.AssignmentResult_builder{
		Deferred: observationpb.Deferred_builder{
			Reason: collectionpb.DeferredReason_builder{
				NoBookableShowtime: collectionpb.NoBookableShowtime_builder{}.Build(),
			}.Build(),
		}.Build(),
	}.Build()
	if err := validator.Validate(missingRunMetadata); err == nil {
		t.Fatal("assignment result without run metadata passed validation")
	}
	valid := observationpb.Completed_builder{
		LiveSeat: seatmap.LiveSeatObservation_builder{
			Layout: validSeatMapSnapshot("auditorium-1", strings.Repeat("a", 64)),
			Availability: seatmap.AvailabilitySnapshot_builder{
				ShowtimeId:   protoString("showtime-1"),
				AuditoriumId: protoString("auditorium-1"),
				LayoutHash:   protoString(strings.Repeat("a", 64)),
				ObservedAt:   timestamppb.New(time.Unix(1, 0).UTC()),
			}.Build(),
		}.Build(),
	}.Build()
	if err := validator.Validate(valid); err != nil {
		t.Fatalf("valid typed assignment completion failed validation: %v", err)
	}

	deferred := observationpb.AssignmentResult_builder{
		RunId:      protoString("run-1"),
		StartedAt:  timestamppb.New(time.Unix(1, 0).UTC()),
		FinishedAt: timestamppb.New(time.Unix(2, 0).UTC()),
		Deferred: observationpb.Deferred_builder{
			Reason: collectionpb.DeferredReason_builder{
				NoBookableShowtime: collectionpb.NoBookableShowtime_builder{}.Build(),
			}.Build(),
		}.Build(),
	}.Build()
	if err := validator.Validate(deferred); err != nil {
		t.Fatalf("valid deferred assignment result failed validation: %v", err)
	}
	invalidInterval := observationpb.AssignmentResult_builder{
		RunId:      protoString("run-2"),
		StartedAt:  timestamppb.New(time.Unix(2, 0).UTC()),
		FinishedAt: timestamppb.New(time.Unix(1, 0).UTC()),
		Deferred: observationpb.Deferred_builder{
			Reason: collectionpb.DeferredReason_builder{
				NoBookableShowtime: collectionpb.NoBookableShowtime_builder{}.Build(),
			}.Build(),
		}.Build(),
	}.Build()
	if err := validator.Validate(invalidInterval); err == nil {
		t.Fatal("assignment result with inverted timestamps passed validation")
	}

	waiting := seatmap.Resolution_builder{
		State: collectionpb.State_builder{
			WaitingForShowtime: collectionpb.WaitingForShowtime_builder{
				Reason: collectionpb.WaitingReason_builder{
					ShowtimeNotDiscovered: collectionpb.ShowtimeNotDiscovered_builder{}.Build(),
				}.Build(),
			}.Build(),
		}.Build(),
	}.Build()
	if err := validator.Validate(waiting); err != nil {
		t.Fatalf("valid undiscovered-showtime waiting state failed validation: %v", err)
	}

	queued := seatmap.Resolution_builder{
		State: collectionpb.State_builder{
			Queued: collectionpb.Queued_builder{
				QueuedAt: timestamppb.New(time.Unix(1, 0).UTC()),
				Trigger: collectionpb.Trigger_builder{
					ClientRequest: collectionpb.ClientRequest_builder{}.Build(),
				}.Build(),
			}.Build(),
		}.Build(),
	}.Build()
	if err := validator.Validate(queued); err != nil {
		t.Fatalf("valid queued resolution failed validation: %v", err)
	}

	collectingWithoutAssignment := seatmap.Resolution_builder{
		State: collectionpb.State_builder{
			Collecting: collectionpb.Collecting_builder{
				StartedAt: timestamppb.New(time.Unix(1, 0).UTC()),
			}.Build(),
		}.Build(),
	}.Build()
	if err := validator.Validate(collectingWithoutAssignment); err == nil {
		t.Fatal("collecting resolution without a real assignment ID passed validation")
	}

	if err := validator.Validate(observationpb.ResultReceipt_builder{
		AssignmentId: protoString("assignment-1"),
		RunId:        protoString("run-1"),
		ContentHash:  protoString(strings.Repeat("a", 64)),
		Accepted:     observationpb.Accepted_builder{}.Build(),
	}.Build()); err != nil {
		t.Fatalf("valid result receipt failed validation: %v", err)
	}
	if err := validator.Validate(observationpb.ResultReceipt_builder{
		RunId:       protoString("run-1"),
		ContentHash: protoString(strings.Repeat("a", 64)),
		Accepted:    observationpb.Accepted_builder{}.Build(),
	}.Build()); err == nil {
		t.Fatal("result receipt without assignment ID passed validation")
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

func validSeatMapSnapshot(auditoriumID, layoutHash string) *seatmap.Snapshot {
	return seatmap.Snapshot_builder{
		Id:           protoString("layout-1"),
		AuditoriumId: protoString(auditoriumID),
		LayoutHash:   protoString(layoutHash),
		Capacity:     protoInt32(1),
		Layout: seatmap.Layout_builder{
			Seats: []*seatmap.Seat{
				seatmap.Seat_builder{
					Id:           protoString("A-1"),
					AuditoriumId: protoString(auditoriumID),
				}.Build(),
			},
		}.Build(),
		ObservedAt: timestamppb.New(time.Unix(1, 0).UTC()),
	}.Build()
}
