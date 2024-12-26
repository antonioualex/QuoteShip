package persistence

import (
	"context"
	"errors"
	"log/slog"
	"sort"
	"strings"
	"sync"

	"cargoplot/domain"
)

var (
	ErrNilContext         = errors.New("context cannot be nil")
	ErrThresholdCounter   = errors.New("threshold count must be greater than 0")
	ErrOperationCancelled = errors.New("operation cancelled")
)

// ShipmentRepository manages shipmentInput offers with thread-safe operations.
type ShipmentRepository struct {
	shipmentsByOrigin   []domain.OriginShipments // shipmentsByOrigin is a slice of domain.OriginShipments, used to store shipmentInput offers by origin.
	latestShipmentBatch []domain.OriginShipments // latestShipmentBatch is a slice of domain.OriginShipments that stores the latest batch of shipmentInput offers, this batch is updated every thresholdCount.
	shipmentCount       int                      // shipmentCount is a counter that keeps track of the number of shipmentInput offers received, we use this to determine when to update the latestShipmentBatch.
	thresholdCount      int                      // thresholdCount is the number of shipmentInput offers to receive before updating the latestShipmentBatch, it acts like a recency threshold.
	mu                  sync.RWMutex             // mu is a read-write mutex that is used to synchronize access to shipmentInput data operations.
	ctx                 context.Context          // ctx is the context used to cancel operations when the context is cancelled.
}

// AddOrUpdate adds or updates a new domain.ShipmentUnit offer to the repository. If the offer is outdated or already exists,
// it will not be updated.
func (r *ShipmentRepository) AddOrUpdate(shipment domain.ShipmentUnit) error {
	err := validateShipment(shipment)
	if err != nil {
		return err
	}

	// Check if the operation is cancelled.
	select {
	case <-r.ctx.Done():
		return ErrOperationCancelled
	default:
		// Proceed with normal processing
	}

	r.mu.Lock()         // Lock the mutex to prevent concurrent access
	defer r.mu.Unlock() // Unlock the mutex when the function returns

	var wg sync.WaitGroup       // WaitGroup to wait for all goroutines to finish
	var muOrigin sync.Mutex     // Protects `updated`
	var updated bool            // Tracks if the shipmentInput was updated
	done := make(chan struct{}) // Signals early termination

	// Iterate over the shipmentsByOrigin to find the shipmentInput origin, we split the work into goroutines for each origin
	// to speed up the process.
	for i := range r.shipmentsByOrigin {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			select {
			case <-done:
				// Exit early if another goroutine has already updated
				return
			default:
			}

			if r.shipmentsByOrigin[i].Origin == shipment.Origin {
				muOrigin.Lock() // Protect shared variable
				if !updated {
					if r.upsertShipment(&r.shipmentsByOrigin[i], &shipment) {
						updated = true
						close(done) // Signal other goroutines to stop
					}
				}
				muOrigin.Unlock()
			}
		}(i)
	}

	wg.Wait() // Wait for all goroutines to finish

	// Append the new shipmentInput if it wasn't updated
	if !updated {
		r.shipmentsByOrigin = append(r.shipmentsByOrigin, domain.OriginShipments{
			Origin: shipment.Origin,
			Quotes: []domain.ShipmentQuote{
				{
					Company: shipment.Company,
					Price:   shipment.Price,
					Date:    shipment.Date,
				},
			},
		})
	}

	r.shipmentCount++

	// Check if the shipmentInput count has reached the threshold count
	r.manageBatch()

	return nil
}

// upsertShipment updates an existing shipmentInput if found, or adds it if the company does not own a shipmentInput quote for the
// inserted origin. Takes as arguments a slice of shipments (originShipmentsInput) and a shipmentInput unit to update.
func (r *ShipmentRepository) upsertShipment(originShipments *domain.OriginShipments, shipment *domain.ShipmentUnit) bool {
	// Check if the shipmentInput company already exists in the originShipmentsInput, if so update the shipmentInput if the new shipmentInput is more recent or has the same date but a lower price.
	for i, shipmentQuote := range originShipments.Quotes {
		if shipmentQuote.Company == shipment.Company {
			if shipment.Date.After(shipmentQuote.Date) {
				originShipments.Quotes[i] = domain.ShipmentQuote{
					Company: shipment.Company,
					Price:   shipment.Price,
					Date:    shipment.Date,
				}

				// remove origin shipment quote
				originShipments.Quotes = append(originShipments.Quotes[:i], originShipments.Quotes[i+1:]...)

				// find the correct index to insert the new shipment quote
				index := sort.Search(len(originShipments.Quotes), func(i int) bool {
					// First condition: sort by price, if equal sort by date
					if originShipments.Quotes[i].Price == shipment.Price {
						// Secondary condition: sort by company if the price and date are equal
						if originShipments.Quotes[i].Date.Equal(shipment.Date) {
							return originShipments.Quotes[i].Company > shipment.Company // return true if the company is greater
						}
						return originShipments.Quotes[i].Date.Before(shipment.Date) // return true if the date is before
					}
					return originShipments.Quotes[i].Price > shipment.Price // return true if the price is greater
				})

				// Insert the new shipment quote at the correct index
				originShipments.Quotes = append(originShipments.Quotes[:index], append([]domain.ShipmentQuote{shipment.ShipmentQuote}, originShipments.Quotes[index:]...)...)
			}
			return true
		}
	}

	// if company does not exist in the originShipmentsInput, find the correct index to insert the new shipment quote
	// Sort the quotes by price, date, and company
	index := sort.Search(len(originShipments.Quotes), func(i int) bool {
		// First condition: sort by price, if equal sort by date
		if originShipments.Quotes[i].Price == shipment.Price {
			// Secondary condition: sort by company if the price and date are equal
			if originShipments.Quotes[i].Date.Equal(shipment.Date) {
				return originShipments.Quotes[i].Company > shipment.Company // return true if the company is greater
			}
			return originShipments.Quotes[i].Date.Before(shipment.Date) // return true if the date is before
		}
		return originShipments.Quotes[i].Price > shipment.Price // return true if the price is greater
	})

	// add the new shipment quote at the correct index
	originShipments.Quotes = append(originShipments.Quotes[:index], append([]domain.ShipmentQuote{shipment.ShipmentQuote}, originShipments.Quotes[index:]...)...)

	return true
}

// manageBatch updates the shipmentInput batch and resets the shipmentInput count if the threshold count is reached.
func (r *ShipmentRepository) manageBatch() {
	if r.shipmentCount%r.thresholdCount == 0 {
		r.latestShipmentBatch = r.shipmentsByOrigin
		r.shipmentCount = 0
	}
}

// GetLatestSortedShipmentsByOrigin retrieves the latest shipments, sorted by price.
func (r *ShipmentRepository) GetLatestSortedShipmentsByOrigin() []domain.OriginShipments {
	// Check if the operation is cancelled
	select {
	case <-r.ctx.Done():
		return nil
	default:
	}

	r.mu.RLock()         // Lock the mutex for reading
	defer r.mu.RUnlock() // Unlock the mutex when the function returns

	return r.latestShipmentBatch
}

// IncrementShipmentUnitsCount increments the shipmentInput count.
func (r *ShipmentRepository) IncrementShipmentUnitsCount() {
	r.mu.Lock()         // Lock the mutex for writing
	defer r.mu.Unlock() // Unlock the mutex when the function returns

	r.shipmentCount++
}

// validateShipment validates the domain.ShipmentUnit argument fields.
func validateShipment(shipment domain.ShipmentUnit) error {
	switch {
	case strings.TrimSpace(shipment.Origin) == "":
		return domain.ErrInvalidOriginPort
	case shipment.Price <= 0:
		return domain.ErrInvalidPrice
	case shipment.Date.IsZero():
		return domain.ErrInvalidDate
	case shipment.Company <= 0:
		return domain.ErrInvalidCompany
	}
	return nil
}

// cleanup clears all the stored data.
func (r *ShipmentRepository) cleanup() {
	r.mu.Lock()         // Lock the mutex for writing
	defer r.mu.Unlock() // Unlock the mutex when the function returns

	r.shipmentsByOrigin = nil
	r.latestShipmentBatch = nil
	r.shipmentCount = 0
	slog.Warn("repository data has been cleared")
}

// NewShipmentOfferRepository initializes a new ShipmentRepository. It takes a context and a thresholdCount as arguments.
// The context is used to cancel operations when the context is cancelled, and the thresholdCount is the number of shipmentInput offers to
// receive before updating the latestShipmentBatch. The latestShipmentBatch is meant to be sent for calculating the estimates prices.
func NewShipmentOfferRepository(ctx context.Context, thresholdCount int) (*ShipmentRepository, error) {
	switch {
	case ctx == nil:
		slog.Error("failed to create repository", "error", ErrNilContext.Error())
		return nil, ErrNilContext
	case thresholdCount <= 0:
		slog.Error("failed to create repository", "error", ErrThresholdCounter.Error())
		return nil, ErrThresholdCounter
	}

	// Initialize a new ShipmentRepository
	repo := &ShipmentRepository{
		shipmentsByOrigin:   []domain.OriginShipments{},
		latestShipmentBatch: []domain.OriginShipments{},
		thresholdCount:      thresholdCount,
		ctx:                 ctx,
	}

	// Cleanup on context cancellation
	go func() {
		<-ctx.Done()
		repo.cleanup() // Clear the repository data
	}()

	return repo, nil
}
