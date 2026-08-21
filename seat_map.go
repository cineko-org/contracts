package contracts

import (
	"errors"
	"fmt"
)

// SeatMapLayout is the canonical static layout stored by Central. Live sale
// status is deliberately excluded because it belongs to a showtime observation.
type SeatMapLayout struct {
	Seats  []SeatMapSeat `json:"seats"`
	Zones  []LayoutZone  `json:"zones"`
	Blocks []LayoutBlock `json:"blocks"`
}

var ErrUnsupportedSeatMapResolution = errors.New("unsupported seat-map resolution")

// SeatMapResolutionStatus is the versioned outcome of a Client seat-map
// request. It describes Central's result without exposing collection details.
type SeatMapResolutionStatus string

const (
	SeatMapResolutionReady        = SeatMapResolutionStatus("ready")
	SeatMapResolutionWaiting      = SeatMapResolutionStatus("waiting")
	SeatMapResolutionUnverifiable = SeatMapResolutionStatus("unverifiable")
)

// SeatMapResolution is Central's complete response to a Client seat-map
// request. Clients do not participate in cache or capture decisions.
type SeatMapResolution struct {
	Status  SeatMapResolutionStatus `json:"status"`
	SeatMap *SeatMapVersion         `json:"seatMap,omitempty"`
}

// Validate rejects unknown states and state/payload combinations at service
// boundaries so a Client never has to infer Central's collection outcome.
func (resolution SeatMapResolution) Validate() error {
	switch resolution.Status {
	case SeatMapResolutionReady:
		if resolution.SeatMap == nil {
			return fmt.Errorf("%w: ready response has no seat map", ErrUnsupportedSeatMapResolution)
		}
	case SeatMapResolutionWaiting, SeatMapResolutionUnverifiable:
		if resolution.SeatMap != nil {
			return fmt.Errorf("%w: %s response includes a seat map", ErrUnsupportedSeatMapResolution, resolution.Status)
		}
	default:
		return fmt.Errorf("%w: %q", ErrUnsupportedSeatMapResolution, resolution.Status)
	}
	return nil
}

// SeatMapSeat describes one physical seat and its normalized auditorium position.
type SeatMapSeat struct {
	ID                 string   `json:"id"`
	AuditoriumID       string   `json:"auditoriumId"`
	Label              string   `json:"label"`
	Row                string   `json:"row"`
	Number             int      `json:"number"`
	X                  float64  `json:"x"`
	Y                  float64  `json:"y"`
	Type               string   `json:"type"`
	ZoneName           string   `json:"zoneName"`
	ZoneKind           string   `json:"zoneKind"`
	SaleFormCode       string   `json:"saleFormCode"`
	SaleFormName       string   `json:"saleFormName"`
	LeftAisle          bool     `json:"leftAisle"`
	RightAisle         bool     `json:"rightAisle"`
	Features           []string `json:"features"`
	SourceLabel        string   `json:"sourceLabel"`
	SourceSeatKindCode string   `json:"sourceSeatKindCode"`
	SourceSeatKindName string   `json:"sourceSeatKindName"`
	SourceClasses      []string `json:"sourceClasses,omitempty"`
}

// LayoutZone preserves a provider-defined pricing or seating zone.
type LayoutZone struct {
	Code     string  `json:"code"`
	Name     string  `json:"name"`
	KindCode string  `json:"kindCode"`
	KindName string  `json:"kindName"`
	MinX     float64 `json:"minX"`
	MaxX     float64 `json:"maxX"`
	MinY     float64 `json:"minY"`
	MaxY     float64 `json:"maxY"`
	Capacity int     `json:"capacity"`
}

// LayoutBlock preserves a provider-defined physical seating block.
type LayoutBlock struct {
	Code     string  `json:"code"`
	Name     string  `json:"name"`
	KindCode string  `json:"kindCode"`
	KindName string  `json:"kindName"`
	MinX     float64 `json:"minX"`
	MaxX     float64 `json:"maxX"`
	MinY     float64 `json:"minY"`
	MaxY     float64 `json:"maxY"`
}
