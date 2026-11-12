package grpcserver

import (
	"context"
	"strconv"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"watcher/backend/internal/biz/internal/model"
	bizsvc "watcher/backend/internal/biz/service"
	bizv1 "watcher/contracts/gen/biz/v1"
)

// Server adapts *bizsvc.Service to generated BizServiceServer (BE-06).
type Server struct {
	bizv1.UnimplementedBizServiceServer
	biz *bizsvc.Service
}

func New(biz *bizsvc.Service) *Server {
	return &Server{biz: biz}
}

func (s *Server) ListAlarms(ctx context.Context, req *bizv1.ListAlarmsRequest) (*bizv1.ListAlarmsResponse, error) {
	page, size := int(req.GetPage()), int(req.GetPageSize())
	list, total, err := s.biz.ListAlarms(ctx, page, size, req.GetStatus(), "")
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	out := &bizv1.ListAlarmsResponse{Total: total}
	for i := range list {
		out.Items = append(out.Items, toPBAlarm(&list[i]))
	}
	return out, nil
}

func (s *Server) AckAlarm(ctx context.Context, req *bizv1.AckAlarmRequest) (*bizv1.AckAlarmResponse, error) {
	id, err := strconv.ParseUint(req.GetAlarmId(), 10, 64)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "alarm_id")
	}
	actor, _ := strconv.ParseUint(req.GetActorId(), 10, 64)
	if err := s.biz.AckAlarm(ctx, id, actor); err != nil {
		return nil, mapErr(err)
	}
	return &bizv1.AckAlarmResponse{}, nil
}

func (s *Server) CloseAlarm(ctx context.Context, req *bizv1.CloseAlarmRequest) (*bizv1.CloseAlarmResponse, error) {
	id, err := strconv.ParseUint(req.GetAlarmId(), 10, 64)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "alarm_id")
	}
	if err := s.biz.CloseAlarm(ctx, id); err != nil {
		return nil, mapErr(err)
	}
	return &bizv1.CloseAlarmResponse{}, nil
}

func (s *Server) EvaluateTelemetry(ctx context.Context, req *bizv1.EvaluateTelemetryRequest) (*bizv1.EvaluateTelemetryResponse, error) {
	raised, err := s.biz.EvaluateTelemetry(ctx, req.GetWarehouseId(), req.GetDeviceId(), req.GetMetrics())
	if err != nil {
		return nil, mapErr(err)
	}
	out := &bizv1.EvaluateTelemetryResponse{}
	for i := range raised {
		out.Raised = append(out.Raised, toPBAlarm(&raised[i]))
	}
	return out, nil
}

func (s *Server) EvaluateLocation(ctx context.Context, req *bizv1.EvaluateLocationRequest) (*bizv1.EvaluateLocationResponse, error) {
	raised, err := s.biz.EvaluateLocation(ctx, req.GetWarehouseId(), req.GetDeviceId(), req.GetLat(), req.GetLng())
	if err != nil {
		return nil, mapErr(err)
	}
	out := &bizv1.EvaluateLocationResponse{}
	for i := range raised {
		out.Raised = append(out.Raised, toPBAlarm(&raised[i]))
	}
	return out, nil
}

func (s *Server) VoidUniqueCode(ctx context.Context, req *bizv1.VoidUniqueCodeRequest) (*bizv1.VoidUniqueCodeResponse, error) {
	actor, _ := strconv.ParseUint(req.GetOperatorId(), 10, 64)
	uc, err := s.biz.VoidUniqueCode(ctx, req.GetCode(), actor, req.GetRemark())
	if err != nil {
		return nil, mapErr(err)
	}
	return &bizv1.VoidUniqueCodeResponse{Code: uc.Code, Status: uc.Status}, nil
}

func toPBAlarm(a *model.Alarm) *bizv1.Alarm {
	if a == nil {
		return nil
	}
	return &bizv1.Alarm{
		Id:          a.ID.String(),
		WarehouseId: a.WarehouseID,
		Title:       a.Title,
		Level:       a.Level,
		Status:      a.Status,
		DeviceId:    a.DeviceID,
		Metric:      a.Metric,
		Value:       a.Value,
	}
}

func mapErr(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "NOT_FOUND"):
		return status.Errorf(codes.NotFound, "%v", err)
	case strings.Contains(msg, "STATE_INVALID"), strings.Contains(msg, "VALIDATION"):
		return status.Errorf(codes.FailedPrecondition, "%v", err)
	default:
		return status.Errorf(codes.Internal, "%v", err)
	}
}
