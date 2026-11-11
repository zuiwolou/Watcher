package service

import (
	"context"
	"strings"
	"testing"
)

func TestSubmitCheckinRequiresPhotoOnDepartArrive(t *testing.T) {
	db := openTestDB(t)
	s := New(db)
	ctx := context.Background()
	_ = s.SeedWarehouse("WH01", "WH01", "一号仓")

	mat, err := s.CreateMaterial(ctx, CreateMaterialInput{Code: "M-T", Name: "运输物资", Unit: "箱"})
	if err != nil {
		t.Fatal(err)
	}
	out, err := s.CreateOutboundOrder(ctx, "WH01", 1, mat.ID, 1, "t")
	if err != nil {
		t.Fatal(err)
	}
	unit, err := s.CreateTransportUnit(ctx, "沪A12345", "truck", "")
	if err != nil {
		t.Fatal(err)
	}
	task, err := s.CreateTransportTask(ctx, CreateTransportTaskInput{
		OutboundOrderID: out.ID, UnitID: unit.ID, DriverUserID: 1, WarehouseID: "WH01", CreatedBy: 1,
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = s.SubmitCheckin(ctx, SubmitCheckinInput{
		TaskID: task.ID, NodeType: "depart", OperatorID: 1, ClientRequestID: "dep-nophoto",
	})
	if err == nil || !strings.Contains(err.Error(), "photo") {
		t.Fatalf("depart without photo should fail, got %v", err)
	}

	lat, lng := 31.23, 121.47
	ck, err := s.SubmitCheckin(ctx, SubmitCheckinInput{
		TaskID: task.ID, NodeType: "depart", OperatorID: 1, ClientRequestID: "dep-ok",
		Lat: &lat, Lng: &lng, PhotoKey: "tr/dep",
	})
	if err != nil {
		t.Fatal(err)
	}
	if ck.PhotoKey != "tr/dep" {
		t.Fatalf("photo=%s", ck.PhotoKey)
	}

	_, err = s.SubmitCheckin(ctx, SubmitCheckinInput{
		TaskID: task.ID, NodeType: "arrive", OperatorID: 1, ClientRequestID: "arr-nophoto",
		Lat: &lat, Lng: &lng,
	})
	if err == nil || !strings.Contains(err.Error(), "photo") {
		t.Fatalf("arrive without photo should fail, got %v", err)
	}
}

func TestReceiveOutboundRequiresPhoto(t *testing.T) {
	db := openTestDB(t)
	s := New(db)
	ctx := context.Background()
	_ = s.SeedWarehouse("WH01", "WH01", "一号仓")

	mat, err := s.CreateMaterial(ctx, CreateMaterialInput{Code: "M-R", Name: "签收物资", Unit: "箱"})
	if err != nil {
		t.Fatal(err)
	}
	order, err := s.CreateInboundOrder(ctx, CreateInboundInput{
		WarehouseID: "WH01", MaterialID: mat.ID, Qty: 1, CreatedBy: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	codes, err := s.IssueCodes(ctx, order.ID, 1)
	if err != nil || len(codes) != 1 {
		t.Fatalf("issue=%v", err)
	}
	if _, err := s.ConfirmInbound(ctx, ConfirmInboundInput{
		OrderID: order.ID, UniqueCode: codes[0].Code, OperatorID: 1, ClientRequestID: "in-r",
	}); err != nil {
		t.Fatal(err)
	}
	out, err := s.CreateOutboundOrder(ctx, "WH01", 1, mat.ID, 1, "recv")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.LockOutbound(ctx, LockOutboundInput{
		OrderID: out.ID, UniqueCode: codes[0].Code, OperatorID: 1, ClientRequestID: "lock-r",
	}); err != nil {
		t.Fatal(err)
	}
	_, err = s.ReceiveOutbound(ctx, ReceiveOutboundInput{
		OrderID: out.ID, UniqueCode: codes[0].Code, OperatorID: 1, ClientRequestID: "recv-nophoto",
	})
	if err == nil || !strings.Contains(err.Error(), "photo") {
		t.Fatalf("receive without photo should fail, got %v", err)
	}
}
