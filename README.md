# Cargoplot

The Cargoplot Pricing Service is designed to aggregate and analyze sea freight shipping quotes, 
enabling clients to **submit quotes** and retrieve the **expected rates** for shipping containers based on 
the cheapest available quotes from multiple freight forwarding companies.

## Features

##### Submit Shipment Quotes

- Accepts JSON payloads with shipping quotes from freight forwarding companies.

- Ensures only the most recent quote for each company-origin combination is retained.

- Aggregates quotes for each origin and dynamically maintains the 10 cheapest rates.

##### Retrieve Expected Rates

- Calculates the average of the 10 cheapest rates for each origin.

- Provides expected rates in a JSON format.


## API Endpoints

The cargoplot service features two main endpoints, one for submitting shipment quotes and another for retrieving 
expected rates.

##### Submit a Shipment Quote

Submit a shipment quote to the service. The quote includes the company identifier, price, origin location, and 
effective date.

- Endpoint: `POST /`
  - Request :
    - Headers: `Content-Type: application/json`
    - Payload:
        ```json
        {
            "company": {int},
            "price": {int},
            "origin": "{string}",
            "date": "{string}"
        }
        ```
    - Body: JSON object with the following fields:
        - `company` (integer): identifier for a company, in range 1-999 (inclusive)
        - `price` (integer): price, in range 1-99999 (inclusive)
        - `origin` (string): 5-character origin location code, one of: `"CNSGH"` (Shanghai), `"SGSIN"` (Singapore),`"CNSNZ"` (Shenzhen), `"CNNBO"` (Ningbo), `"CNGGZ"` (Guangzhou)
        - `date` (string): first date that the given price is in effect, formatted `YYYY-MM-DD`
    - Example:
      ```bash
      curl --location '{host}:{port}' \
          --header 'Content-Type: application/json' \
          --data '{
              "company":1,
              "price":200,
              "origin":"CNSGH",
              "date":"2018-04-10"
          }'
      ```

  
##### Retrieve Expected Rates 

Retrieve the expected rates for all known locations. The expected rate is 
defined as the average of the prices of the 10 cheapest shipping companies for that origin location.

- Endpoint: `GET /`
- Response Headers: `Content-Type: application/json`
- Response Body: JSON object with origin location codes as keys and applicable expected rate as values.

- Response:
  - Content-Type: application/json
  - Payload:
    ```json
        {
            "CNGGZ": {int},
            "CNNBO": {int},
            "CNSGH": {int},
            "CNSNZ": {int},
            "SGSIN": {int}
        }
    ```
    - The above output means:
        - For Guangzhou (`CNGGZ`), the average price for the ten forwarders with lowest rates was $948.
        - For Ningbo (`CNNBO`), the average price for the ten forwarders with lowest rates was $1892.
        - For Shanghai (`CNSGH`), the average price for the ten forwarders with lowest rates was $2615.
        - For Shenzhen (`CNSNZ`), the average price for the ten forwarders with lowest rates was $1618.
        - For Singapore (`SGSIN`), the average price for the ten forwarders with lowest rates was $3029.
  - Example:
  ```bash
      curl --location '{host}:{port}'
  ```

## Data Storage

In-memory data structures for rapid access and processing.
When the service is shut down or restarted, all data are being erased.

## HowTo

First of all, you need to clone the repository to your local machine and navigate to the project directory.

```shell
cd cargoplot
```
Then to run the Cargoplot service you can do that by using `Go(1.23+)` or `Docker`.

### Using Golang

Start by testing the service to ensure everything is working as expected.

```shell
go test ./...
```

Then you can build and run the service using the following command:

```shell
go build -o cargoplot cmd/main.go
./cargoplot
```

You can also run the service using the `go run` command:

```shell
go run cmd/main.go
```

There is also an option to run the service with **environment variables** to specify the HTTP server address:
>Note: The default HTTP server address is `localhost:3142` and the port format requires a semi-colon `:`, e.g., `:3142`.

```shell
HTTP_SERVER_ADDR=your_server_address go run cmd/main.go
```


### Using Docker

You can build and run the service using Docker. First, build the Docker image using the following command:

```shell
docker build -t cargoplot .
```

Then run the Docker container:

```shell
docker run -p 3142:3142 cargoplot
```

Alternatively, you can run the Docker container with environment variables to specify the HTTP server address:

```shell
docker run -e HTTP_SERVER_ADDR=your_server_address -p your_server_address:your_server_address your_image_name
```


## Additional Information

### Design 

The service is implemented following `domain-driven design` principles, with the core logic divided into four main packages:

- **Domain**: Contains the domain declarations, such as structs, interfaces, and errors, which are shared across the other packages.
- **Persistence**: Implements the repository interfaces. In this case, it is responsible for storing and retrieving shipment quotes.
- **Service**: Encapsulates the business logic, including processing shipment quotes and calculating expected rates.
- **Presentation**: Includes HTTP handlers and server configuration. This package configures the server to listen on a 
specified address and port. The handlers parse incoming requests, invoke service methods, and return responses.



### Missing Features

- **Rate-Limiting and Throttling**: The service does not currently implement rate-limiting or throttling. 
  This can be added to prevent abuse and ensure fair usage of the service.
- **Monitoring and Metrics**: The service does not currently provide monitoring or metrics. 
  This can be added to track the performance and health of the service.
- **Benchmarking**: The service does not currently provide benchmarking capabilities. 
  This can be added to measure the performance of the service under different loads.
- **Race Conditions**: Might need a closer look for any kind of race condition.

### Thoughts for applying optimization

- **Testing**: The service includes unit tests for the core logic. 
  Additional tests and in a more structure can be added to cover edge cases and improve test coverage.
- **Performance**: By tweaking the data structures and algorithms, the service can be optimized for performance.
- **Concurrency**: The service can be made more efficient by introducing concurrency.
- **Logging**: Adding structured logging can help in debugging and monitoring the service. At the moment, logging is mostly for critical errors.