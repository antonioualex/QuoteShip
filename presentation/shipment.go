package presentation

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"cargoplot/domain"
)

const (
	OriginShanghai  = "CNSGH" // Shanghai
	OriginSingapore = "SGSIN" // Singapore
	OriginShenzhen  = "CNSNZ" // Shenzhen
	OriginNingbo    = "CNNBO" // Ningbo
	OriginGuangzhou = "CNGGZ" // Guangzhou

	MinCompanyID = 1
	MaxCompanyID = 999
	MinPrice     = 1
	MaxPrice     = 99999

	dateFormat = "2006-01-02" // Go's reference format for date parsing
)

var (
	ErrInvalidRequestPayload = errors.New("invalid request payload")
	ErrInvalidContentType    = errors.New("invalid content type")
	ErrIntervalServerError   = errors.New("internal server error")

	expectedRatesPerOriginNum = 10 // Number of expected rates per origin port
)

// ShipmentHandler is a struct that contains the domain.ShipmentService interface. Through this interface, the handler can
// interact with the domain layer to perform operations related to shipment data.
type ShipmentHandler struct {
	s domain.ShipmentService // s is the service that provides business logic for managing and retrieving shipment data.
}

// requestedShipmentOffer is a struct that represents the expected structure of a shipment offer request payload. This
// struct is used to decode the request body for requested shipment offers.
type requestedShipmentOffer struct {
	Company int    `json:"company"` // Company is the name of the company that provided the quote.
	Price   int    `json:"price"`   // Price is the cost of the shipment.
	Origin  string `json:"origin"`  // Origin is the located port where the shipment starts (e.g., "CNSGH").
	Date    string `json:"date"`    // Date is the date when the shipment will start. It should be in the format "YYYY-MM-DD".
}

// GetLatestExpectedRates is an HTTP handler that retrieves the latest expected rates for shipments grouped by origin and
// sorted by price. It considers the `top` lowest-priced offers for each origin and returns the expected rates.
// The handler returns a JSON response containing the expected rates for each origin port, e.g., {"CNSGH": 100, "SGSIN": 200}.
func (h ShipmentHandler) GetLatestExpectedRates(writer http.ResponseWriter, _ *http.Request) {
	// Calling the GetLatestExpectedRates method from the service layer to get the expected rates
	expectedRates, err := h.s.GetLatestExpectedRates(expectedRatesPerOriginNum)
	if err != nil {
		// Return a nil response with a status of OK
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusBadRequest)

		// Write a JSON null to the response body
		_, writeErr := writer.Write([]byte("null"))
		if writeErr != nil {
			slog.Error("error writing null response", "error", writeErr)
		}
		return
	}

	// Marshal the expected rates into a JSON byte slice
	expectedRatesMarshaled, err := json.Marshal(expectedRates)
	if err != nil {
		slog.Error("error marshaling expected rates", "error", err)
		http.Error(writer, ErrIntervalServerError.Error(), http.StatusInternalServerError)
		return
	}

	writer.Header().Set("Content-Type", "application/json") // Set header first

	writer.WriteHeader(http.StatusOK) // Write status code before writing body

	// Write the expected rates JSON response to the writer
	write, err := writer.Write(expectedRatesMarshaled)
	if err != nil {
		slog.Error("error writing response", "error", err)
		http.Error(writer, ErrIntervalServerError.Error(), http.StatusInternalServerError)
		return
	}

	// Check if the number of bytes written is equal to the length of the expected rates JSON byte slice
	if write != len(expectedRatesMarshaled) {
		slog.Error("error writing response", slog.Int("expected", len(expectedRatesMarshaled)), slog.Int("actual", write))
		http.Error(writer, ErrIntervalServerError.Error(), http.StatusInternalServerError)
		return
	}
}

// SubmitShipmentOffer is an HTTP handler that submits a new shipment offer to the system. It expects a JSON payload
// containing the details of the shipment offer. The handler decodes the request body, validates the offer, and submits
// the shipment to the service layer. The handler returns a JSON response with a status of OK if the shipment was
// successfully submitted.
func (h ShipmentHandler) SubmitShipmentOffer(writer http.ResponseWriter, request *http.Request) {
	// Check request headers for Content-Type and validate it is application/json
	if !strings.HasPrefix(request.Header.Get("Content-Type"), "application/json") {
		slog.Warn("invalid content type", "content-type", request.Header.Get("Content-Type"))
		writeJSONResponse(writer, http.StatusUnsupportedMediaType, map[string]string{"error": ErrInvalidContentType.Error()})
		return
	}

	var shipmentOffer requestedShipmentOffer
	// Decode the request body into the requestedShipmentOffer struct
	if err := json.NewDecoder(request.Body).Decode(&shipmentOffer); err != nil {
		slog.Error("error decoding request payload", "error", err)
		writeJSONResponse(writer, http.StatusBadRequest, map[string]string{"error": ErrInvalidRequestPayload.Error()})
		return
	}

	// Defer closing the request body after the function returns
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			slog.Error("error closing request body", "error", err)
		}
	}(request.Body)

	// Validate and parse the shipment offer
	shipment, err := validateAndParseShipment(shipmentOffer)
	if err != nil {
		h.s.IncrementShipmentUnitsCount() // Increment the shipment units count for statistics reporting and average rate calculation
		writeJSONResponse(writer, http.StatusOK, nil)
		return
	}

	// Submit the shipment to the service layer
	if err = h.s.SubmitShipment(&shipment); err != nil {
		slog.Error("error adding shipment offer", "error", err)
		writeJSONResponse(writer, http.StatusInternalServerError, map[string]string{"error": ErrIntervalServerError.Error()})
		return
	}

	writeJSONResponse(writer, http.StatusOK, nil) // Increment the shipment units count for statistics reporting and average rate calculation
}

// validateAndParseShipment validates the requestedShipmentOffer and parses it into a domain.ShipmentUnit struct.
func validateAndParseShipment(shipmentOffer requestedShipmentOffer) (domain.ShipmentUnit, error) {
	switch {
	case shipmentOffer.Company < MinCompanyID || shipmentOffer.Company > MaxCompanyID:
		return domain.ShipmentUnit{}, domain.ErrInvalidCompany
	case shipmentOffer.Price < MinPrice || shipmentOffer.Price > MaxPrice:
		return domain.ShipmentUnit{}, domain.ErrInvalidPrice
	case shipmentOffer.Origin != OriginShanghai && shipmentOffer.Origin != OriginSingapore && shipmentOffer.Origin != OriginShenzhen && shipmentOffer.Origin != OriginNingbo && shipmentOffer.Origin != OriginGuangzhou:
		return domain.ShipmentUnit{}, domain.ErrInvalidOriginPort
	default:
		// continue
	}

	// Parse the date string into a time.Time object, the date should be in the format "YYYY-MM-DD"
	parsedDate, err := time.Parse(dateFormat, shipmentOffer.Date)
	if err != nil {
		return domain.ShipmentUnit{}, domain.ErrInvalidDate
	}

	shipment := domain.ShipmentUnit{
		Origin: shipmentOffer.Origin,
		ShipmentQuote: domain.ShipmentQuote{
			Company: shipmentOffer.Company,
			Price:   shipmentOffer.Price,
			Date:    parsedDate,
		},
	}

	return shipment, nil
}

// writeJSONResponse writes a JSON response to the writer with the specified status code and data.
func writeJSONResponse(writer http.ResponseWriter, status int, data interface{}) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	if data != nil {
		err := json.NewEncoder(writer).Encode(data)
		if err != nil {
			slog.Error("error writing response", "error", err)
			return
		}
	}
}

// CreateShipmentHandler creates a new requestedShipmentOffer handler.
func CreateShipmentHandler(s domain.ShipmentService) *ShipmentHandler {
	return &ShipmentHandler{s: s}
}
