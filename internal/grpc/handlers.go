package grpc

import (
	"context"

	pb "github.com/dantte-lp/ocserv-agent/pkg/proto/agent/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// HealthCheck implements the HealthCheck RPC method
func (s *Server) HealthCheck(ctx context.Context, req *pb.HealthCheckRequest) (*pb.HealthCheckResponse, error) {
	s.logger.Debug().
		Int32("tier", req.Tier).
		Msg("HealthCheck called")

	// Validate tier
	if req.Tier < 1 || req.Tier > 3 {
		return nil, status.Errorf(codes.InvalidArgument, "tier must be 1, 2, or 3")
	}

	// Perform health check based on tier
	checks := make(map[string]string)
	healthy := true
	statusMsg := "OK"

	switch req.Tier {
	case 1:
		// Tier 1: Basic heartbeat
		checks["agent"] = "running"
		checks["config"] = "loaded"

	case 2:
		// Tier 2: Deep check (not yet implemented)
		checks["agent"] = "running"
		checks["config"] = "loaded"
		checks["ocserv_process"] = "not_implemented"
		checks["port_listening"] = "not_implemented"
		statusMsg = "Tier 2 checks not fully implemented"

	case 3:
		// Tier 3: Application check (not yet implemented)
		checks["agent"] = "running"
		checks["end_to_end"] = "not_implemented"
		statusMsg = "Tier 3 checks not implemented"
		healthy = false
	}

	return &pb.HealthCheckResponse{
		Healthy:       healthy,
		StatusMessage: statusMsg,
		Checks:        checks,
		Timestamp:     timestamppb.Now(),
	}, nil
}

// ExecuteCommand implements the ExecuteCommand RPC method
func (s *Server) ExecuteCommand(ctx context.Context, req *pb.CommandRequest) (*pb.CommandResponse, error) {
	s.logger.Info().
		Str("request_id", req.RequestId).
		Str("command_type", req.CommandType).
		Strs("args", req.Args).
		Msg("ExecuteCommand called")

	// Execute command through ocserv manager
	result, err := s.ocservManager.ExecuteCommand(ctx, req.CommandType, req.Args)

	response := &pb.CommandResponse{
		RequestId: req.RequestId,
	}

	if err != nil {
		response.Success = false
		response.ErrorMessage = err.Error()
		if result != nil {
			response.Stdout = result.Stdout
			response.Stderr = result.Stderr
			response.ExitCode = int32(result.ExitCode)
		}
		return response, nil
	}

	response.Success = result.Success
	response.Stdout = result.Stdout
	response.Stderr = result.Stderr
	response.ExitCode = int32(result.ExitCode)
	response.ErrorMessage = result.ErrorMsg

	return response, nil
}

// UpdateConfig implements the UpdateConfig RPC method
func (s *Server) UpdateConfig(ctx context.Context, req *pb.ConfigUpdateRequest) (*pb.ConfigUpdateResponse, error) {
	s.logger.Info().
		Str("request_id", req.RequestId).
		Str("config_type", req.ConfigType.String()).
		Str("config_name", req.ConfigName).
		Msg("UpdateConfig called")

	// TODO: Implement config update
	return &pb.ConfigUpdateResponse{
		RequestId:    req.RequestId,
		Success:      false,
		ErrorMessage: "not implemented yet",
	}, nil
}

// StreamLogs implements the StreamLogs RPC method
func (s *Server) StreamLogs(req *pb.LogStreamRequest, stream pb.AgentService_StreamLogsServer) error {
	s.logger.Info().
		Str("log_source", req.LogSource).
		Bool("follow", req.Follow).
		Msg("StreamLogs called")

	// TODO: Implement log streaming
	return status.Error(codes.Unimplemented, "not implemented yet")
}

// AgentStream implements the AgentStream RPC method
func (s *Server) AgentStream(stream pb.AgentService_AgentStreamServer) error {
	s.logger.Info().Msg("AgentStream called")

	// TODO: Implement bidirectional streaming
	return status.Error(codes.Unimplemented, "not implemented yet")
}
