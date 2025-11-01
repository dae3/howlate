package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"
)

var (
	routes        []Route
	trips         []Trip
	searchTmpl *template.Template
)

func main() {
	var err error
	searchTmpl, err = template.New("results").Parse(`
{{range .}}
<div class="pa2 br2 bb b--black-20"
     onclick="selectRoute('{{.ID}}', '{{.ShortName}} - {{.LongName}}')">
    {{.ShortName}} - {{.LongName}}
</div>
{{end}}
`)
	if err != nil {
		log.Fatalf("failed to parse search template: %v", err)
	}
	routes, err = readRoutes("data/routes.txt")
	if err != nil {
		log.Fatalf("failed to read routes: %v", err)
	}

	trips, err = readTrips("data/trips.txt")
	if err != nil {
		log.Fatalf("failed to read trips: %v", err)
	}

	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/search", handleSearch)
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
            <input id="route-search" name="route-search" class="db w-100 pa2 ba b--black-20 br2"
                   type="text" placeholder="Search for a route..."
                   hx-get="/search"
                   hx-trigger="keyup changed delay:500ms"
                   hx-target="#route-results"
                   hx-indicator="#loading-routes">
            <input type="hidden" id="route" name="route"
                   hx-get="/trips"
                   hx-target="#trip"
                   hx-indicator="#loading-trips">
            <div id="route-results"></div>
            <span id="loading-routes" class="htmx-indicator">Searching...</span>
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
            const routeSearchInput = document.getElementById('route-search');
            const routeInput = document.getElementById('route');
            const tripSelect = document.getElementById('trip');
            const routeResults = document.getElementById('route-results');

            // Function to handle route selection
            window.selectRoute = function(id, name) {
                routeInput.value = id;
                routeSearchInput.value = name;
                routeResults.innerHTML = '';
                localStorage.setItem('selectedRouteId', id);
                localStorage.setItem('selectedRouteName', name);
                localStorage.removeItem('selectedTrip');
                tripSelect.innerHTML = '<option value="">Select a Trip</option>';
                htmx.trigger(routeInput, 'change');
            }

            // Load saved values
            const savedRouteId = localStorage.getItem('selectedRouteId');
            const savedRouteName = localStorage.getItem('selectedRouteName');
            if (savedRouteId && savedRouteName) {
                routeInput.value = savedRouteId;
                routeSearchInput.value = savedRouteName;
                htmx.trigger(routeInput, 'change');
            }

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
	tmpl.Execute(w, nil)
}

func handleSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("route")
	var matchingRoutes []Route
	if query != "" {
		for _, route := range routes {
			if strings.Contains(strings.ToLower(route.LongName), strings.ToLower(query)) ||
				strings.Contains(strings.ToLower(route.ShortName), strings.ToLower(query)) {
				matchingRoutes = append(matchingRoutes, route)
			}
		}
	}
	searchTmpl.Execute(w, matchingRoutes)
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
