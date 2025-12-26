package portal

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log/slog"
	"os"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

// Client provides communication with the portal server
type Client struct {
	conn   *grpc.ClientConn
	logger *slog.Logger
	tracer trace.Tracer
	config *Config
}

// Config configures the portal client
type Config struct {
	Address  string
	TLSCert  string
	TLSKey   string
	TLSCA    string
	Timeout  time.Duration
	Insecure bool
}

// NewClient creates a new portal client
func NewClient(ctx context.Context, cfg *Config, logger *slog.Logger, tracer trace.Tracer, tracerProvider trace.TracerProvider, meterProvider metric.MeterProvider) (*Client, error) {
	if cfg.Address == "" {
		return nil, fmt.Errorf("portal address is required")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}
	if tracer == nil {
		return nil, fmt.Errorf("tracer is required")
	}

	// Set default timeout
	if cfg.Timeout == 0 {
		cfg.Timeout = 10 * time.Second
	}

	// Prepare dial options
	var opts []grpc.DialOption

	// Add OpenTelemetry instrumentation
	opts = append(opts,
		grpc.WithStatsHandler(otelgrpc.NewClientHandler(
			otelgrpc.WithTracerProvider(tracerProvider),
			otelgrpc.WithMeterProvider(meterProvider),
		)),
	)

	// Configure TLS or insecure connection
	if cfg.Insecure {
		logger.WarnContext(ctx, "using insecure gRPC connection to portal")
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	} else {
		tlsConfig, err := loadTLSConfig(cfg)
		if err != nil {
			return nil, fmt.Errorf("load TLS config: %w", err)
		}
		creds := credentials.NewTLS(tlsConfig)
		opts = append(opts, grpc.WithTransportCredentials(creds))
	}

	// Configure keepalive
	opts = append(opts, grpc.WithKeepaliveParams(keepalive.ClientParameters{
		Time:                10 * time.Second,
		Timeout:             5 * time.Second,
		PermitWithoutStream: true,
	}))

	// Dial portal
	conn, err := grpc.NewClient(cfg.Address, opts...)
	if err != nil {
		return nil, fmt.Errorf("dial portal: %w", err)
	}

	logger.InfoContext(ctx, "portal client connected",
		slog.String("address", cfg.Address),
		slog.Bool("tls", !cfg.Insecure),
	)

	return &Client{
		conn:   conn,
		logger: logger,
		tracer: tracer,
		config: cfg,
	}, nil
}

// Close closes the portal client connection
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// loadTLSConfig loads mTLS configuration
func loadTLSConfig(cfg *Config) (*tls.Config, error) {
	// Load CA certificate
	caCert, err := os.ReadFile(cfg.TLSCA)
	if err != nil {
		return nil, fmt.Errorf("read CA cert: %w", err)
	}

	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to append CA cert")
	}

	// Load client certificate and key
	cert, err := tls.LoadX509KeyPair(cfg.TLSCert, cfg.TLSKey)
	if err != nil {
		return nil, fmt.Errorf("load client cert: %w", err)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      certPool,
		MinVersion:   tls.VersionTLS13,
	}, nil
}
