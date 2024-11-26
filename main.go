package main

import (
	"context"
	//"database/sql"
	"fmt"
	"html/template"
	"log"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
)

type Airport struct {
	ID   string `field:"airport_code"`
	Name string `field:"airport_name"`
}

type Flight struct {
	ID                int    `field:"flight_id"`
	Departure_airport string `field:"departure_airport"`
	Arrival_airport   string `field:"arrival_airport"`
	Status            string `field:"status"`
}

type Airplane struct {
	ID string `field:"aircraft_code"`
}

type Seatssum struct {
	Result int `field:"result"`
}

var db *pgx.Conn

func init() {
	var err error
	db, err = pgx.Connect(context.Background(), "postgres://student:student@192.168.1.41:5432/demo")
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
}

func main() {
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/airports", airportsHandler)
	http.HandleFunc("/flights/", flightsHandler)
	http.HandleFunc("/calculate", calculateSeatsHandler)

	log.Println("Starting server on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, `<h1>Welcome</h1><a href="/airports">View Airports</a>`)
}

func airportsHandler(w http.ResponseWriter, r *http.Request) {
	airports, err := getAirports()
	if err != nil {
		http.Error(w, "Error fetching airports", http.StatusInternalServerError)
		return
	}
	tmpl := template.Must(template.ParseFiles("templates/airports.html"))
	tmpl.Execute(w, airports)
}

func flightsHandler(w http.ResponseWriter, r *http.Request) {
	airportID := r.URL.Path[len("/flights/"):]
	flights, err := getFlightsForAirport(airportID)
	if err != nil {
		http.Error(w, "Error fetching flights", http.StatusInternalServerError)
		return
	}
	tmpl := template.Must(template.ParseFiles("templates/flights.html"))
	tmpl.Execute(w, flights)
}

func calculateSeatsHandler(w http.ResponseWriter, r *http.Request) {
	airplanes, err := getAirplanes()
	if err != nil {
		http.Error(w, "Error fetching airplanes", http.StatusInternalServerError)
		return
	}
	var wg sync.WaitGroup
	results := make([]string, len(airplanes))

	start := time.Now()

	for i, airplaneID := range airplanes {
		wg.Add(1)
		go func(i int, airplaneID string) {
			defer wg.Done()
			time.Sleep(2 * time.Second)
			seatCount, err := calculateSeatSum(airplaneID)
			if err != nil {
				http.Error(w, "Error calculating seats", http.StatusInternalServerError)
				return
			}
			results[i] = fmt.Sprintf("Airplane %s: %d", airplaneID, seatCount)
		}(i, airplaneID)
	}

	wg.Wait()
	duration := time.Since(start)
	for _, result := range results {
		fmt.Fprintln(w, result)
	}
	fmt.Fprintln(w, "Total calculation time:", duration)
}

func getAirports() ([]Airport, error) {
	var airports []Airport
	rows, err := db.Query(context.Background(), "SELECT airport_code, airport_name FROM bookings.airports")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var airport Airport
		if err := rows.Scan(&airport.ID, &airport.Name); err != nil {
			return nil, err
		}
		airports = append(airports, airport)
	}
	return airports, nil
}

func getFlightsForAirport(airportID string) ([]Flight, error) {
	var flights []Flight
	rows, err := db.Query(context.Background(), "SELECT flight_id, departure_airport, arrival_airport, status FROM bookings.flights WHERE departure_airport=$1 OR arrival_airport=$1", airportID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var flight Flight
		if err := rows.Scan(&flight.ID, &flight.Departure_airport, &flight.Arrival_airport, &flight.Status); err != nil {
			return nil, err
		}
		flights = append(flights, flight)
	}
	return flights, nil
}

func getAirplanes() ([]string, error) {
	var airplanes []string
	rows, err := db.Query(context.Background(), "SELECT aircraft_code FROM bookings.aircrafts_data")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var airplane Airplane
		if err := rows.Scan(&airplane.ID); err != nil {
			return nil, err
		}
		airplanes = append(airplanes, airplane.ID)
	}
	return airplanes, nil
}

func calculateSeatSum(airplaneID string) (int, error) {
	row := db.QueryRow(context.Background(), "SELECT COUNT(*) AS result FROM bookings.seats WHERE aircraft_code = $1;", airplaneID)

	var seatsum Seatssum
	if err := row.Scan(&seatsum.Result); err != nil {
		return 0, err
	}

	result64 := math.Pow(float64(seatsum.Result), 2.0)
	result := int(result64)
	return result, nil
}
