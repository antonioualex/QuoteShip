package presentation

import (
	"log/slog"
	"net/http"

	"quoteship/domain"
)

// RegisterRoutes registers routes for the requested Shipment service.
func RegisterRoutes(mux *http.ServeMux, s domain.ShipmentService) {
	// Create a new Shipment handler.
	h := CreateShipmentHandler(s)

	// Register the handler functions with the provided ServeMux. The handler functions are registered at the specified
	// routes with the corresponding HTTP methods.
	mux.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		switch request.Method {
		case http.MethodGet:
			// Call the GetLatestExpectedRates handler function when a GET request is received at the root route.
			h.GetLatestExpectedRates(writer, request)
		case http.MethodPost:
			// Call the SubmitShipmentOffer handler function when a POST request is received at the root route.
			h.SubmitShipmentOffer(writer, request)
		default:
			http.Error(writer, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	})

	slog.Info("Creating routes for requestedShipmentOffer service...")
	slog.Info("Registered GetLatestExpectedRates handler at / using GET method")
	slog.Info("Registered SubmitShipmentOffer handler at / using POST method")
	slog.Info("Created routes for requestedShipmentOffer service")
}
