package ipc

// AuthRequest represents an authentication request from vpn-auth CLI
type AuthRequest struct {
	Reason    string `json:"reason"`     // connect, disconnect, host-update
	Username  string `json:"username"`   // From certificate CN
	GroupName string `json:"groupname"`  // From certificate OU
	IPReal    string `json:"ip_real"`    // Client IP
	IPRemote  string `json:"ip_remote"`  // VPN IP
	Device    string `json:"device"`     // tun/tap device
	SessionID string `json:"session_id"` // ocserv session ID
}

// AuthResponse represents the response to an authentication request
type AuthResponse struct {
	Allowed bool   `json:"allowed"`           // Whether connection is allowed
	Error   string `json:"error,omitempty"`   // Error message if not allowed
	Message string `json:"message,omitempty"` // Additional information
}
