package service

import (
	"context"
	"testing"
	"time"

	"watcher/backend/internal/biz/internal/model"
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
	_, err := s.UpsertGeofence(ctx, 0, "WH01", "bad", "circle", `{"lat":1,"lng":2,"radiusM":0}`, 0, true, "")
	if err == nil {
		t.Fatal("expected invalid circle")
	}
	g, err := s.UpsertGeofence(ctx, 0, "WH01", "ok", "circle", `{"type":"circle","lat":31.2,"lng":121.5,"radiusM":100}`, 5, true, "")
	if err != nil {
		t.Fatal(err)
	}
	if g.ID == 0 || !g.Enabled {
		t.Fatalf("%+v", g)
	}
}

func TestEvaluateLocationRaisesAndSuppresses(t *testing.T) {
	db := openTestDB(t)
	s := New(db)
	ctx := context.Background()
	_, err := s.UpsertGeofence(ctx, 0, "WH01", "yard", "circle",
		`{"type":"circle","lat":31.2304,"lng":121.4737,"radiusM":50}`, 0, true, "")
	if err != nil {
		t.Fatal(err)
	}
	// Far outside Shanghai CBD circle (~10km)
	alarms, err := s.EvaluateLocation(ctx, "WH01", "dev-1", 31.0, 121.0)
	if err != nil {
		t.Fatal(err)
	}
	if len(alarms) != 1 {
		t.Fatalf("want 1 alarm got %d", len(alarms))
	}
	if alarms[0].Status != model.AlarmRaised {
		t.Fatalf("status=%s", alarms[0].Status)
	}
	again, err := s.EvaluateLocation(ctx, "WH01", "dev-1", 31.0, 121.0)
	if err != nil {
		t.Fatal(err)
	}
	if len(again) != 0 {
		t.Fatalf("suppress window should skip, got %d", len(again))
	}
	inside, err := s.EvaluateLocation(ctx, "WH01", "dev-2", 31.2304, 121.4737)
	if err != nil {
		t.Fatal(err)
	}
	if len(inside) != 0 {
		t.Fatalf("inside should not raise, got %d", len(inside))
	}
}

func TestPolygonHysteresisAndSchedule(t *testing.T) {
	db := openTestDB(t)
	s := New(db)
	ctx := context.Background()

	// ~111m per degree lat; square ~200m side around origin-like coords near Shanghai
	poly := `{"type":"polygon","points":[[31.2300,121.4700],[31.2300,121.4720],[31.2320,121.4720],[31.2320,121.4700]]}`
	_, err := s.UpsertGeofence(ctx, 0, "WH01", "poly", "polygon", poly, 80, true, "")
	if err != nil {
		t.Fatal(err)
	}
	// Just outside south edge (~ few meters); with 80m hysteresis should NOT alarm
	near := 31.2297
	alarms, err := s.EvaluateLocation(ctx, "WH01", "dev-poly-near", near, 121.4710)
	if err != nil {
		t.Fatal(err)
	}
	if len(alarms) != 0 {
		t.Fatalf("hysteresis should keep near-outside silent, got %d", len(alarms))
	}
	// Far outside should alarm
	farAlarms, err := s.EvaluateLocation(ctx, "WH01", "dev-poly-far", 31.20, 121.40)
	if err != nil {
		t.Fatal(err)
	}
	if len(farAlarms) != 1 {
		t.Fatalf("far outside want 1 alarm got %d", len(farAlarms))
	}

	// Schedule: only Sunday 00:00-00:01 — almost never active; use empty windows check via helper
	if !geofenceActiveNow("", time.Now()) {
		t.Fatal("empty schedule should be active")
	}
	inactive := `{"windows":[{"dow":[0],"start":"00:00","end":"00:01"}]}`
	// Force a Monday noon
	mon := time.Date(2026, 11, 9, 12, 0, 0, 0, time.Local) // 2026-11-09 is Monday
	if mon.Weekday() != time.Monday {
		t.Fatalf("fixture weekday=%v", mon.Weekday())
	}
	if geofenceActiveNow(inactive, mon) {
		t.Fatal("Monday noon should be outside Sunday-only window")
	}
	g2, err := s.UpsertGeofence(ctx, 0, "WH02", "sched", "circle",
		`{"type":"circle","lat":31.23,"lng":121.47,"radiusM":10}`, 0, true, inactive)
	if err != nil {
		t.Fatal(err)
	}
	if g2.ScheduleJSON == "" {
		t.Fatal("schedule should persist")
	}
}
