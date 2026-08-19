package contracts

import (
	"bytes"
	"embed"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

//go:embed testdata/wire/*.json
var wireFixtures embed.FS

func TestRepresentativeWireNames(t *testing.T) {
	value := ClaimAssignmentResponse{
		AssignmentID:   "assignment",
		LeaseToken:     "lease",
		LeaseExpiresAt: time.Date(2026, time.August, 12, 0, 0, 0, 0, time.UTC),
		Task: AssignmentTask{
			Kind: "schedule",
			Theater: Theater{
				ID: "theater-001", ProviderID: ProviderCGV,
				SourceKey: "seoul/yongsan", Region: "Seoul", Name: "Yongsan",
			},
			TargetDates: []string{"2026-08-20"}, Locale: "ko-KR", TimeZone: "Asia/Seoul",
		},
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"assignmentId", "leaseExpiresAt", "targetDates", "timeZone"} {
		if !strings.Contains(string(encoded), `"`+field+`"`) {
			t.Fatalf("wire field %q missing from %s", field, encoded)
		}
	}
}

func TestCatalogWireNames(t *testing.T) {
	encoded, err := json.Marshal(CatalogSnapshot{
		Provider: Provider{ID: ProviderCGV, Name: "CGV"},
		Theaters: []Theater{{
			ID:         CatalogID(ProviderCGV, "theater", "서울/용산아이파크몰"),
			ProviderID: ProviderCGV, SourceKey: "서울/용산아이파크몰",
			Region: "서울", Name: "용산아이파크몰",
		}},
		ObservedAt: time.Date(2026, time.August, 14, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"provider", "theaters", "sourceKey", "observedAt"} {
		if !strings.Contains(string(encoded), `"`+field+`"`) {
			t.Fatalf("catalog wire field %q missing from %s", field, encoded)
		}
	}
}

func TestCatalogAssignmentResultWireShape(t *testing.T) {
	result := AssignmentResult{
		RunID: "run", Status: "completed",
		Catalog: &CatalogSnapshot{
			Provider:   Provider{ID: ProviderCGV, Name: "CGV"},
			ObservedAt: time.Date(2026, time.August, 18, 0, 0, 0, 0, time.UTC),
		},
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"catalog"`) {
		t.Fatalf("catalog result missing from %s", encoded)
	}
	result.Catalog = nil
	encoded, err = json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), `"catalog"`) {
		t.Fatalf("empty catalog result encoded: %s", encoded)
	}
}

func TestSeatMapAssignmentWireShape(t *testing.T) {
	auditorium := Auditorium{ID: "auditorium_01", TheaterID: "theater_01", SourceKey: "theater/1관", Name: "1관"}
	showtime := Showtime{ID: "showtime_01", TheaterID: "theater_01", Auditorium: auditorium}
	value := struct {
		Task   AssignmentTask   `json:"task"`
		Result AssignmentResult `json:"result"`
	}{
		Task: AssignmentTask{
			Kind: CapabilityCGVSeatMapCapture, Auditorium: &auditorium, Showtime: &showtime,
		},
		Result: AssignmentResult{
			RunID: "run_01", Status: "completed",
			SeatMap: &SeatMapVersion{
				ID: "seat-map_01", AuditoriumID: auditorium.ID, LayoutHash: "hash", Capacity: 1,
			},
		},
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{`"auditorium":`, `"showtime":`, `"seatMap":`} {
		if !strings.Contains(string(encoded), field) {
			t.Fatalf("seat-map assignment wire shape %s missing %s", encoded, field)
		}
	}
}

func TestProbeHeartbeatAdvertisesCurrentCapabilities(t *testing.T) {
	encoded, err := json.Marshal(ProbeHeartbeatRequest{
		AvailableCapabilities: []string{CapabilityCGVScheduleCapture, CapabilityCGVSeatMapCapture},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"availableCapabilities":["cgv.schedule.capture.v2","cgv.seat-map.capture.v1"]`) {
		t.Fatalf("current Probe capabilities missing from %s", encoded)
	}
}

func TestSecretFieldsDoNotLeakThroughOmitEmpty(t *testing.T) {
	encoded, err := json.Marshal(AuthExchangeResponse{User: ClientUser{ID: "user"}}) // #nosec G117 -- no credential value is populated.
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "launch") {
		t.Fatalf("empty launch context encoded: %s", encoded)
	}
}

func TestLauncherReleaseWireNames(t *testing.T) {
	encoded, err := json.Marshal(LauncherRelease{
		Channel: "stable", Platform: "darwin", Arch: "arm64", Version: "1.2.0",
		Protocol: ProtocolVersion,
		Launcher: ReleaseArtifact{
			URL:  "https://cineko.example/v1/releases/artifacts/1.2.0/launcher.zip",
			Size: 42, SHA256: strings.Repeat("a", 64),
			Executable: "Cineko Launcher.app/Contents/MacOS/Cineko Launcher",
		},
		PublishedAt: time.Date(2026, time.August, 12, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"platform", "arch", "version", "launcher", "publishedAt"} {
		if !strings.Contains(string(encoded), `"`+field+`"`) {
			t.Fatalf("wire field %q missing from %s", field, encoded)
		}
	}
}

func TestRuntimeReleaseGoldenWire(t *testing.T) {
	assertGoldenJSON(t, "v2-runtime-release.json", RuntimeRelease{
		Client: ClientRelease{
			Channel: "stable", Platform: "darwin", Arch: "arm64", Version: "2.3.4",
			MinimumLauncherVersion: "2.0.0", MinimumBrowserRevision: "1228", PlaywrightVersion: "1.61.1",
			Protocol: ProtocolVersion,
			Artifact: ReleaseArtifact{
				URL: "https://cdn.example/client/v2.3.4/darwin-arm64/client.tar.gz", Size: 101,
				SHA256: strings.Repeat("a", 64), Executable: "Cineko Client.app/Contents/MacOS/Cineko Client",
			},
			ProbeBootstrapPublicKeys: map[string]string{"2026-08": "public-key"},
			PublishedAt:              time.Date(2026, time.August, 12, 1, 2, 3, 0, time.UTC),
		},
		Browser: BrowserRelease{
			Channel: "stable", Platform: "darwin", Arch: "arm64", Revision: "1228",
			CompatiblePlaywrightVersions: []string{"1.61.1"},
			Artifact: ReleaseArtifact{
				URL: "https://cdn.example/browser/r1228/darwin-arm64/browser.tar.gz", Size: 202,
				SHA256: strings.Repeat("b", 64), Executable: "chromium/Chromium.app/Contents/MacOS/Chromium",
			},
			PublishedAt: time.Date(2026, time.August, 12, 1, 2, 4, 0, time.UTC),
		},
		Playwright: PlaywrightRelease{
			Channel: "stable", Platform: "darwin", Arch: "arm64", Version: "1.61.1",
			Artifact: ReleaseArtifact{
				URL: "https://cdn.example/playwright/v1.61.1/darwin-arm64/playwright.tar.gz", Size: 303,
				SHA256: strings.Repeat("c", 64), Executable: "playwright/node",
			},
			PublishedAt: time.Date(2026, time.August, 12, 1, 2, 5, 0, time.UTC),
		},
	})
}

func TestLaunchGenerationWireNames(t *testing.T) {
	encoded, err := json.Marshal(LaunchTicketRequest{ReleaseGeneration: 17})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"releaseGeneration":17`) {
		t.Fatalf("release generation missing from launch ticket: %s", encoded)
	}

	encoded, err = json.Marshal(ClientLaunchContext{ReleaseGeneration: 17})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"releaseGeneration":17`) {
		t.Fatalf("release generation missing from launch context: %s", encoded)
	}
}

func TestProbeReleaseWireNames(t *testing.T) {
	encoded, err := json.Marshal(ProbeRelease{
		Version: "1.2.3", BrowserRevision: "1228", Image: "registry.example.com/cineko/probe:v1.2.3",
		ImageDigest: "sha256:" + strings.Repeat("a", 64),
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{`"version"`, `"browserRevision"`, `"image"`, `"imageDigest"`} {
		if !strings.Contains(string(encoded), name) {
			t.Fatalf("Probe release field %s missing from %s", name, encoded)
		}
	}
}

func TestReleaseSetGoldenWire(t *testing.T) {
	release := ClientRelease{
		Channel: "stable", Platform: "linux", Arch: "amd64", Version: "2.3.4",
		MinimumLauncherVersion: "2.0.0", MinimumBrowserRevision: "1228", PlaywrightVersion: "1.61.1",
		Protocol: ProtocolVersion,
		Artifact: ReleaseArtifact{
			URL: "https://cdn.example/client/v2.3.4/linux-amd64/client.tar.gz", Size: 404,
			SHA256: strings.Repeat("d", 64), Executable: "cineko-client",
		},
		ProbeBootstrapPublicKeys: map[string]string{"2026-08": "public-key"},
		PublishedAt:              time.Date(2026, time.August, 12, 2, 3, 4, 0, time.UTC),
	}
	assertGoldenJSON(t, "v2-client-release-set.json", ReleaseEnvelope[ReleaseSet[ClientRelease]]{
		SchemaVersion: ReleasePayloadSchemaVersion,
		Payload:       ReleaseSet[ClientRelease]{Releases: []ClientRelease{release}},
	})
	assertGoldenJSON(t, "v2-client-release-payload.json", ReleaseEnvelope[ClientRelease]{
		SchemaVersion: ReleasePayloadSchemaVersion,
		Payload:       release,
	})
}

func TestProbeBootstrapClaimsGoldenWire(t *testing.T) {
	assertGoldenJSON(t, "v2-probe-bootstrap-claims.json", ProbeBootstrapClaims{
		Issuer: ProbeBootstrapIssuer, Audience: ProbeBootstrapAudience, UserID: "user-1", TicketID: "ticket-1",
		IssuedAt: 1_786_493_723, NotBefore: 1_786_493_708, ExpiresAt: 1_786_493_783,
		InstallationID: "installation-1", DeviceID: "device-1", Kind: "client",
		Capabilities: []string{CapabilityCGVScheduleCapture}, MaxConcurrency: 1,
		Runtime: Runtime{
			Version: "2.3.4", Protocol: ProtocolVersion, BrowserRevision: "1228",
			Platform: "darwin", Arch: "arm64",
		},
	})
}

func TestClientLaunchEnvelopeGoldenWire(t *testing.T) {
	assertGoldenJSON(t, "v2-client-launch-envelope.json", ClientLaunchEnvelope{
		LaunchTicket: "one-time-launch-ticket",
		ClientLaunchContext: ClientLaunchContext{
			InstallationID: "installation-1", DeviceID: "device-1", ReleaseGeneration: 17,
			ClientVersion: "2.3.4", ArtifactSHA256: strings.Repeat("a", 64), Protocol: ProtocolVersion,
			BrowserRevision: "1228", BrowserArtifactSHA256: strings.Repeat("b", 64),
			PlaywrightVersion: "1.61.1", PlaywrightArtifactSHA256: strings.Repeat("c", 64),
		},
	})
}

func assertGoldenJSON[Value any](t *testing.T, name string, value Value) {
	t.Helper()
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	encoded = append(encoded, '\n')
	expected, err := wireFixtures.ReadFile("testdata/wire/" + name)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(encoded, expected) {
		t.Fatalf("wire fixture %s changed\nwant:\n%s\ngot:\n%s", name, expected, encoded)
	}
	var decoded Value
	if err := json.Unmarshal(expected, &decoded); err != nil {
		t.Fatalf("decode wire fixture %s: %v", name, err)
	}
	roundTrip, err := json.MarshalIndent(decoded, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	roundTrip = append(roundTrip, '\n')
	if !bytes.Equal(roundTrip, expected) {
		t.Fatalf("wire fixture %s does not round-trip\nwant:\n%s\ngot:\n%s", name, expected, roundTrip)
	}
}
