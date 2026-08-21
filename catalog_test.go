package contracts

import "testing"

func TestCatalogIDConvergesOnNormalizedSourceIdentity(t *testing.T) {
	want := CatalogID("cgv", "theater", "서울/용산아이파크몰")
	if got := CatalogID(" CGV ", "theater", " 서울/용산아이파크몰 "); got != want {
		t.Fatalf("normalized catalog id = %q, want %q", got, want)
	}
	if got := CatalogID("cgv", "auditorium", "서울/용산아이파크몰"); got == want {
		t.Fatal("catalog entity kinds collided")
	}
}

func TestSeatMapVersionIDBindsAuditoriumAndLayout(t *testing.T) {
	first := SeatMapVersionID("auditorium-a", "aaaaaaaa")
	if first == SeatMapVersionID("auditorium-b", "aaaaaaaa") ||
		first == SeatMapVersionID("auditorium-a", "bbbbbbbb") {
		t.Fatal("seat map version identity is not content-addressed")
	}
}

func TestSeatIDNormalizesOnlyTheProviderSeatLabel(t *testing.T) {
	first := SeatID("auditorium-a", " h10 ")
	if first != SeatID("auditorium-a", "H10") {
		t.Fatal("SeatID() did not normalize the provider seat label")
	}
	if first == SeatID("auditorium-b", "H10") || first == SeatID("auditorium-a", "H11") {
		t.Fatal("SeatID() did not bind auditorium and seat label")
	}
}
