package service

import (
	"context"
	"testing"

	"watcher/backend/internal/biz/internal/model"
)

func TestInboundOutboundStockFlow(t *testing.T) {
	db := openTestDB(t)
	s := New(db)
	ctx := context.Background()

	mat, err := s.CreateMaterial(ctx, CreateMaterialInput{Code: "M-FLOW", Name: "测试物资", Unit: "只"})
	if err != nil {
		t.Fatal(err)
	}
	order, err := s.CreateInboundOrder(ctx, CreateInboundInput{
		WarehouseID: "WH01", MaterialID: mat.ID, Qty: 2, CreatedBy: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	codes, err := s.IssueCodes(ctx, order.ID, 2)
	if err != nil || len(codes) != 2 {
		t.Fatalf("issue=%v len=%d", err, len(codes))
	}
	for i, c := range codes {
		_, err := s.ConfirmInbound(ctx, ConfirmInboundInput{
			OrderID: order.ID, UniqueCode: c.Code, OperatorID: 1,
			ClientRequestID: "in-" + c.Code,
		})
		if err != nil {
			t.Fatalf("confirm[%d]: %v", i, err)
		}
	}

	stocks, err := s.ListStocks(ctx, "WH01")
	if err != nil {
		t.Fatal(err)
	}
	var bal *model.StockBalance
	for i := range stocks {
		if stocks[i].MaterialID == mat.ID {
			bal = &stocks[i]
			break
		}
	}
	if bal == nil || bal.Qty != 2 {
		t.Fatalf("stock after inbound=%+v", bal)
	}

	out, err := s.CreateOutboundOrder(ctx, "WH01", 1, mat.ID, 1, "smoke")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.LockOutbound(ctx, LockOutboundInput{
		OrderID: out.ID, UniqueCode: codes[0].Code, OperatorID: 1, ClientRequestID: "lock-1",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ReceiveOutbound(ctx, ReceiveOutboundInput{
		OrderID: out.ID, UniqueCode: codes[0].Code, OperatorID: 1, ClientRequestID: "recv-1",
	}); err != nil {
		t.Fatal(err)
	}

	stocks, _ = s.ListStocks(ctx, "WH01")
	for i := range stocks {
		if stocks[i].MaterialID == mat.ID {
			if stocks[i].Qty != 1 {
				t.Fatalf("qty after receive want 1 got %v locked=%v", stocks[i].Qty, stocks[i].LockedQty)
			}
			if stocks[i].LockedQty != 0 {
				t.Fatalf("locked should be 0 got %v", stocks[i].LockedQty)
			}
		}
	}

	code, err := s.GetUniqueCode(ctx, codes[0].Code)
	if err != nil {
		t.Fatal(err)
	}
	if code.Status != model.CodeSigned {
		t.Fatalf("code status=%s", code.Status)
	}

	// outbox rows should exist for inbound/outbound
	var n int64
	db.Model(&model.IntegrationOutbox{}).Count(&n)
	if n < 2 {
		t.Fatalf("expected outbox rows, got %d", n)
	}
}

func TestApprovalStockAdjustTwoLevel(t *testing.T) {
	db := openTestDB(t)
	s := New(db)
	ctx := context.Background()

	mat, err := s.CreateMaterial(ctx, CreateMaterialInput{Code: "M-ADJ", Name: "调整物资", Unit: "kg"})
	if err != nil {
		t.Fatal(err)
	}
	// seed stock via inbound
	in, _ := s.CreateInboundOrder(ctx, CreateInboundInput{WarehouseID: "WH01", MaterialID: mat.ID, Qty: 1, CreatedBy: 1})
	codes, _ := s.IssueCodes(ctx, in.ID, 1)
	_, _ = s.ConfirmInbound(ctx, ConfirmInboundInput{OrderID: in.ID, UniqueCode: codes[0].Code, OperatorID: 1, ClientRequestID: "seed-1"})

	adj, order, err := s.CreateStockAdjustment(ctx, "WH01", mat.ID, 3, "test", 1)
	if err != nil {
		t.Fatal(err)
	}
	if adj.Status != model.AdjustPending || order == nil {
		t.Fatalf("adj=%+v order=%+v", adj, order)
	}
	hasAdmin := func(code string) bool { return code == "admin" }
	o1, err := s.ActApproval(ctx, ActApprovalInput{OrderID: order.ID, ActorID: 1, Action: "pass", Comment: "L1", HasRole: hasAdmin})
	if err != nil {
		t.Fatal(err)
	}
	if o1.Status != model.ApprovalPending || o1.CurrentLevel != 2 {
		t.Fatalf("after L1: %+v", o1)
	}
	o2, err := s.ActApproval(ctx, ActApprovalInput{OrderID: order.ID, ActorID: 1, Action: "pass", Comment: "L2", HasRole: hasAdmin})
	if err != nil {
		t.Fatal(err)
	}
	if o2.Status != model.ApprovalApproved {
		t.Fatalf("after L2: %+v", o2)
	}
	stocks, _ := s.ListStocks(ctx, "WH01")
	for i := range stocks {
		if stocks[i].MaterialID == mat.ID && stocks[i].Qty != 4 {
			t.Fatalf("qty want 4 got %v", stocks[i].Qty)
		}
	}
}

func TestIdempotentInboundConfirm(t *testing.T) {
	db := openTestDB(t)
	s := New(db)
	ctx := context.Background()
	mat, _ := s.CreateMaterial(ctx, CreateMaterialInput{Code: "M-IDEM", Name: "幂等", Unit: "只"})
	in, _ := s.CreateInboundOrder(ctx, CreateInboundInput{WarehouseID: "WH01", MaterialID: mat.ID, Qty: 1, CreatedBy: 1})
	codes, _ := s.IssueCodes(ctx, in.ID, 1)
	r1, err := s.ConfirmInbound(ctx, ConfirmInboundInput{OrderID: in.ID, UniqueCode: codes[0].Code, OperatorID: 1, ClientRequestID: "same-key"})
	if err != nil {
		t.Fatal(err)
	}
	r2, err := s.ConfirmInbound(ctx, ConfirmInboundInput{OrderID: in.ID, UniqueCode: codes[0].Code, OperatorID: 1, ClientRequestID: "same-key"})
	if err != nil {
		t.Fatal(err)
	}
	if !r2.Replayed {
		t.Fatal("second confirm should replay")
	}
	if r1.UniqueCode != r2.UniqueCode {
		t.Fatal("mismatch")
	}
	stocks, _ := s.ListStocks(ctx, "WH01")
	for i := range stocks {
		if stocks[i].MaterialID == mat.ID && stocks[i].Qty != 1 {
			t.Fatalf("qty should stay 1 got %v", stocks[i].Qty)
		}
	}
}
