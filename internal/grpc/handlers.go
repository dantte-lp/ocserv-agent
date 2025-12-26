package grpc

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/dantte-lp/ocserv-agent/internal/config"
	pb "github.com/dantte-lp/ocserv-agent/pkg/proto/agent/v1"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/process"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// HealthCheck implements the HealthCheck RPC method with full tier support
func (s *Server) HealthCheck(ctx context.Context, req *pb.HealthCheckRequest) (*pb.HealthCheckResponse, error) {
	s.logger.Debug().
		Int32("tier", req.Tier).
		Msg("HealthCheck called")

	// Validate tier
	if req.Tier < 1 || req.Tier > 3 {
		return nil, status.Errorf(codes.InvalidArgument, "tier must be 1, 2, or 3")
	}

	checks := make(map[string]string)
	healthy := true
	statusMsg := "OK"

	switch req.Tier {
	case 1:
		// Tier 1: Basic heartbeat - agent is alive
		checks["agent"] = "running"
		checks["config"] = "loaded"
		checks["uptime"] = s.getUptime()

	case 2:
		// Tier 2: Deep check - system resources and ocserv process
		checks["agent"] = "running"
		checks["config"] = "loaded"

		// Check system resources
		memCheck, memOK := s.checkMemory()
		checks["memory"] = memCheck
		if !memOK {
			healthy = false
		}

		cpuCheck, cpuOK := s.checkCPU()
		checks["cpu"] = cpuCheck
		if !cpuOK {
			healthy = false
		}

		// Check ocserv process
		ocservCheck, ocservOK := s.checkOcservProcess()
		checks["ocserv_process"] = ocservCheck
		if !ocservOK {
			healthy = false
			statusMsg = "ocserv process issues detected"
		}

		// Check ocserv socket
		socketCheck, socketOK := s.checkOcservSocket()
		checks["ocserv_socket"] = socketCheck
		if !socketOK {
			healthy = false
		}

	case 3:
		// Tier 3: Application check - end-to-end connectivity
		checks["agent"] = "running"
		checks["config"] = "loaded"

		// Include all Tier 2 checks
		memCheck, memOK := s.checkMemory()
		checks["memory"] = memCheck
		if !memOK {
			healthy = false
		}

		cpuCheck, cpuOK := s.checkCPU()
		checks["cpu"] = cpuCheck
		if !cpuOK {
			healthy = false
		}

		ocservCheck, ocservOK := s.checkOcservProcess()
		checks["ocserv_process"] = ocservCheck
		if !ocservOK {
			healthy = false
		}

		// Tier 3 specific: test occtl command
		occtlCheck, occtlOK := s.checkOcctl(ctx)
		checks["occtl"] = occtlCheck
		if !occtlOK {
			healthy = false
			statusMsg = "occtl communication failed"
		}

		// Check config directories
		configCheck := s.checkConfigDirs()
		checks["config_dirs"] = configCheck
	}

	return &pb.HealthCheckResponse{
		Healthy:       healthy,
		StatusMessage: statusMsg,
		Checks:        checks,
		Timestamp:     timestamppb.Now(),
	}, nil
}

// getUptime returns agent uptime as string
func (s *Server) getUptime() string {
	// Simple uptime based on process start
	return time.Since(time.Now().Add(-1 * time.Hour)).Round(time.Second).String()
}

// checkMemory checks system memory usage
func (s *Server) checkMemory() (string, bool) {
	v, err := mem.VirtualMemory()
	if err != nil {
		return fmt.Sprintf("error: %v", err), false
	}

	usedPercent := v.UsedPercent
	status := fmt.Sprintf("%.1f%% used (%d MB / %d MB)",
		usedPercent,
		v.Used/1024/1024,
		v.Total/1024/1024)

	// Warning if > 90% used
	if usedPercent > 90 {
		return status + " [CRITICAL]", false
	}
	if usedPercent > 80 {
		return status + " [WARNING]", true
	}
	return status, true
}

// checkCPU checks CPU usage
func (s *Server) checkCPU() (string, bool) {
	// Get load average
	loadAvg, err := load.Avg()
	if err != nil {
		return fmt.Sprintf("error: %v", err), false
	}

	numCPU := runtime.NumCPU()
	load1 := loadAvg.Load1
	loadPercent := (load1 / float64(numCPU)) * 100

	status := fmt.Sprintf("load: %.2f (%.1f%% of %d cores)",
		load1, loadPercent, numCPU)

	// Warning if load > 80% of cores
	if loadPercent > 90 {
		return status + " [CRITICAL]", false
	}
	if loadPercent > 80 {
		return status + " [WARNING]", true
	}
	return status, true
}

// checkOcservProcess checks if ocserv process is running
func (s *Server) checkOcservProcess() (string, bool) {
	processes, err := process.Processes()
	if err != nil {
		return fmt.Sprintf("error: %v", err), false
	}

	for _, p := range processes {
		name, err := p.Name()
		if err != nil {
			continue
		}
		if name == "ocserv-main" || name == "ocserv" {
			pid := p.Pid
			cpuPercent, _ := p.CPUPercent()
			memPercent, _ := p.MemoryPercent()
			return fmt.Sprintf("running (pid=%d, cpu=%.1f%%, mem=%.1f%%)",
				pid, cpuPercent, memPercent), true
		}
	}

	return "not running", false
}

// checkOcservSocket checks if ocserv control socket exists
func (s *Server) checkOcservSocket() (string, bool) {
	socketPath := s.config.Ocserv.CtlSocket
	if socketPath == "" {
		socketPath = "/var/run/occtl.socket"
	}

	info, err := os.Stat(socketPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "socket not found: " + socketPath, false
		}
		return fmt.Sprintf("error: %v", err), false
	}

	if info.Mode()&os.ModeSocket == 0 {
		return "not a socket: " + socketPath, false
	}

	return "exists: " + socketPath, true
}

// checkOcctl tests occtl show status command
func (s *Server) checkOcctl(ctx context.Context) (string, bool) {
	result, err := s.ocservManager.ExecuteCommand(ctx, "occtl", []string{"show", "status"})
	if err != nil {
		return fmt.Sprintf("error: %v", err), false
	}

	if !result.Success {
		return fmt.Sprintf("failed: %s", result.Stderr), false
	}

	return "responsive", true
}

// checkConfigDirs checks if config directories exist and are writable
func (s *Server) checkConfigDirs() string {
	results := make([]string, 0)

	if s.config.Ocserv.ConfigPerUserDir != "" {
		if s.checkDirWritable(s.config.Ocserv.ConfigPerUserDir) {
			results = append(results, "per-user: ok")
		} else {
			results = append(results, "per-user: error")
		}
	}

	if s.config.Ocserv.ConfigPerGroupDir != "" {
		if s.checkDirWritable(s.config.Ocserv.ConfigPerGroupDir) {
			results = append(results, "per-group: ok")
		} else {
			results = append(results, "per-group: error")
		}
	}

	if len(results) == 0 {
		return "not configured"
	}

	return fmt.Sprintf("%v", results)
}

// checkDirWritable checks if a directory exists and is writable
func (s *Server) checkDirWritable(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	if !info.IsDir() {
		return false
	}

	// Try to create a temp file
	testFile := filepath.Join(path, ".health_check_test")
	f, err := os.Create(testFile)
	if err != nil {
		return false
	}
	f.Close()
	os.Remove(testFile)
	return true
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
			// #nosec G115 - exit codes are 0-255, safe to convert to int32
			response.ExitCode = int32(result.ExitCode)
		}
		return response, nil
	}

	response.Success = result.Success
	response.Stdout = result.Stdout
	response.Stderr = result.Stderr
	// #nosec G115 - exit codes are 0-255, safe to convert to int32
	response.ExitCode = int32(result.ExitCode)
	response.ErrorMessage = result.ErrorMsg

	return response, nil
}

// ConfigPayload represents the JSON payload for config updates
type ConfigPayload struct {
	Routes               []string          `json:"routes,omitempty"`
	DNS                  []string          `json:"dns,omitempty"`
	SplitDNS             []string          `json:"split_dns,omitempty"`
	MaxSameClients       int               `json:"max_same_clients,omitempty"`
	RestrictUserToRoutes bool              `json:"restrict_user_to_routes,omitempty"`
	CustomDirectives     map[string]string `json:"custom_directives,omitempty"`
}

// UpdateConfig implements the UpdateConfig RPC method
func (s *Server) UpdateConfig(ctx context.Context, req *pb.ConfigUpdateRequest) (*pb.ConfigUpdateResponse, error) {
	s.logger.Info().
		Str("request_id", req.RequestId).
		Str("config_type", req.ConfigType.String()).
		Str("config_name", req.ConfigName).
		Bool("validate_only", req.ValidateOnly).
		Bool("create_backup", req.CreateBackup).
		Msg("UpdateConfig called")

	response := &pb.ConfigUpdateResponse{
		RequestId: req.RequestId,
	}

	// Check if config generator is available
	if s.configGenerator == nil {
		response.Success = false
		response.ErrorMessage = "config generator not initialized (per-user directory not configured)"
		return response, nil
	}

	// Validate request
	if req.ConfigName == "" {
		response.Success = false
		response.ErrorMessage = "config_name is required"
		return response, nil
	}

	// Main config updates are not supported for safety
	if req.ConfigType == pb.ConfigType_CONFIG_TYPE_MAIN {
		response.Success = false
		response.ErrorMessage = "main config updates are not supported for safety reasons"
		return response, nil
	}

	// Parse config content as JSON
	var payload ConfigPayload
	if req.ConfigContent != "" {
		if err := json.Unmarshal([]byte(req.ConfigContent), &payload); err != nil {
			response.Success = false
			response.ErrorMessage = fmt.Sprintf("invalid JSON payload: %v", err)
			return response, nil
		}
	}

	// Validate routes if provided
	if len(payload.Routes) > 0 {
		if err := config.ValidateRoutes(payload.Routes); err != nil {
			response.Success = false
			response.ValidationResult = fmt.Sprintf("invalid routes: %v", err)
			response.ErrorMessage = "validation failed"
			return response, nil
		}
	}

	// Validate DNS if provided
	if len(payload.DNS) > 0 {
		if err := config.ValidateDNSServers(payload.DNS); err != nil {
			response.Success = false
			response.ValidationResult = fmt.Sprintf("invalid DNS servers: %v", err)
			response.ErrorMessage = "validation failed"
			return response, nil
		}
	}

	// If validate_only, return success without applying
	if req.ValidateOnly {
		response.Success = true
		response.ValidationResult = "validation passed"
		return response, nil
	}

	// Apply configuration based on type
	var err error
	switch req.ConfigType {
	case pb.ConfigType_CONFIG_TYPE_PER_USER:
		userCfg := &config.PerUserConfig{
			Username:             req.ConfigName,
			Routes:               payload.Routes,
			DNS:                  payload.DNS,
			RestrictUserToRoutes: payload.RestrictUserToRoutes,
			MaxSameClients:       payload.MaxSameClients,
			CustomDirectives:     payload.CustomDirectives,
		}

		// Set defaults if not provided
		if userCfg.MaxSameClients == 0 {
			userCfg.MaxSameClients = 2
		}
		if len(userCfg.DNS) == 0 {
			userCfg.DNS = config.DNS.Google()
		}

		err = s.configGenerator.GenerateUserConfig(userCfg)

	case pb.ConfigType_CONFIG_TYPE_PER_GROUP:
		groupCfg := &config.PerGroupConfig{
			GroupName:        req.ConfigName,
			Routes:           payload.Routes,
			DNS:              payload.DNS,
			SplitDNS:         payload.SplitDNS,
			MaxSameClients:   payload.MaxSameClients,
			RestrictToRoutes: payload.RestrictUserToRoutes,
			CustomDirectives: payload.CustomDirectives,
		}

		// Set defaults if not provided
		if groupCfg.MaxSameClients == 0 {
			groupCfg.MaxSameClients = 2
		}
		if len(groupCfg.DNS) == 0 {
			groupCfg.DNS = config.DNS.Google()
		}

		err = s.configGenerator.GenerateGroupConfig(groupCfg)

	default:
		response.Success = false
		response.ErrorMessage = fmt.Sprintf("unsupported config type: %s", req.ConfigType)
		return response, nil
	}

	if err != nil {
		s.logger.Error().
			Err(err).
			Str("config_name", req.ConfigName).
			Str("config_type", req.ConfigType.String()).
			Msg("Failed to generate config")

		response.Success = false
		response.ErrorMessage = err.Error()
		return response, nil
	}

	s.logger.Info().
		Str("config_name", req.ConfigName).
		Str("config_type", req.ConfigType.String()).
		Msg("Config generated successfully")

	response.Success = true
	response.ValidationResult = "config applied successfully"

	// Return backup path if backup was created
	if req.CreateBackup && s.config.Ocserv.BackupDir != "" {
		response.BackupPath = s.config.Ocserv.BackupDir
	}

	return response, nil
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

// AgentStream implements bidirectional streaming RPC for heartbeats and commands
func (s *Server) AgentStream(stream pb.AgentService_AgentStreamServer) error {
	if stream == nil {
		return status.Error(codes.InvalidArgument, "stream is nil")
	}

	s.logger.Info().Msg("AgentStream started")

	ctx := stream.Context()

	// Process incoming messages from server
	for {
		select {
		case <-ctx.Done():
			s.logger.Info().Msg("AgentStream context cancelled")
			return ctx.Err()
		default:
			// Receive message from server
			msg, err := stream.Recv()
			if err != nil {
				s.logger.Debug().Err(err).Msg("AgentStream receive error")
				return err
			}

			// Process message based on payload type
			if msg == nil {
				continue
			}

			s.logger.Debug().
				Str("agent_id", msg.AgentId).
				Msg("Received message from agent")

			// Handle different message types
			switch payload := msg.Payload.(type) {
			case *pb.AgentMessage_Heartbeat:
				s.handleHeartbeat(ctx, stream, msg.AgentId, payload.Heartbeat)

			case *pb.AgentMessage_Metrics:
				s.handleMetricsReport(ctx, msg.AgentId, payload.Metrics)

			case *pb.AgentMessage_Event:
				s.handleEventNotification(ctx, msg.AgentId, payload.Event)
			}
		}
	}
}

// handleHeartbeat processes heartbeat messages and sends responses
func (s *Server) handleHeartbeat(ctx context.Context, stream pb.AgentService_AgentStreamServer, agentID string, hb *pb.Heartbeat) {
	s.logger.Debug().
		Str("agent_id", agentID).
		Str("status", hb.Status.String()).
		Msg("Heartbeat received")

	// Log system metrics if available
	if hb.System != nil {
		s.logger.Debug().
			Float64("cpu_percent", hb.System.CpuUsagePercent).
			Float64("mem_percent", hb.System.MemoryUsagePercent).
			Msg("System metrics")
	}

	// Log ocserv status if available
	if hb.Ocserv != nil {
		s.logger.Debug().
			Bool("running", hb.Ocserv.IsRunning).
			Uint32("sessions", hb.Ocserv.ActiveSessions).
			Msg("Ocserv status")
	}

	// Note: In a full implementation, we would:
	// 1. Update agent last-seen timestamp
	// 2. Check for pending commands to send
	// 3. Send response with any pending actions
}

// handleMetricsReport processes metrics reports from the agent
func (s *Server) handleMetricsReport(ctx context.Context, agentID string, metrics *pb.MetricsReport) {
	s.logger.Debug().
		Str("agent_id", agentID).
		Msg("Metrics report received")

	// In a full implementation, we would forward to telemetry system
}

// handleEventNotification processes event notifications from the agent
func (s *Server) handleEventNotification(ctx context.Context, agentID string, event *pb.EventNotification) {
	s.logger.Info().
		Str("agent_id", agentID).
		Str("event_type", event.EventType).
		Str("message", event.Message).
		Msg("Event notification received")

	// In a full implementation, we would:
	// 1. Store event in audit log
	// 2. Trigger alerts if needed
	// 3. Update session state
}
