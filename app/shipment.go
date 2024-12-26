package app

import (
	"log/slog"
	"strings"

	"cargoplot/domain"
)

// ShipmentService handles business logic for managing and retrieving shipment data.
type ShipmentService struct {
	r domain.ShipmentRepository // r is the repository that provides access to shipment data.
}

// GetLatestExpectedRates calculates the expected rates for shipments grouped by origin.
// It considers the `top` lowest-priced offers for each origin and returns the expected rates.
// Note that the fetched most recent offers are automatically updated every 1000 offer submissions.
func (s ShipmentService) GetLatestExpectedRates(top int) (map[string]int, error) {
	if top <= 0 {
		return nil, domain.ErrInvalidTopValue // Return an error if the top
	}

	// Get the latest sorted shipments by origin from the repository.
	shipmentsByOrigin := s.r.GetLatestSortedShipmentsByOrigin()

	// Return an error if no expected rates are available
	if len(shipmentsByOrigin) == 0 {
		return nil, domain.ErrNoExpectedRates
	}

	// Calculate the expected rates for each origin based on the top (lowest) origin shipments
	expectedRates := make(map[string]int)
	for _, originShipments := range shipmentsByOrigin {
		// Skip if origin shipments are empty to avoid division by zero
		if len(originShipments.Quotes) == 0 || strings.TrimSpace(originShipments.Origin) == "" {
			continue
		}

		// Use the actual length of the slice, or a maximum of 10
		unitsCount := len(originShipments.Quotes)
		if unitsCount > top {
			unitsCount = top
		}

		// Safely calculate the total price of the top origin shipments
		totalPrice := 0
		for _, originShipmentQuote := range originShipments.Quotes[:unitsCount] {
			totalPrice += originShipmentQuote.Price
		}

		// Ensure unitsCount is not zero before calculating the average
		if unitsCount > 0 {
			expectedRates[originShipments.Origin] = totalPrice / unitsCount
		}
	}

	// Return an error if no rates could be calculated
	if len(expectedRates) == 0 {
		return nil, domain.ErrNoValidRates
	}

	return expectedRates, nil
}

// SubmitShipment submits a new shipment unit to the repository.
func (s ShipmentService) SubmitShipment(shipment *domain.ShipmentUnit) error {
	if shipment == nil {
		slog.Warn("failed to submit shipment", "error", domain.ErrNilShipmentUnit)
		return domain.ErrNilShipmentUnit // Return an error if the shipment is nil.
	}

	switch {
	case strings.TrimSpace(shipment.Origin) == "":
		return domain.ErrInvalidOriginPort // Return an error if the origin port is empty.
	case shipment.Price <= 0:
		return domain.ErrInvalidPrice // Return an error if the price is invalid.
	case shipment.Date.IsZero():
		return domain.ErrInvalidDate // Return an error if the date is invalid.
	case shipment.Company <= 0:
		return domain.ErrInvalidCompany // Return an error if the company is invalid.
	}

	return s.r.AddOrUpdate(*shipment) // Store the shipment in the repository.
}

// IncrementShipmentUnitsCount increments the internal counter for received shipment units.
func (s ShipmentService) IncrementShipmentUnitsCount() {
	s.r.IncrementShipmentUnitsCount() // Calls the repository method to increment the shipment count.
}

// CreateShipmentService creates a new instance of ShipmentService with the provided repository.
func CreateShipmentService(repository domain.ShipmentRepository) (*ShipmentService, error) {
	if repository == nil {
		slog.Error("failed to create shipment service", "error", domain.ErrNilRepository)
		return nil, domain.ErrNilRepository // Return an error if the repository is nil.
	}

	return &ShipmentService{r: repository}, nil // Return a new instance of ShipmentService.
}
