package config

import (
	"fmt"
	"net"
	"net/netip"
	"strings"

	"github.com/cockroachdb/errors"
)

// Route represents a network route in various formats
type Route struct {
	Network net.IPNet
	CIDR    string
	Netmask string
}

// ParseRoute parses a route in CIDR or IP/Netmask format
func ParseRoute(route string) (*Route, error) {
	route = strings.TrimSpace(route)
	if route == "" {
		return nil, errors.New("empty route")
	}

	// Try parsing as CIDR
	if strings.Contains(route, "/") {
		// Check if it's CIDR notation (e.g., "10.0.0.0/8")
		if _, ipnet, err := net.ParseCIDR(route); err == nil {
			return &Route{
				Network: *ipnet,
				CIDR:    route,
				Netmask: netmaskFromCIDR(ipnet),
			}, nil
		}

		// Try parsing as IP/Netmask format (e.g., "10.0.0.0/255.0.0.0")
		parts := strings.Split(route, "/")
		if len(parts) == 2 {
			ip := net.ParseIP(parts[0])
			mask := net.ParseIP(parts[1])

			if ip != nil && mask != nil {
				// Convert netmask to CIDR
				ipnet := &net.IPNet{
					IP:   ip,
					Mask: net.IPMask(mask.To4()),
				}

				cidr := ipnet.String()
				return &Route{
					Network: *ipnet,
					CIDR:    cidr,
					Netmask: route,
				}, nil
			}
		}
	}

	return nil, errors.Newf("invalid route format: %s", route)
}

// ValidateRoutes validates a list of routes
func ValidateRoutes(routes []string) error {
	for i, route := range routes {
		// Skip empty routes
		if strings.TrimSpace(route) == "" {
			continue
		}

		// Check for no-route directive and trim prefix unconditionally
		route = strings.TrimPrefix(route, "no-route = ")

		// Parse route
		if _, err := ParseRoute(route); err != nil {
			return errors.Wrapf(err, "route[%d]", i)
		}
	}
	return nil
}

// ValidateDNSServers validates a list of DNS server IP addresses
func ValidateDNSServers(servers []string) error {
	for i, server := range servers {
		server = strings.TrimSpace(server)
		if server == "" {
			continue
		}

		// Parse as IP address
		if ip := net.ParseIP(server); ip == nil {
			return errors.Newf("dns[%d]: invalid IP address: %s", i, server)
		}
	}
	return nil
}

// NormalizeRoutes converts routes to consistent format (IP/Netmask)
func NormalizeRoutes(routes []string) ([]string, error) {
	normalized := make([]string, 0, len(routes))

	for _, route := range routes {
		route = strings.TrimSpace(route)
		if route == "" {
			continue
		}

		// Handle no-route directive
		if strings.HasPrefix(route, "no-route = ") {
			normalized = append(normalized, route)
			continue
		}

		// Parse and normalize
		r, err := ParseRoute(route)
		if err != nil {
			return nil, errors.Wrapf(err, "normalize route: %s", route)
		}

		// Use netmask format for ocserv
		normalized = append(normalized, r.Netmask)
	}

	return normalized, nil
}

// RouteContains checks if a route contains a specific IP address
func RouteContains(route string, ip string) (bool, error) {
	r, err := ParseRoute(route)
	if err != nil {
		return false, errors.Wrap(err, "parse route")
	}

	testIP := net.ParseIP(ip)
	if testIP == nil {
		return false, errors.Newf("invalid IP: %s", ip)
	}

	return r.Network.Contains(testIP), nil
}

// RouteOverlaps checks if two routes overlap
func RouteOverlaps(route1, route2 string) (bool, error) {
	r1, err := ParseRoute(route1)
	if err != nil {
		return false, errors.Wrap(err, "parse route1")
	}

	r2, err := ParseRoute(route2)
	if err != nil {
		return false, errors.Wrap(err, "parse route2")
	}

	// Check if r1 contains r2's network address
	if r1.Network.Contains(r2.Network.IP) {
		return true, nil
	}

	// Check if r2 contains r1's network address
	if r2.Network.Contains(r1.Network.IP) {
		return true, nil
	}

	return false, nil
}

// MergeRoutes merges multiple route lists, removing duplicates
func MergeRoutes(routeLists ...[]string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0)

	for _, routes := range routeLists {
		for _, route := range routes {
			route = strings.TrimSpace(route)
			if route == "" {
				continue
			}

			// Normalize route for deduplication
			normalized, err := NormalizeRoutes([]string{route})
			if err != nil {
				continue
			}

			for _, nr := range normalized {
				if !seen[nr] {
					seen[nr] = true
					result = append(result, nr)
				}
			}
		}
	}

	return result
}

// netmaskFromCIDR converts a CIDR network to IP/Netmask format
func netmaskFromCIDR(ipnet *net.IPNet) string {
	ip := ipnet.IP.String()
	mask := net.IP(ipnet.Mask).String()
	return fmt.Sprintf("%s/%s", ip, mask)
}

// CIDRToNetmask converts CIDR notation to netmask notation
func CIDRToNetmask(cidr string) (string, error) {
	_, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return "", errors.Wrapf(err, "parse CIDR: %s", cidr)
	}
	return netmaskFromCIDR(ipnet), nil
}

// NetmaskToCIDR converts netmask notation to CIDR notation
func NetmaskToCIDR(netmask string) (string, error) {
	r, err := ParseRoute(netmask)
	if err != nil {
		return "", errors.Wrap(err, "parse netmask")
	}
	return r.CIDR, nil
}

// SummarizeRoutes attempts to summarize overlapping routes into larger networks
func SummarizeRoutes(routes []string) ([]string, error) {
	if len(routes) == 0 {
		return routes, nil
	}

	// Parse all routes into netip.Prefix for better manipulation
	prefixes := make([]netip.Prefix, 0, len(routes))

	for _, route := range routes {
		r, err := ParseRoute(route)
		if err != nil {
			return nil, errors.Wrapf(err, "parse route: %s", route)
		}

		// Convert to netip.Prefix
		prefix, err := netip.ParsePrefix(r.CIDR)
		if err != nil {
			return nil, errors.Wrapf(err, "parse prefix: %s", r.CIDR)
		}

		prefixes = append(prefixes, prefix)
	}

	// Simple deduplication (real summarization is complex)
	seen := make(map[string]bool)
	result := make([]string, 0)

	for _, prefix := range prefixes {
		cidr := prefix.String()
		if !seen[cidr] {
			seen[cidr] = true
			// Convert back to netmask format
			netmask, _ := CIDRToNetmask(cidr)
			result = append(result, netmask)
		}
	}

	return result, nil
}
