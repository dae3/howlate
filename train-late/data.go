package main

import (
	"encoding/csv"
	"io"
	"os"
)

type Route struct {
	ID        string
	ShortName string
	LongName  string
}

type Trip struct {
	RouteID string
	ID      string
}

func readRoutes(path string) ([]Route, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Read() // skip header

	var routes []Route
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if record[1] == "x0001" || record[1] == "X0000" { // trains only
			routes = append(routes, Route{
				ID:        record[0],
				ShortName: record[2],
				LongName:  record[3],
			})
		}
	}

	return routes, nil
}

func readTrips(path string) ([]Trip, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Read() // skip header

	var trips []Trip
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if true {
			trips = append(trips, Trip{
				RouteID: record[0],
				ID:      record[2],
			})
		}
	}

	return trips, nil
}
