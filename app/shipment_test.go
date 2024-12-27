package app

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"quoteship/domain"
	"quoteship/persistence"
)

func TestShipmentService_GetLatestExpectedRates(t *testing.T) {
	tests := []struct {
		name          string
		repository    func(context.Context, int) (*persistence.ShipmentRepository, error)
		input         int
		expectedError error
		expectedRates map[string]int
	}{
		{
			name: "invalid input - negative top",
			repository: func(ctx context.Context, i int) (*persistence.ShipmentRepository, error) {
				repository, err := persistence.NewShipmentOfferRepository(context.Background(), 1)
				if err != nil {
					return nil, err
				}
				return repository, nil
			},
			input:         -1,
			expectedError: domain.ErrInvalidTopValue,
			expectedRates: nil,
		},
		{
			name: "received no expected rates from repository",
			repository: func(ctx context.Context, i int) (*persistence.ShipmentRepository, error) {
				repository, err := persistence.NewShipmentOfferRepository(context.Background(), i)
				if err != nil {
					return nil, err
				}
				return repository, nil
			},
			input:         1,
			expectedError: domain.ErrNoExpectedRates,
			expectedRates: nil,
		},
		{
			name: "valid input - single origin",
			repository: func(ctx context.Context, i int) (*persistence.ShipmentRepository, error) {
				repository, err := persistence.NewShipmentOfferRepository(context.Background(), i)
				if err != nil {
					return nil, err
				}
				repository.AddOrUpdate(domain.ShipmentUnit{
					Origin: "NYC",
					ShipmentQuote: domain.ShipmentQuote{
						Company: 1,
						Price:   100,
						Date:    time.Now(),
					},
				})

				return repository, nil
			},
			input: 1,
			expectedRates: map[string]int{
				"NYC": 100,
			},
			expectedError: nil,
		},
		{
			name: "valid input - multiple origins",
			repository: func(ctx context.Context, i int) (*persistence.ShipmentRepository, error) {
				repository, err := persistence.NewShipmentOfferRepository(context.Background(), i)
				if err != nil {
					return nil, err
				}
				repository.AddOrUpdate(domain.ShipmentUnit{
					Origin: "NYC",
					ShipmentQuote: domain.ShipmentQuote{
						Company: 1,
						Price:   100,
						Date:    time.Now(),
					},
				})
				repository.AddOrUpdate(domain.ShipmentUnit{
					Origin: "LAX",
					ShipmentQuote: domain.ShipmentQuote{
						Company: 2,
						Price:   100,
						Date:    time.Now(),
					},
				})

				repository.AddOrUpdate(domain.ShipmentUnit{
					Origin: "LAX",
					ShipmentQuote: domain.ShipmentQuote{
						Company: 1,
						Price:   200,
						Date:    time.Now(),
					},
				})

				return repository, nil
			},
			input: 10,
			expectedRates: map[string]int{
				"NYC": 100,
				"LAX": 150,
			},
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shipmentRepository, _ := tt.repository(context.Background(), 1)
			service, err := CreateShipmentService(shipmentRepository)
			if err != nil {
				t.Fatalf("failed to create shipment service: %v", err)
			}

			rates, err := service.GetLatestExpectedRates(tt.input)
			if !errors.Is(err, tt.expectedError) {
				t.Errorf("expected error %v, got %v", tt.expectedError, err)
			}

			if !reflect.DeepEqual(rates, tt.expectedRates) {
				t.Errorf("expected rates %v, got %v", tt.expectedRates, rates)
			}
		})
	}
}

func TestShipmentService_SubmitShipment(t *testing.T) {
	shipmentUnit := &domain.ShipmentUnit{
		Origin: "NYC",
		ShipmentQuote: domain.ShipmentQuote{
			Company: 1,
			Price:   100,
			Date:    time.Now(),
		},
	}

	tests := []struct {
		name          string
		repository    func(context.Context, int) (*persistence.ShipmentRepository, error)
		input         *domain.ShipmentUnit
		expectedError error
	}{
		{
			name: "invalid shipment - nil input",
			repository: func(ctx context.Context, i int) (*persistence.ShipmentRepository, error) {
				return &persistence.ShipmentRepository{}, nil
			},
			input:         nil,
			expectedError: domain.ErrNilShipmentUnit,
		},
		{
			name: "invalid shipment - empty origin port",
			repository: func(ctx context.Context, i int) (*persistence.ShipmentRepository, error) {
				return &persistence.ShipmentRepository{}, nil
			},
			input: &domain.ShipmentUnit{
				Origin: "",
				ShipmentQuote: domain.ShipmentQuote{
					Company: shipmentUnit.Company,
					Price:   shipmentUnit.Price,
					Date:    shipmentUnit.Date,
				},
			},
			expectedError: domain.ErrInvalidOriginPort,
		},
		{
			name: "invalid shipment - invalid price",
			repository: func(ctx context.Context, i int) (*persistence.ShipmentRepository, error) {
				return &persistence.ShipmentRepository{}, nil
			},
			input: &domain.ShipmentUnit{
				Origin: shipmentUnit.Origin,
				ShipmentQuote: domain.ShipmentQuote{
					Company: shipmentUnit.Company,
					Price:   -1,
					Date:    shipmentUnit.Date,
				},
			},
			expectedError: domain.ErrInvalidPrice,
		},
		{
			name: "invalid shipment - invalid date",
			repository: func(ctx context.Context, i int) (*persistence.ShipmentRepository, error) {
				return &persistence.ShipmentRepository{}, nil
			},
			input: &domain.ShipmentUnit{
				Origin: shipmentUnit.Origin,
				ShipmentQuote: domain.ShipmentQuote{
					Company: shipmentUnit.Company,
					Price:   shipmentUnit.Price,
					Date:    time.Time{},
				},
			},
			expectedError: domain.ErrInvalidDate,
		},
		{
			name: "invalid shipment - invalid company",
			repository: func(ctx context.Context, i int) (*persistence.ShipmentRepository, error) {
				return &persistence.ShipmentRepository{}, nil
			},
			input: &domain.ShipmentUnit{
				Origin: shipmentUnit.Origin,
				ShipmentQuote: domain.ShipmentQuote{
					Company: 0,
					Price:   shipmentUnit.Price,
					Date:    shipmentUnit.Date,
				},
			},
			expectedError: domain.ErrInvalidCompany,
		},
		{
			name: "valid shipment",
			repository: func(ctx context.Context, i int) (*persistence.ShipmentRepository, error) {
				return persistence.NewShipmentOfferRepository(ctx, i)
			},
			input:         shipmentUnit,
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository, err := tt.repository(context.Background(), 1)
			if err != nil {
				t.Fatalf("failed to create shipment repository: %v", err)
			}
			service, err := CreateShipmentService(repository)
			if err != nil {
				t.Fatalf("failed to create shipment service: %v", err)
			}

			err = service.SubmitShipment(tt.input)
			if !errors.Is(err, tt.expectedError) {
				t.Errorf("expected error %v, got %v", tt.expectedError, err)
			}
		})
	}
}

func TestCreateShipmentService(t *testing.T) {
	tests := []struct {
		name                    string
		repository              func(context.Context, int) (*persistence.ShipmentRepository, error)
		expectedRepositoryError error
		expectedServiceError    error
	}{
		{
			name: "valid repository",
			repository: func(ctx context.Context, i int) (*persistence.ShipmentRepository, error) {
				return persistence.NewShipmentOfferRepository(ctx, i)
			},
			expectedServiceError:    nil,
			expectedRepositoryError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository, err := tt.repository(context.Background(), 1)
			if !errors.Is(err, tt.expectedRepositoryError) {
				t.Errorf("expected repository error %v, got %v", tt.expectedRepositoryError, err)
			}

			_, err = CreateShipmentService(repository)
			if !errors.Is(err, tt.expectedServiceError) {
				t.Errorf("expected service error %v, got %v", tt.expectedServiceError, err)
			}
		})
	}
}
