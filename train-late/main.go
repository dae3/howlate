package main

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"google.golang.org/protobuf/proto"
	"train-late/gtfs-realtime"
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
	tripID := r.URL.Query().Get("trip")
	if tripID == "" {
		http.Error(w, "tripID is required", http.StatusBadRequest)
		return
	}

	apiKey := os.Getenv("TFNWS_API_KEY")
	if apiKey == "" {
		log.Println("TFNWS_API_KEY not set")
		http.Error(w, "API key not configured", http.StatusInternalServerError)
		return
	}

	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://api.transport.nsw.gov.au/v1/gtfs/realtime/nswtrains", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	req.Header.Set("Authorization", "apikey "+apiKey)

	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	feed := &gtfs_realtime.FeedMessage{}
	if err := proto.Unmarshal(body, feed); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for _, entity := range feed.Entity {
		if entity.TripUpdate != nil &&
			entity.TripUpdate.Trip != nil &&
			entity.TripUpdate.Trip.TripId != nil &&
			*entity.TripUpdate.Trip.TripId == tripID {
			if len(entity.TripUpdate.StopTimeUpdate) > 0 &&
				entity.TripUpdate.StopTimeUpdate[0].Departure != nil &&
				entity.TripUpdate.StopTimeUpdate[0].Departure.Delay != nil {
				delay := *entity.TripUpdate.StopTimeUpdate[0].Departure.Delay
				minutesLate := delay / 60
				fmt.Fprintf(w, `<div class="f-headline tc">%d</div><div class="tc">minutes late</div>`, minutesLate)
				return
			}
		}
	}

	fmt.Fprintf(w, `<div class="f2 tc">No delay information found for this trip.</div>`)
}
