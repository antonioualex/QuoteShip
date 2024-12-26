package domain

import (
	"errors"
	"time"
)

var (
	ErrInvalidTopValue   = errors.New("invalid top value provided")
	ErrNoExpectedRates   = errors.New("no expected rates available")
	ErrNilShipmentUnit   = errors.New("nil shipment unit provided")
	ErrInvalidOriginPort = errors.New("invalid origin port provided")
	ErrInvalidPrice      = errors.New("invalid price provided")
	ErrInvalidDate       = errors.New("invalid date provided")
	ErrInvalidCompany    = errors.New("invalid company provided")
	ErrNoValidRates      = errors.New("no valid rates calculated")
	ErrNilRepository     = errors.New("nil repository provided")
)

// OriginShipments represents a list of ShipmentQuote for a specific Origin.
type OriginShipments struct {
	Origin string          // Origin is the located port where the shipment starts (e.g., "CNSGH").
	Quotes []ShipmentQuote // Quotes is a list of ShipmentQuote for the specified origin.
}

// ShipmentUnit represents a single shipment details in a form that is origin based.
type ShipmentUnit struct {
	Origin        string // Origin is the located port where the shipment starts (e.g., "CNSGH").
	ShipmentQuote        // ShipmentQuote contains the details of a shipment quote (company, price, date).
}

// ShipmentQuote holds the details of a single shipping quote.
type ShipmentQuote struct {
	Company int       // Company is the name of the company that provided the quote.
	Price   int       // Price is the cost of the shipment.
	Date    time.Time // Date is the date when the shipment will start.
}

// ShipmentService defines the operations related to managing and retrieving shipment data.
type ShipmentService interface {
	GetLatestExpectedRates(top int) (map[string]int, error) // GetLatestExpectedRates retrieves the expected rates for the top lowest-priced offers, grouped by origin. The top parameter specifies the number of offers to consider.
	SubmitShipment(shipment *ShipmentUnit) error            // SubmitShipment submits a new ShipmentUnit offer to the system.
	IncrementShipmentUnitsCount()                           // IncrementShipmentUnitsCount increments the internal counter for received shipment units.
}

// ShipmentRepository defines the data layer operations for managing shipment units.
type ShipmentRepository interface {
	AddOrUpdate(shipment ShipmentUnit) error             // AddOrUpdate adds or updates a new ShipmentUnit offer to the repository, if it is outdated or already exists then it will not be updated.
	GetLatestSortedShipmentsByOrigin() []OriginShipments // GetLatestSortedShipmentsByOrigin retrieves the latest batched shipment units grouped by origin port and sorted by price.
	IncrementShipmentUnitsCount()                        // IncrementShipmentUnitsCount tracks the number of received shipment units by incrementing an internal counter.
}
