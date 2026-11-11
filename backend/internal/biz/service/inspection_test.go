package service

import (
	"context"
	"strings"
	"testing"
)

func TestInspectionCheckinForceOrderAndRadius(t *testing.T) {
	db := openTestDB(t)
	s := New(db)
	ctx := context.Background()

	plan, err := s.CreateInspectionPlan(ctx, "日巡", "0 0 * * *")
	if err != nil {
		t.Fatal(err)
	}
	route, err := s.CreateInspectionRoute(ctx, plan.ID, "路线A", true)
	if err != nil {
		t.Fatal(err)
	}
	p1, err := s.AddInspectionPoint(ctx, route.ID, 1, "点1", 31.23, 121.47, 50, false)
	if err != nil {
		t.Fatal(err)
	}
	p2, err := s.AddInspectionPoint(ctx, route.ID, 2, "点2", 31.231, 121.471, 50, true)
	if err != nil {
		t.Fatal(err)
	}
	task, err := s.CreateInspectionTask(ctx, plan.ID, route.ID, 1, nil)
	if err != nil {
		t.Fatal(err)
	}

	// 跳点应失败
	lat, lng := 31.23, 121.47
	_, _, err = s.InspectionCheckin(ctx, InspectionCheckinInput{
		TaskID: task.ID, PointID: p2.ID, Status: "done",
		Lat: &lat, Lng: &lng, ClientRequestID: "skip-1",
	})
	if err == nil || !strings.Contains(err.Error(), "must check in") {
		t.Fatalf("want force-order error, got %v", err)
	}

	// 无定位应失败（有半径）
	_, _, err = s.InspectionCheckin(ctx, InspectionCheckinInput{
		TaskID: task.ID, PointID: p1.ID, Status: "done", ClientRequestID: "noloc-1",
	})
	if err == nil || !strings.Contains(err.Error(), "lat/lng") {
		t.Fatalf("want lat/lng required, got %v", err)
	}

	// 按序打点1
	_, _, err = s.InspectionCheckin(ctx, InspectionCheckinInput{
		TaskID: task.ID, PointID: p1.ID, Status: "done",
		Lat: &lat, Lng: &lng, ClientRequestID: "ok-1",
	})
	if err != nil {
		t.Fatal(err)
	}

	// 点2 缺照片
	lat2, lng2 := 31.231, 121.471
	_, _, err = s.InspectionCheckin(ctx, InspectionCheckinInput{
		TaskID: task.ID, PointID: p2.ID, Status: "done",
		Lat: &lat2, Lng: &lng2, ClientRequestID: "nophoto",
	})
	if err == nil || !strings.Contains(err.Error(), "photo") {
		t.Fatalf("want photo required, got %v", err)
	}

	_, alm, err := s.InspectionCheckin(ctx, InspectionCheckinInput{
		TaskID: task.ID, PointID: p2.ID, Status: "abnormal",
		Lat: &lat2, Lng: &lng2, PhotoKey: "insp/p2", Remark: "破损", ClientRequestID: "ok-2",
	})
	if err != nil {
		t.Fatal(err)
	}
	if alm == nil || alm.RuleCode != "inspection_abnormal" {
		t.Fatalf("want abnormal alarm, got %+v", alm)
	}
}
