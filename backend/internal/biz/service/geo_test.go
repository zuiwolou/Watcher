package service

import (
	"context"
	"testing"
)

func TestPointInPolygon(t *testing.T) {
	sq := [][2]float64{
		{0, 0}, {0, 2}, {2, 2}, {2, 0},
	}
	if !pointInPolygon(1, 1, sq) {
		t.Fatal("center should be inside")
	}
	if pointInPolygon(3, 3, sq) {
		t.Fatal("outside point")
	}
	if pointInPolygon(1, 1, sq[:2]) {
		t.Fatal("<3 points should be outside")
	}
}

func TestUpsertGeofenceValidation(t *testing.T) {
	db := openTestDB(t)
	s := New(db)
	ctx := context.Background()
	_, err := s.UpsertGeofence(ctx, 0, "WH01", "bad", "circle", `{"lat":1,"lng":2,"radiusM":0}`, 0, true)
	if err == nil {
		t.Fatal("expected invalid circle")
	}
	g, err := s.UpsertGeofence(ctx, 0, "WH01", "ok", "circle", `{"type":"circle","lat":31.2,"lng":121.5,"radiusM":100}`, 5, true)
	if err != nil {
		t.Fatal(err)
	}
	if g.ID == 0 || !g.Enabled {
		t.Fatalf("%+v", g)
	}
}
