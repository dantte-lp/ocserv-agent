package ocserv

import "time"

// UserDetailed represents complete user information from 'show user' command
// Based on production ocserv 1.3.0 JSON output
type UserDetailed struct {
	// Identity
	ID        int    `json:"ID"`
	Username  string `json:"Username"`
	Groupname string `json:"Groupname"`
	State     string `json:"State"` // "connected", "authenticated"
	Vhost     string `json:"vhost"` // "default" or virtual host name

	// Network
	Device        string `json:"Device"`          // "vpns0"
	MTU           string `json:"MTU"`             // "1402"
	RemoteIP      string `json:"Remote IP"`       // Client's real IP
	Location      string `json:"Location"`        // GeoIP location or "unknown"
	LocalDeviceIP string `json:"Local Device IP"` // Server's interface IP

	// VPN IPs
	IPv4    string `json:"IPv4"`       // "10.0.16.23"
	PtPIPv4 string `json:"P-t-P IPv4"` // "10.0.16.1"
	IPv6    string `json:"IPv6"`       // "fc00::1:8651"
	PtPIPv6 string `json:"P-t-P IPv6"` // "fc00::1:8601"

	// Client info
	UserAgent string `json:"User-Agent"`         // "AnyConnect AppleSSLVPN_Darwin_ARM (iPhone) 5.1.11.347"
	Hostname  string `json:"Hostname,omitempty"` // "localhost" (optional)

	// Traffic stats
	RX          string `json:"RX"`         // "0"
	TX          string `json:"TX"`         // "96"
	RXFormatted string `json:"_RX"`        // "0 bytes"
	TXFormatted string `json:"_TX"`        // "96 bytes"
	AverageRX   string `json:"Average RX"` // "0 bytes/s"
	AverageTX   string `json:"Average TX"` // "32 bytes/s"

	// Connection params
	DPD       string `json:"DPD"`       // "90"
	KeepAlive string `json:"KeepAlive"` // "32400"

	// Connection time
	ConnectedAt         string `json:"Connected at"`     // "2025-10-23 02:32"
	ConnectedAtRelative string `json:"_Connected at"`    // "    3s"
	RawConnectedAt      int64  `json:"raw_connected_at"` // 1761175942 (Unix timestamp)

	// Session
	FullSession string `json:"Full session"` // "0/zuQ1RjBWv5J/hneJun8+sesWs="
	Session     string `json:"Session"`      // "0/zuQ1"

	// Security
	TLSCiphersuite string `json:"TLS ciphersuite"`       // "(TLS1.3)-(ECDHE-SECP256R1)-(RSA-PSS-RSAE-SHA256)-(AES-256-GCM)"
	DTLSCipher     string `json:"DTLS cipher,omitempty"` // "(DTLS1.2)-(ECDHE-RSA)-(AES-256-GCM)" (optional)

	// Compression
	CSTPCompression string `json:"CSTP compression,omitempty"` // "lzs" (optional)
	DTLSCompression string `json:"DTLS compression,omitempty"` // "lzs" (optional)

	// Network config
	DNS             []string    `json:"DNS"`               // ["10.0.16.1", "fc00::1:8601"]
	NBNS            []string    `json:"NBNS"`              // []
	SplitDNSDomains []string    `json:"Split-DNS-Domains"` // []
	Routes          interface{} `json:"Routes"`            // "defaultroute" or []string
	NoRoutes        []string    `json:"No-routes"`         // []
	IRoutes         []string    `json:"iRoutes"`           // []

	// Restrictions
	RestrictedToRoutes string   `json:"Restricted to routes"` // "False" or "True"
	RestrictedToPorts  []string `json:"Restricted to ports"`  // []
}

// SessionInfo represents session information from 'show sessions' or 'show session' commands
type SessionInfo struct {
	Session     string `json:"Session"`      // "0/zuQ1"
	FullSession string `json:"Full session"` // "0/zuQ1RjBWv5J/hneJun8+sesWs="
	Created     string `json:"Created"`      // "2025-10-23 02:30"
	State       string `json:"State"`        // "authenticated"
	Username    string `json:"Username"`     // "lpa"
	Groupname   string `json:"Groupname"`    // "(none)"
	Vhost       string `json:"vhost"`        // "default"
	UserAgent   string `json:"User-Agent"`   // "AnyConnect AppleSSLVPN_Darwin_ARM (iPhone) 5.1.11.347"
	RemoteIP    string `json:"Remote IP"`    // "90.156.164.225"
	Location    string `json:"Location"`     // "unknown"

	// Session flags
	SessionIsOpen int `json:"session_is_open"` // 1 or 0
	TLSAuthOK     int `json:"tls_auth_ok"`     // 1 or 0
	InUse         int `json:"in_use"`          // 1 or 0
}

// ServerStatusDetailed represents complete server status from 'show status' command
type ServerStatusDetailed struct {
	// Status
	Status          string `json:"Status"`                 // "online"
	ServerPID       int    `json:"Server PID"`             // 802
	SecModPID       int    `json:"Sec-mod PID"`            // 821
	SecModInstances int    `json:"Sec-mod instance count"` // 1

	// Uptime
	UpSince         string `json:"Up since"`     // "2025-09-12 14:37"
	UpSinceRelative string `json:"_Up since"`    // "40days"
	RawUpSince      int64  `json:"raw_up_since"` // 1757677078
	Uptime          int64  `json:"uptime"`       // 3498723 (seconds)

	// Sessions
	ActiveSessions int `json:"Active sessions"`               // 0
	TotalSessions  int `json:"Total sessions"`                // 44
	TotalAuthFails int `json:"Total authentication failures"` // 10
	IPsInBanList   int `json:"IPs in ban list"`               // 0

	// Stats reset
	LastStatsReset         string `json:"Last stats reset"`     // "2025-10-20 20:40"
	LastStatsResetRelative string `json:"_Last stats reset"`    // " 2days"
	RawLastStatsReset      int64  `json:"raw_last_stats_reset"` // 1760982020

	// Since last reset
	SessionsHandled      int `json:"Sessions handled"`             // 4
	TimedOutSessions     int `json:"Timed out sessions"`           // 0
	IdleTimedOutSessions int `json:"Timed out (idle) sessions"`    // 0
	ErrorClosedSessions  int `json:"Closed due to error sessions"` // 2
	AuthFailures         int `json:"Authentication failures"`      // 0

	// Timing stats
	AvgAuthTime    string `json:"Average auth time"` // "    0s"
	RawAvgAuthTime int    `json:"raw_avg_auth_time"` // 0 (seconds)
	MaxAuthTime    string `json:"Max auth time"`     // "    5s"
	RawMaxAuthTime int    `json:"raw_max_auth_time"` // 5

	AvgSessionTime    string `json:"Average session time"` // " 3h:43m"
	RawAvgSessionTime int    `json:"raw_avg_session_time"` // 13380 (seconds)
	MaxSessionTime    string `json:"Max session time"`     // " 1h:32m"
	RawMaxSessionTime int    `json:"raw_max_session_time"` // 5520 (seconds)

	// Network
	MinMTU int `json:"Min MTU"` // 1324
	MaxMTU int `json:"Max MTU"` // 1402

	// Traffic (since last reset)
	RX    string `json:"RX"`     // "110.0 MB"
	RawRX int64  `json:"raw_rx"` // 110013000 (bytes)
	TX    string `json:"TX"`     // "1.8 GB"
	RawTX int64  `json:"raw_tx"` // 1827434000 (bytes)
}

// IRoute represents user-provided route information from 'show iroutes' command
type IRoute struct {
	ID       int      `json:"ID"`       // 835257
	Username string   `json:"Username"` // "lpa"
	Vhost    string   `json:"vhost"`    // "default"
	Device   string   `json:"Device"`   // "vpns0"
	IP       string   `json:"IP"`       // "10.0.16.23"
	IRoutes  []string `json:"iRoutes"`  // [] or ["192.168.1.0/24"]
}

// IPBan represents banned IP information from 'show ip bans' command
type IPBan struct {
	IP        string    `json:"ip"`
	Score     int       `json:"score"`
	BannedAt  time.Time `json:"banned_at"`
	ExpiresAt time.Time `json:"expires_at"`
	Reason    string    `json:"reason,omitempty"`
}

// IPBanPoints represents IP with accumulated violation points from 'show ip ban points' command
type IPBanPoints struct {
	IP           string    `json:"ip"`
	Points       int       `json:"points"`
	LastActivity time.Time `json:"last_activity"`
	Events       []string  `json:"events,omitempty"`
}

// Event represents a connection event from 'show events' command (streaming)
type Event struct {
	Timestamp time.Time `json:"timestamp"`
	EventType string    `json:"event_type"` // "connect", "disconnect", "auth-failure"
	Username  string    `json:"username"`
	RemoteIP  string    `json:"remote_ip"`
	SessionID string    `json:"session_id,omitempty"`
	Reason    string    `json:"reason,omitempty"`
	Details   string    `json:"details,omitempty"`
}
