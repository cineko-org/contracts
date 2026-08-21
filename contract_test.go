package contracts_test

import (
	"testing"

	"buf.build/go/protovalidate"
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

func protoString(value string) *string { return &value }
