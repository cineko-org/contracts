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
	observationpb "github.com/cineko-org/contracts/v3/gen/go/cineko/observation"
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

func TestMoviePosterAcceptsObservedLargeCGVResponse(t *testing.T) {
	t.Parallel()

	validator, err := protovalidate.New()
	if err != nil {
		t.Fatalf("create validator: %v", err)
	}
	poster := catalogpb.MoviePoster_builder{
		MovieId:     protoString("movie-1"),
		MediaType:   protoString("image/jpeg"),
		Data:        make([]byte, 12_132_418),
		ContentHash: protoString(strings.Repeat("a", 64)),
	}.Build()
	if err := validator.Validate(poster); err != nil {
		t.Fatalf("observed 12.1 MB CGV poster failed validation: %v", err)
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

func TestSeatMapTaskRequiresAuditorium(t *testing.T) {
	t.Parallel()

	validator, err := protovalidate.New()
	if err != nil {
		t.Fatalf("create validator: %v", err)
	}
	task := validSeatMapTask()
	if err := validator.Validate(task); err != nil {
		t.Fatalf("valid auditorium-scoped seat-map task failed validation: %v", err)
	}
	task.SetAuditorium(nil)
	if err := validator.Validate(task); err == nil {
		t.Fatal("seat-map task without an auditorium passed validation")
	}
}

func TestProviderIdentityValidatesOpaqueTheaterSiteNumber(t *testing.T) {
	t.Parallel()

	validator, err := protovalidate.New()
	if err != nil {
		t.Fatalf("create validator: %v", err)
	}
	for _, siteNo := range []string{"0056", "P001", "P004", "P013"} {
		identity := catalogpb.TheaterIdentity_builder{
			Cgv: catalogpb.CgvTheaterIdentity_builder{SiteNo: protoString(siteNo)}.Build(),
		}.Build()
		if err := validator.Validate(identity); err != nil {
			t.Fatalf("opaque theater site number %q failed validation: %v", siteNo, err)
		}
	}
	for _, siteNo := range []string{"", strings.Repeat("x", 65)} {
		identity := catalogpb.TheaterIdentity_builder{
			Cgv: catalogpb.CgvTheaterIdentity_builder{SiteNo: protoString(siteNo)}.Build(),
		}.Build()
		if err := validator.Validate(identity); err == nil {
			t.Fatalf("invalid theater site number of length %d passed validation", len(siteNo))
		}
	}

	valid := catalogpb.ShowtimeIdentity_builder{
		Cgv: catalogpb.CgvShowtimeIdentity_builder{
			SiteNo:       protoString("P001"),
			ScheduleDate: commonpb.LocalDate_builder{Year: protoInt32(2026), Month: protoInt32(8), Day: protoInt32(22)}.Build(),
			ScreenNo:     protoString("0007"),
			Sequence:     protoString("0003"),
		}.Build(),
	}.Build()
	if err := validator.Validate(valid); err != nil {
		t.Fatalf("valid typed showtime identity failed validation: %v", err)
	}

	invalidScreen := catalogpb.AuditoriumIdentity_builder{
		Cgv: catalogpb.CgvAuditoriumIdentity_builder{
			SiteNo:   protoString("P004"),
			ScreenNo: protoString("IMAX관"),
		}.Build(),
	}.Build()
	if err := validator.Validate(invalidScreen); err == nil {
		t.Fatal("display text passed provider screen identity validation")
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
				TargetDateUnavailable: collectionpb.TargetDateUnavailable_builder{}.Build(),
			}.Build(),
		}.Build(),
	}.Build()
	if err := validator.Validate(missingRunMetadata); err == nil {
		t.Fatal("assignment result without run metadata passed validation")
	}
	valid := observationpb.Completed_builder{
		Schedule: observationpb.ScheduleCaptures_builder{
			Captures: []*observationpb.Capture{observationpb.Capture_builder{}.Build()},
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
				TargetDateUnavailable: collectionpb.TargetDateUnavailable_builder{}.Build(),
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
				TargetDateUnavailable: collectionpb.TargetDateUnavailable_builder{}.Build(),
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
		t.Fatal("collecting resolution without a claim ID and expiry passed validation")
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

func validSeatMapTask() *observationpb.SeatMapTask {
	return observationpb.SeatMapTask_builder{
		Theater:    validSeatMapTaskTheater(),
		Auditorium: validSeatMapTaskAuditorium(),
		Locale:     protoString("ko-KR"),
		TimeZone:   protoString("Asia/Seoul"),
	}.Build()
}

func validSeatMapTaskTheater() *catalogpb.Theater {
	return catalogpb.Theater_builder{
		Id:         protoString("theater-1"),
		ProviderId: protoString("cgv"),
		Identity: catalogpb.TheaterIdentity_builder{
			Cgv: catalogpb.CgvTheaterIdentity_builder{SiteNo: protoString("0056")}.Build(),
		}.Build(),
		Region: protoString("서울"),
		Name:   protoString("용산아이파크몰"),
	}.Build()
}

func validSeatMapTaskAuditorium() *catalogpb.Auditorium {
	return catalogpb.Auditorium_builder{
		Id:        protoString("auditorium-1"),
		TheaterId: protoString("theater-1"),
		Identity: catalogpb.AuditoriumIdentity_builder{
			Cgv: catalogpb.CgvAuditoriumIdentity_builder{
				SiteNo: protoString("0056"), ScreenNo: protoString("0007"),
			}.Build(),
		}.Build(),
		Name:     protoString("IMAX관"),
		Capacity: protoInt32(624),
	}.Build()
}

func protoString(value string) *string { return &value }

func protoInt32(value int32) *int32 { return &value }

func validLiveSeatObservation() *seatmap.LiveSeatObservation {
	return seatmap.LiveSeatObservation_builder{
		Layout: validSeatMapSnapshot("auditorium-1", strings.Repeat("a", 64)),
		Availability: seatmap.AvailabilitySnapshot_builder{
			ShowtimeId:   protoString("showtime-1"),
			AuditoriumId: protoString("auditorium-1"),
			LayoutHash:   protoString(strings.Repeat("a", 64)),
			ObservedAt:   timestamppb.New(time.Unix(1, 0).UTC()),
		}.Build(),
	}.Build()
}

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
