package persistence

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"quoteship/domain"
)

var (
	testingOriginShipments []domain.OriginShipments
	testingShipmentUnit    domain.ShipmentUnit
)

func TestMain(m *testing.M) {
	testingOriginShipments = []domain.OriginShipments{

		{
			Origin: "LAX",
			Quotes: []domain.ShipmentQuote{
				{
					Company: 1,
					Price:   200,
					Date:    time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
				},
				{
					Company: 2,
					Price:   100,
					Date:    time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},
		},
		{
			Origin: "NYC",
			Quotes: []domain.ShipmentQuote{
				{
					Company: 1,
					Price:   150,
					Date:    time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},
		},
	}

	testingShipmentUnit = domain.ShipmentUnit{
		Origin: "AKR",
		ShipmentQuote: domain.ShipmentQuote{
			Company: 32,
			Price:   777,
			Date:    time.Date(2010, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	os.Exit(m.Run())
}

func TestNewShipmentOfferRepository(t *testing.T) {
	tests := []struct {
		name                string
		contextInput        context.Context
		thresholdCountInput int
		repository          func(ctx context.Context, i int) (*ShipmentRepository, error)
		expectedError       error
	}{
		{
			name:                "valid input",
			contextInput:        context.Background(),
			thresholdCountInput: 1,
			repository: func(ctx context.Context, i int) (*ShipmentRepository, error) {
				return NewShipmentOfferRepository(ctx, int(i))
			},
			expectedError: nil,
		},
		{
			name:                "invalid input - nil context",
			contextInput:        nil,
			thresholdCountInput: 1,
			repository: func(ctx context.Context, i int) (*ShipmentRepository, error) {
				return NewShipmentOfferRepository(ctx, int(i))
			},
			expectedError: ErrNilContext,
		},
		{
			name:                "invalid input - zero threshold count",
			contextInput:        context.Background(),
			thresholdCountInput: 0,
			repository: func(ctx context.Context, i int) (*ShipmentRepository, error) {
				return NewShipmentOfferRepository(ctx, int(i))
			},
			expectedError: ErrThresholdCounter,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.repository(tt.contextInput, int(tt.thresholdCountInput))

			if !errors.Is(err, tt.expectedError) {
				t.Errorf("expected error: %v, got: %v", tt.expectedError, err)
			}
		})
	}
}

func TestShipmentRepository_AddOrUpdate(t *testing.T) {
	tests := []struct {
		name                          string
		shipmentInput                 func() domain.ShipmentUnit
		expectedError                 error
		repository                    func(ctx context.Context, thresholdCount int) (*ShipmentRepository, error)
		repositoryContextInput        context.Context
		repositoryThresholdCountInput int
		expectedShipments             func() []domain.OriginShipments
	}{
		{
			name: "invalid shipmentInput - empty origin port",
			shipmentInput: func() domain.ShipmentUnit {
				testingShipmentInput := testingShipmentUnit
				testingShipmentInput.Origin = ""
				return testingShipmentInput
			},
			expectedError: domain.ErrInvalidOriginPort,
			repository: func(ctx context.Context, thresholdCount int) (*ShipmentRepository, error) {
				repository, err := NewShipmentOfferRepository(ctx, thresholdCount)
				if err != nil {
					return nil, err
				}
				repository.shipmentsByOrigin = testingOriginShipments
				repository.latestShipmentBatch = testingOriginShipments
				repository.shipmentCount = len(testingOriginShipments)
				return repository, nil
			},
			repositoryContextInput:        context.Background(),
			repositoryThresholdCountInput: len(testingOriginShipments),
			expectedShipments: func() []domain.OriginShipments {
				return testingOriginShipments
			},
		},
		{
			name: "invalid shipmentInput - invalid price",
			shipmentInput: func() domain.ShipmentUnit {
				testingShipmentInput := testingShipmentUnit
				testingShipmentInput.Price = -1
				return testingShipmentInput
			},
			expectedError: domain.ErrInvalidPrice,
			repository: func(ctx context.Context, thresholdCount int) (*ShipmentRepository, error) {
				repository, err := NewShipmentOfferRepository(ctx, thresholdCount)
				if err != nil {
					return nil, err
				}
				repository.shipmentsByOrigin = testingOriginShipments
				repository.latestShipmentBatch = testingOriginShipments
				repository.shipmentCount = len(testingOriginShipments)
				return repository, nil
			},
			repositoryContextInput:        context.Background(),
			repositoryThresholdCountInput: len(testingOriginShipments),
			expectedShipments:             func() []domain.OriginShipments { return testingOriginShipments },
		},
		{
			name: "invalid shipmentInput - invalid date",
			shipmentInput: func() domain.ShipmentUnit {
				testingShipmentInput := testingShipmentUnit
				testingShipmentInput.Date = time.Time{}
				return testingShipmentInput
			},
			expectedError: domain.ErrInvalidDate,
			repository: func(ctx context.Context, thresholdCount int) (*ShipmentRepository, error) {
				repository, err := NewShipmentOfferRepository(ctx, thresholdCount)
				if err != nil {
					return nil, err
				}
				repository.shipmentsByOrigin = testingOriginShipments
				repository.latestShipmentBatch = testingOriginShipments
				repository.shipmentCount = len(testingOriginShipments)
				return repository, nil
			},
			repositoryContextInput:        context.Background(),
			repositoryThresholdCountInput: len(testingOriginShipments),
			expectedShipments: func() []domain.OriginShipments {
				return testingOriginShipments
			},
		},
		{
			name: "invalid shipmentInput - invalid company",
			shipmentInput: func() domain.ShipmentUnit {
				testingShipmentInput := testingShipmentUnit
				testingShipmentInput.Company = 0
				return testingShipmentInput
			},
			expectedError: domain.ErrInvalidCompany,
			repository: func(ctx context.Context, thresholdCount int) (*ShipmentRepository, error) {
				repository, err := NewShipmentOfferRepository(ctx, thresholdCount)
				if err != nil {
					return nil, err
				}
				repository.shipmentsByOrigin = testingOriginShipments
				repository.latestShipmentBatch = testingOriginShipments
				repository.shipmentCount = len(testingOriginShipments)
				return repository, nil
			},
			repositoryContextInput:        context.Background(),
			repositoryThresholdCountInput: len(testingOriginShipments),
			expectedShipments: func() []domain.OriginShipments {
				return testingOriginShipments
			},
		},
		{
			name: "valid shipmentInput - added",
			shipmentInput: func() domain.ShipmentUnit {
				return testingShipmentUnit
			},
			expectedError: nil,
			repository: func(ctx context.Context, thresholdCount int) (*ShipmentRepository, error) {
				repository, err := NewShipmentOfferRepository(ctx, thresholdCount)
				if err != nil {
					return nil, err
				}
				repository.shipmentsByOrigin = testingOriginShipments
				repository.latestShipmentBatch = testingOriginShipments
				repository.shipmentCount = len(testingOriginShipments)
				return repository, nil
			},
			repositoryContextInput:        context.Background(),
			repositoryThresholdCountInput: len(testingOriginShipments),
			expectedShipments: func() []domain.OriginShipments {
				return append(testingOriginShipments, domain.OriginShipments{
					Origin: testingShipmentUnit.Origin,
					Quotes: []domain.ShipmentQuote{
						{
							Company: testingShipmentUnit.Company,
							Price:   testingShipmentUnit.Price,
							Date:    testingShipmentUnit.Date,
						},
					},
				})
			},
		},
		{
			name: "valid shipmentInput - updated",
			shipmentInput: func() domain.ShipmentUnit {
				return domain.ShipmentUnit{
					Origin: testingOriginShipments[0].Origin,
					ShipmentQuote: domain.ShipmentQuote{
						Company: testingOriginShipments[0].Quotes[0].Company,
						Price:   testingOriginShipments[0].Quotes[0].Price + 1,
						Date:    time.Date(2044, 1, 1, 0, 0, 0, 0, time.UTC),
					},
				}
			},
			expectedError: nil,
			repository: func(ctx context.Context, thresholdCount int) (*ShipmentRepository, error) {
				repository, err := NewShipmentOfferRepository(ctx, thresholdCount)
				if err != nil {
					return nil, err
				}
				repository.shipmentsByOrigin = testingOriginShipments
				repository.latestShipmentBatch = testingOriginShipments
				repository.shipmentCount = len(testingOriginShipments)
				return repository, nil
			},
			repositoryContextInput:        context.Background(),
			repositoryThresholdCountInput: len(testingOriginShipments),
			expectedShipments: func() []domain.OriginShipments {
				updatedTestingOriginShipments := testingOriginShipments
				updatedTestingOriginShipments[0].Quotes[0].Price++
				updatedTestingOriginShipments[0].Quotes[0].Date = time.Date(2044, 1, 1, 0, 0, 0, 0, time.UTC)
				return updatedTestingOriginShipments
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, err := tt.repository(tt.repositoryContextInput, tt.repositoryThresholdCountInput)
			if err != nil {
				t.Fatalf("failed to create repository: %v", err)
			}

			err = repo.AddOrUpdate(tt.shipmentInput())
			if !errors.Is(err, tt.expectedError) {
				t.Errorf("expected error %v, got %v", tt.expectedError, err)
			}

			if len(tt.expectedShipments()) != len(repo.shipmentsByOrigin) {
				t.Errorf("expected shipments length %d, got %d", len(tt.expectedShipments()), len(repo.shipmentsByOrigin))
			}

			for i, expectedOriginShipments := range tt.expectedShipments() {
				if expectedOriginShipments.Origin != repo.shipmentsByOrigin[i].Origin {
					t.Errorf("expected origin %s, got %s", expectedOriginShipments.Origin, repo.shipmentsByOrigin[i].Origin)
				}

				if len(expectedOriginShipments.Quotes) != len(repo.shipmentsByOrigin[i].Quotes) {
					t.Errorf("expected quotes length %d, got %d", len(expectedOriginShipments.Quotes), len(repo.shipmentsByOrigin[i].Quotes))
				}

				for j, expectedQuote := range expectedOriginShipments.Quotes {
					if expectedQuote.Company != repo.shipmentsByOrigin[i].Quotes[j].Company {
						t.Errorf("expected company %d, got %d", expectedQuote.Company, repo.shipmentsByOrigin[i].Quotes[j].Company)
					}

					if expectedQuote.Price != repo.shipmentsByOrigin[i].Quotes[j].Price {
						t.Errorf("expected price %d, got %d", expectedQuote.Price, repo.shipmentsByOrigin[i].Quotes[j].Price)
					}

					if expectedQuote.Date != repo.shipmentsByOrigin[i].Quotes[j].Date {
						t.Errorf("expected date %v, got %v", expectedQuote.Date, repo.shipmentsByOrigin[i].Quotes[j].Date)
					}
				}
			}
		})
	}
}

func TestShipmentRepository_cleanup(t *testing.T) {
	tests := []struct {
		name                          string
		repository                    func(ctx context.Context, thresholdCount int) (*ShipmentRepository, error)
		repositoryContextInput        context.Context
		repositoryThresholdCountInput int
	}{
		{
			name: "valid cleanup",
			repository: func(ctx context.Context, thresholdCount int) (*ShipmentRepository, error) {
				repository, err := NewShipmentOfferRepository(ctx, thresholdCount)
				if err != nil {
					return nil, err
				}
				repository.shipmentsByOrigin = testingOriginShipments
				repository.latestShipmentBatch = testingOriginShipments
				repository.shipmentCount = len(testingOriginShipments)
				return repository, nil
			},
			repositoryContextInput:        context.Background(),
			repositoryThresholdCountInput: len(testingOriginShipments),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, err := tt.repository(tt.repositoryContextInput, tt.repositoryThresholdCountInput)
			if err != nil {
				t.Fatalf("failed to create repository: %v", err)
			}

			repo.cleanup()

			if len(repo.shipmentsByOrigin) != 0 {
				t.Errorf("expected shipments length 0, got %d", len(repo.shipmentsByOrigin))
			}

			if len(repo.latestShipmentBatch) != 0 {
				t.Errorf("expected latest shipmentInput batch length 0, got %d", len(repo.latestShipmentBatch))
			}

			if repo.shipmentCount != 0 {
				t.Errorf("expected shipmentInput count 0, got %d", repo.shipmentCount)
			}
		})
	}
}

func TestShipmentRepository_GetLatestSortedShipmentsByOrigin(t *testing.T) {
	tests := []struct {
		name                          string
		repository                    func(ctx context.Context, thresholdCount int) (*ShipmentRepository, error)
		repositoryContextInput        context.Context
		repositoryThresholdCountInput int
		expectedShipments             func() []domain.OriginShipments
	}{
		{
			name: "valid get latest sorted shipments",
			repository: func(ctx context.Context, thresholdCount int) (*ShipmentRepository, error) {
				repository, err := NewShipmentOfferRepository(ctx, thresholdCount)
				if err != nil {
					return nil, err
				}
				repository.shipmentsByOrigin = testingOriginShipments
				repository.latestShipmentBatch = testingOriginShipments
				repository.shipmentCount = len(testingOriginShipments)
				return repository, nil
			},
			repositoryContextInput:        context.Background(),
			repositoryThresholdCountInput: len(testingOriginShipments),
			expectedShipments: func() []domain.OriginShipments {
				return testingOriginShipments
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, err := tt.repository(tt.repositoryContextInput, tt.repositoryThresholdCountInput)
			if err != nil {
				t.Fatalf("failed to create repository: %v", err)
			}

			shipments := repo.GetLatestSortedShipmentsByOrigin()

			if len(shipments) != len(tt.expectedShipments()) {
				t.Errorf("expected shipments length %d, got %d", len(tt.expectedShipments()), len(shipments))
			}

			for i, expectedOriginShipments := range tt.expectedShipments() {
				if expectedOriginShipments.Origin != shipments[i].Origin {
					t.Errorf("expected origin %s, got %s", expectedOriginShipments.Origin, shipments[i].Origin)
				}

				if len(expectedOriginShipments.Quotes) != len(shipments[i].Quotes) {
					t.Errorf("expected quotes length %d, got %d", len(expectedOriginShipments.Quotes), len(shipments[i].Quotes))
				}

				for j, expectedQuote := range expectedOriginShipments.Quotes {
					if expectedQuote.Company != shipments[i].Quotes[j].Company {
						t.Errorf("expected company %d, got %d", expectedQuote.Company, shipments[i].Quotes[j].Company)
					}
				}
			}
		})
	}
}

func TestShipmentRepository_upsertShipment(t *testing.T) {
	tests := []struct {
		name                          string
		repository                    func(ctx context.Context, thresholdCount int) (*ShipmentRepository, error)
		repositoryContextInput        context.Context
		repositoryThresholdCountInput int
		originShipmentsInput          func() *domain.OriginShipments
		expectedShipments             func() []domain.OriginShipments
		shipmentInput                 func() *domain.ShipmentUnit
		expectedUpdated               bool
	}{
		{
			name: "valid upsert - added",
			repository: func(ctx context.Context, thresholdCount int) (*ShipmentRepository, error) {
				repository, err := NewShipmentOfferRepository(ctx, thresholdCount)
				if err != nil {
					return nil, err
				}
				repository.shipmentsByOrigin = testingOriginShipments
				repository.latestShipmentBatch = testingOriginShipments
				repository.shipmentCount = len(testingOriginShipments)
				return repository, nil
			},
			repositoryContextInput:        context.Background(),
			repositoryThresholdCountInput: len(testingOriginShipments),
			originShipmentsInput: func() *domain.OriginShipments {
				return &testingOriginShipments[0]
			},
			shipmentInput: func() *domain.ShipmentUnit {
				return &domain.ShipmentUnit{
					Origin: testingOriginShipments[0].Origin,
					ShipmentQuote: domain.ShipmentQuote{
						Company: 666,
						Price:   333,
						Date:    time.Now(),
					},
				}
			},
			expectedUpdated: true,
			expectedShipments: func() []domain.OriginShipments {
				updatedTestingOriginShipments := testingOriginShipments
				updatedTestingOriginShipments[0].Quotes = append(updatedTestingOriginShipments[0].Quotes, domain.ShipmentQuote{
					Company: 666,
					Price:   333,
					Date:    time.Now(),
				})
				return updatedTestingOriginShipments
			},
		},
		{
			name: "valid upsert - updated",
			repository: func(ctx context.Context, thresholdCount int) (*ShipmentRepository, error) {
				repository, err := NewShipmentOfferRepository(ctx, thresholdCount)
				if err != nil {
					return nil, err
				}
				repository.shipmentsByOrigin = testingOriginShipments
				repository.latestShipmentBatch = testingOriginShipments
				repository.shipmentCount = len(testingOriginShipments)
				return repository, nil
			},
			repositoryContextInput:        context.Background(),
			repositoryThresholdCountInput: len(testingOriginShipments),
			originShipmentsInput: func() *domain.OriginShipments {
				return &testingOriginShipments[0]
			},
			shipmentInput: func() *domain.ShipmentUnit {
				return &domain.ShipmentUnit{
					Origin: testingOriginShipments[0].Origin,
					ShipmentQuote: domain.ShipmentQuote{
						Company: testingOriginShipments[0].Quotes[0].Company,
						Price:   testingOriginShipments[0].Quotes[0].Price + 1,
						Date:    time.Date(2044, 1, 1, 0, 0, 0, 0, time.UTC),
					},
				}
			},
			expectedUpdated: true,
			expectedShipments: func() []domain.OriginShipments {
				updatedTestingOriginShipments := testingOriginShipments
				updatedTestingOriginShipments[0].Quotes[0].Price++
				updatedTestingOriginShipments[0].Quotes[0].Date = time.Date(2044, 1, 1, 0, 0, 0, 0, time.UTC)
				return updatedTestingOriginShipments
			},
		},
		{
			name: "valid upsert - not updated",
			repository: func(ctx context.Context, thresholdCount int) (*ShipmentRepository, error) {
				repository, err := NewShipmentOfferRepository(ctx, thresholdCount)
				if err != nil {
					return nil, err
				}
				repository.shipmentsByOrigin = testingOriginShipments
				repository.latestShipmentBatch = testingOriginShipments
				repository.shipmentCount = len(testingOriginShipments)
				return repository, nil
			},
			repositoryContextInput:        context.Background(),
			repositoryThresholdCountInput: len(testingOriginShipments),
			originShipmentsInput: func() *domain.OriginShipments {
				return &testingOriginShipments[0]
			},
			shipmentInput: func() *domain.ShipmentUnit {
				return &domain.ShipmentUnit{
					Origin: testingOriginShipments[0].Origin,
					ShipmentQuote: domain.ShipmentQuote{
						Company: testingOriginShipments[0].Quotes[0].Company,
						Price:   testingOriginShipments[0].Quotes[0].Price,
						Date:    testingOriginShipments[0].Quotes[0].Date,
					},
				}
			},
			expectedUpdated: true,
			expectedShipments: func() []domain.OriginShipments {
				return testingOriginShipments
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository, err := tt.repository(tt.repositoryContextInput, tt.repositoryThresholdCountInput)
			if err != nil {
				t.Fatalf("failed to create repository: %v", err)
			}

			originShipments := tt.originShipmentsInput()
			shipment := tt.shipmentInput()
			updated := repository.upsertShipment(originShipments, shipment)

			if updated != tt.expectedUpdated {
				t.Errorf("expected updated %t, got %t", tt.expectedUpdated, updated)
			}

			if len(tt.expectedShipments()) != len(repository.shipmentsByOrigin) {
				t.Errorf("expected shipments length %d, got %d", len(tt.expectedShipments()), len(repository.shipmentsByOrigin))
			}

			for i, expectedOriginShipments := range tt.expectedShipments() {
				if expectedOriginShipments.Origin != repository.shipmentsByOrigin[i].Origin {
					t.Errorf("expected origin %s, got %s", expectedOriginShipments.Origin, repository.shipmentsByOrigin[i].Origin)
				}

				if len(expectedOriginShipments.Quotes) != len(repository.shipmentsByOrigin[i].Quotes) {
					t.Errorf("expected quotes length %d, got %d", len(expectedOriginShipments.Quotes), len(repository.shipmentsByOrigin[i].Quotes))
				}

				for j, expectedQuote := range expectedOriginShipments.Quotes {
					if expectedQuote.Company != repository.shipmentsByOrigin[i].Quotes[j].Company {
						t.Errorf("expected company %d, got %d", expectedQuote.Company, repository.shipmentsByOrigin[i].Quotes[j].Company)
					}

					if expectedQuote.Price != repository.shipmentsByOrigin[i].Quotes[j].Price {
						t.Errorf("expected price %d, got %d", expectedQuote.Price, repository.shipmentsByOrigin[i].Quotes[j].Price)
					}

					if expectedQuote.Date != repository.shipmentsByOrigin[i].Quotes[j].Date {
						t.Errorf("expected date %v, got %v", expectedQuote.Date, repository.shipmentsByOrigin[i].Quotes[j].Date)
					}
				}
			}
		})
	}
}
