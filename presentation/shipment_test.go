package presentation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"quoteship/app"
	"quoteship/domain"
	"quoteship/persistence"
)

func TestShipmentHandler_SubmitShipmentOfferOffer(t *testing.T) {
	ctx := context.Background()
	shipmentRepository, err := persistence.NewShipmentOfferRepository(ctx, 1)
	if err != nil {
		t.Fatalf("failed to create shipment repository: %v", err)
	}
	shipmentService, err := app.CreateShipmentService(shipmentRepository)
	if err != nil {
		t.Fatalf("failed to create shipment service: %v", err)
	}

	handler := ShipmentHandler{
		s: shipmentService,
	}

	tests := []struct {
		name                       string
		contentType                string
		body                       interface{}
		expectedStatus             int
		expectedBody               interface{}
		expectedShipmentUnitsCount int
	}{
		{
			name:           "Invalid Content-Type",
			contentType:    "text/plain",
			body:           requestedShipmentOffer{Company: 1, Price: 100, Origin: "NYC", Date: "2023-01-01"},
			expectedStatus: http.StatusUnsupportedMediaType,
			expectedBody:   fmt.Sprintf(`{"error":"%s"}`+"\n", ErrInvalidContentType.Error()),
		},
		{
			name:           "Missing Content-Type",
			contentType:    "",
			body:           requestedShipmentOffer{Company: 1, Price: 100, Origin: "NYC", Date: "2023-01-01"},
			expectedStatus: http.StatusUnsupportedMediaType,
			expectedBody:   fmt.Sprintf(`{"error":"%s"}`+"\n", ErrInvalidContentType.Error()),
		},
		{
			name:           "Invalid request payload",
			contentType:    "application/json",
			body:           `not a struct neither a json formatted string`,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   fmt.Sprintf(`{"error":"%s"}`+"\n", ErrInvalidRequestPayload.Error()),
		},
		{
			name:           "Invalid date format",
			contentType:    "application/json",
			body:           requestedShipmentOffer{Company: 1, Price: 100, Origin: OriginShanghai, Date: "01-01-2023"},
			expectedStatus: http.StatusOK,
			expectedBody:   "",
		},
		{
			name:           "Invalid company - lower bound",
			contentType:    "application/json",
			body:           requestedShipmentOffer{Company: 0, Price: 100, Origin: OriginShanghai, Date: "2023-01-01"},
			expectedStatus: http.StatusOK,
			expectedBody:   "",
		},
		{
			name:           "Invalid company - upper bound",
			contentType:    "application/json",
			body:           requestedShipmentOffer{Company: 1000, Price: 100, Origin: OriginShanghai, Date: "2023-01-01"},
			expectedStatus: http.StatusOK,
			expectedBody:   "",
		},
		{
			name:           "Invalid price - lower bound",
			contentType:    "application/json",
			body:           requestedShipmentOffer{Company: 1, Price: 0, Origin: OriginShanghai, Date: "2023-01-01"},
			expectedStatus: http.StatusOK,
			expectedBody:   "",
		},
		{
			name:           "Invalid price - upper bound",
			contentType:    "application/json",
			body:           requestedShipmentOffer{Company: 1, Price: 100000, Origin: OriginShanghai, Date: "2023-01-01"},
			expectedStatus: http.StatusOK,
			expectedBody:   "",
		},
		{
			name:                       "Valid request",
			contentType:                "application/json",
			body:                       requestedShipmentOffer{Company: 1, Price: 100, Origin: OriginShanghai, Date: "2023-01-01"},
			expectedStatus:             http.StatusOK,
			expectedBody:               "",
			expectedShipmentUnitsCount: len(shipmentRepository.GetLatestSortedShipmentsByOrigin()) + 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert body struct to JSON
			var bodyBytes []byte
			if tt.body != nil {
				var err error
				bodyBytes, err = json.Marshal(tt.body)
				if err != nil {
					t.Fatalf("Failed to marshal JSON body: %v", err)
				}
			}

			// Create a new HTTP request
			req := httptest.NewRequest(http.MethodPost, "/add-offer", bytes.NewBuffer(bodyBytes))
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}

			// Record the response
			rec := httptest.NewRecorder()
			handler.SubmitShipmentOffer(rec, req)

			// Check the status code
			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			// Check the response body
			switch expectedBody := tt.expectedBody.(type) {
			case string:
				// Compare as plain text
				if rec.Body.String() != expectedBody {
					t.Errorf("expected body %q, got %q", expectedBody, rec.Body.String())
				}
			default:
				// Marshal the expected body to JSON for comparison
				expectedBodyBytes, err := json.Marshal(expectedBody)
				if err != nil {
					t.Fatalf("Failed to marshal expected body: %v", err)
				}
				if rec.Body.String() != string(expectedBodyBytes) {
					t.Errorf("expected body %q, got %q", string(expectedBodyBytes), rec.Body.String())
				}
			}

			// Check the number of shipment units
			if len(shipmentRepository.GetLatestSortedShipmentsByOrigin()) != tt.expectedShipmentUnitsCount {
				t.Errorf("expected %d shipment units, got %d", tt.expectedShipmentUnitsCount, len(shipmentRepository.GetLatestSortedShipmentsByOrigin()))
			}
		})
	}
}

func TestShipmentHandler_GetLatestExpectedRates(t *testing.T) {
	ctx := context.Background()
	shipmentRepository, err := persistence.NewShipmentOfferRepository(ctx, 1)
	if err != nil {
		t.Fatalf("failed to create shipment repository: %v", err)
	}

	err = shipmentRepository.AddOrUpdate(domain.ShipmentUnit{
		Origin: "NYC",
		ShipmentQuote: domain.ShipmentQuote{
			Company: 1,
			Price:   100,
			Date:    time.Now(),
		},
	})
	if err != nil {
		t.Fatalf("failed to add shipment unit: %v", err)
	}

	err = shipmentRepository.AddOrUpdate(domain.ShipmentUnit{
		Origin: "NYC",
		ShipmentQuote: domain.ShipmentQuote{
			Company: 2,
			Price:   150,
			Date:    time.Now(),
		},
	})
	if err != nil {
		t.Fatalf("failed to add shipment unit: %v", err)
	}

	err = shipmentRepository.AddOrUpdate(domain.ShipmentUnit{
		Origin: "LA",
		ShipmentQuote: domain.ShipmentQuote{
			Company: 1,
			Price:   466,
			Date:    time.Now(),
		},
	})
	if err != nil {
		t.Fatalf("failed to add shipment unit: %v", err)
	}

	shipmentService, err := app.CreateShipmentService(shipmentRepository)
	if err != nil {
		t.Fatalf("failed to create shipment service: %v", err)
	}

	handler := ShipmentHandler{
		s: shipmentService,
	}

	tests := []struct {
		name           string
		expectedStatus int
		expectedBody   interface{}
	}{
		{
			name:           "valid request",
			expectedStatus: http.StatusOK,
			expectedBody:   map[string]int{"NYC": 125, "LA": 466},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new HTTP request
			req := httptest.NewRequest(http.MethodGet, "/expected-rates", nil)

			// Record the response
			rec := httptest.NewRecorder()
			handler.GetLatestExpectedRates(rec, req)

			// Check the status code
			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			// Check the response body
			switch expectedBody := tt.expectedBody.(type) {
			case string:
				// Compare as plain text
				if rec.Body.String() != expectedBody {
					t.Errorf("expected body %q, got %q", expectedBody, rec.Body.String())
				}
			default:
				// Marshal the expected body to JSON for comparison
				expectedBodyBytes, err := json.Marshal(expectedBody)
				if err != nil {
					t.Fatalf("Failed to marshal expected body: %v", err)
				}
				if rec.Body.String() != string(expectedBodyBytes) {
					t.Errorf("expected body %q, got %q", string(expectedBodyBytes), rec.Body.String())
				}
			}
		})
	}
}

func TestShipmentHandler_validateAndParseShipment(t *testing.T) {
	tests := []struct {
		name                 string
		offer                requestedShipmentOffer
		expectedError        error
		expectedShipmentUnit domain.ShipmentUnit
	}{
		{
			name: "Valid request",
			offer: requestedShipmentOffer{
				Company: 1,
				Price:   100,
				Origin:  OriginShanghai,
				Date:    "2023-01-01",
			},
			expectedError: nil,
			expectedShipmentUnit: domain.ShipmentUnit{
				Origin: OriginShanghai,
				ShipmentQuote: domain.ShipmentQuote{
					Company: 1,
					Price:   100,
					Date:    time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},
		},
		{
			name: "Invalid company",
			offer: requestedShipmentOffer{
				Company: 0,
				Price:   100,
				Origin:  OriginShanghai,
				Date:    "2023-01-01",
			},
			expectedError: domain.ErrInvalidCompany,
		},
		{
			name: "Invalid price",
			offer: requestedShipmentOffer{
				Company: 1,
				Price:   0,
				Origin:  OriginShanghai,
				Date:    "2023-01-01",
			},
			expectedError: domain.ErrInvalidPrice,
		},
		{
			name: "Invalid origin",
			offer: requestedShipmentOffer{
				Company: 1,
				Price:   100,
				Origin:  "invalid",
				Date:    "2023-01-01",
			},
			expectedError: domain.ErrInvalidOriginPort,
		},
		{
			name: "Invalid date",
			offer: requestedShipmentOffer{
				Company: 1,
				Price:   100,
				Origin:  OriginShanghai,
				Date:    "01-01-2023",
			},
			expectedError: domain.ErrInvalidDate,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shipment, err := validateAndParseShipment(tt.offer)
			if !errors.Is(err, tt.expectedError) {
				t.Errorf("expected error %v, got %v", tt.expectedError, err)
			}
			if err == nil && shipment != tt.expectedShipmentUnit {
				t.Errorf("expected shipment %+v, got %+v", tt.expectedShipmentUnit, shipment)
			}
		})
	}

}
