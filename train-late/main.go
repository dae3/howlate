package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
)

var (
	routes []Route
	trips  []Trip
)

func main() {
	var err error
	routes, err = readRoutes("data/routes.txt")
	if err != nil {
		log.Fatalf("failed to read routes: %v", err)
	}

	trips, err = readTrips("data/trips.txt")
	if err != nil {
		log.Fatalf("failed to read trips: %v", err)
	}

	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/trips", handleTrips)
	http.HandleFunc("/lateness", handleLateness)

	fmt.Println("Server starting on port 8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.New("index").Parse(`
<!DOCTYPE html>
<html>
<head>
    <title>Train Lateness</title>
    <link rel="stylesheet" href="https://unpkg.com/tachyons@4.12.0/css/tachyons.min.css"/>
    <script src="https://unpkg.com/htmx.org@1.9.10"></script>
</head>
<body class="bg-light-blue sans-serif">
    <div class="mw7 center pa4 mt4 bg-white shadow-5 br3">
        <h1 class="f2 tc">Train Lateness</h1>
        <div class="mb3">
            <label for="route" class="f6 b db mb2">Route</label>
            <select id="route" name="route" class="db w-100 pa2 ba b--black-20 br2"
                hx-get="/trips"
                hx-target="#trip"
                hx-indicator="#loading-trips">
                <option value="">Select a Route</option>
                {{range .}}
                <option value="{{.ID}}">{{.ShortName}} - {{.LongName}}</option>
                {{end}}
            </select>
        </div>
        <div class="mb3">
            <label for="trip" class="f6 b db mb2">Trip</label>
            <select id="trip" name="trip" class="db w-100 pa2 ba b--black-20 br2"
                hx-get="/lateness"
                hx-target="#lateness-display"
                hx-indicator="#loading-lateness">
                <option value="">Select a Trip</option>
            </select>
            <span id="loading-trips" class="htmx-indicator">Loading trips...</span>
        </div>
        <div id="lateness-display" class="tc">
            <!-- Lateness will be displayed here -->
        </div>
        <span id="loading-lateness" class="htmx-indicator">Loading lateness...</span>
    </div>

    <script>
        document.addEventListener('DOMContentLoaded', (event) => {
            const routeSelect = document.getElementById('route');
            const tripSelect = document.getElementById('trip');

            // Load saved values
            const savedRoute = localStorage.getItem('selectedRoute');
            if (savedRoute) {
                routeSelect.value = savedRoute;
                htmx.trigger(routeSelect, 'change'); // Trigger HTMX to load trips
            }

            // Save values on change
            routeSelect.addEventListener('change', () => {
                localStorage.setItem('selectedRoute', routeSelect.value);
                localStorage.removeItem('selectedTrip'); // Clear trip selection
                tripSelect.innerHTML = '<option value="">Select a Trip</option>'; // Reset trip dropdown
            });

            tripSelect.addEventListener('change', () => {
                localStorage.setItem('selectedTrip', tripSelect.value);
            });

            // Restore trip selection after trips have been loaded
            htmx.on('htmx:afterSwap', function(evt) {
                if (evt.detail.target.id === 'trip') {
                    const savedTrip = localStorage.getItem('selectedTrip');
                    if (savedTrip) {
                        tripSelect.value = savedTrip;
                        htmx.trigger(tripSelect, 'change');
                    }
                }
            });
        });
    </script>
</body>
</html>
`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, routes)
}

func handleTrips(w http.ResponseWriter, r *http.Request) {
	routeID := r.URL.Query().Get("route")
	var routeTrips []Trip
	for _, trip := range trips {
		if trip.RouteID == routeID {
			routeTrips = append(routeTrips, trip)
		}
	}

	tmpl, err := template.New("trips").Parse(`
{{range .}}
<option value="{{.ID}}">{{.ID}}</option>
{{end}}
`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, routeTrips)
}

func handleLateness(w http.ResponseWriter, r *http.Request) {
	// NOTE: The real-time trip update API provides data in a binary format
	// using protocol buffers. To parse this data, you would need the .proto
	// file from the Transport for NSW open data portal and a library like
	// protoc-gen-go.
	//
	// For the purpose of this demo, we'll continue to return a static value.
	tripID := r.URL.Query().Get("trip")
	if tripID == "" {
		http.Error(w, "tripID is required", http.StatusBadRequest)
		return
	}

	// In a real application, you would fetch the data from the API and
	// decode it using the .proto file. The logic would look something
	// like this:
	//
	// 1. Fetch the data from the API endpoint for the selected trip.
	// 2. Unmarshal the protocol buffer data into a Go struct.
	// 3. Extract the delay information from the struct.
	// 4. Return the delay to the user.

	fmt.Fprintf(w, `<div class="f-headline tc">5</div><div class="tc">minutes late</div>`)
}
