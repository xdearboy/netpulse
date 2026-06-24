package tests

import (
	"testing"

	"netpulse/internal/services/sources"
)

func TestParseASNumber(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"AS15169", 15169},
		{"AS15169 Google LLC", 15169},
		{"as174", 174},
		{"15169", 15169},
		{"", 0},
		{"invalid", 0},
		{"AS0", 0},
	}
	for _, tt := range tests {
		got := sources.ParseASNumber(tt.input)
		if got != tt.want {
			t.Errorf("ParseASNumber(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestParseLatLon(t *testing.T) {
	lat, lon := sources.ParseLatLon("37.4056,-122.0775")
	if lat < 37.4 || lat > 37.5 {
		t.Errorf("lat = %f, want ~37.4", lat)
	}
	if lon < -122.1 || lon > -122.0 {
		t.Errorf("lon = %f, want ~-122.07", lon)
	}

	lat, lon = sources.ParseLatLon("invalid")
	if lat != 0 || lon != 0 {
		t.Errorf("expected 0,0 for invalid input, got %f,%f", lat, lon)
	}
}

func TestParseOrgString(t *testing.T) {
	asn, name := sources.ParseOrgString("AS15169 Google LLC")
	if asn != 15169 {
		t.Errorf("asn = %d, want 15169", asn)
	}
	if name != "Google LLC" {
		t.Errorf("name = %q, want %q", name, "Google LLC")
	}

	asn, name = sources.ParseOrgString("Some Org")
	if asn != 0 {
		t.Errorf("asn = %d, want 0", asn)
	}
	if name != "Some Org" {
		t.Errorf("name = %q, want %q", name, "Some Org")
	}
}

func TestParseFloat(t *testing.T) {
	if f := sources.ParseFloat("37.4056"); f < 37.4 || f > 37.5 {
		t.Errorf("ParseFloat = %f", f)
	}
	if f := sources.ParseFloat("invalid"); f != 0 {
		t.Errorf("ParseFloat(invalid) = %f, want 0", f)
	}
}
