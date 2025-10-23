package testutil

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// FixturesPath returns the absolute path to fixtures directory
func FixturesPath(t *testing.T) string {
	t.Helper()

	// From internal/ocserv/testutil, go up to project root
	pwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	// Navigate to fixtures from project root
	fixturesPath := filepath.Join(pwd, "..", "..", "..", "test", "fixtures", "ocserv", "occtl")

	// Verify path exists
	if _, err := os.Stat(fixturesPath); os.IsNotExist(err) {
		t.Fatalf("Fixtures directory not found: %s", fixturesPath)
	}

	return fixturesPath
}

// LoadFixture loads a fixture file and returns its contents
func LoadFixture(t *testing.T, name string) []byte {
	t.Helper()

	fixturesPath := FixturesPath(t)
	fixturePath := filepath.Join(fixturesPath, name)

	data, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("Failed to read fixture %s: %v", name, err)
	}

	return data
}

// LoadFixtureJSON loads a fixture and validates it's valid JSON
func LoadFixtureJSON(t *testing.T, name string) []byte {
	t.Helper()

	data := LoadFixture(t, name)

	// Validate JSON
	var js interface{}
	if err := json.Unmarshal(data, &js); err != nil {
		t.Fatalf("Fixture %s is not valid JSON: %v", name, err)
	}

	return data
}

// ValidateFixture validates a fixture file exists and is valid JSON
func ValidateFixture(t *testing.T, name string) {
	t.Helper()

	fixturesPath := FixturesPath(t)
	fixturePath := filepath.Join(fixturesPath, name)

	// Check file exists
	info, err := os.Stat(fixturePath)
	if err != nil {
		t.Fatalf("Fixture %s does not exist: %v", name, err)
	}

	// Check not empty
	if info.Size() == 0 {
		t.Fatalf("Fixture %s is empty", name)
	}

	// Validate JSON
	data, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("Failed to read fixture %s: %v", name, err)
	}

	var js interface{}
	if err := json.Unmarshal(data, &js); err != nil {
		t.Fatalf("Fixture %s is not valid JSON: %v (content: %s)", name, err, string(data))
	}

	t.Logf("Fixture %s is valid (%d bytes)", name, info.Size())
}

// ValidateAllFixtures validates all expected fixture files exist and are valid JSON
func ValidateAllFixtures(t *testing.T) {
	t.Helper()

	expectedFixtures := []string{
		"occtl -j show users",
		"occtl -j show user",
		"occtl -j show id",
		"occtl -j show status",
		"occtl -j show sessions all",
		"occtl -j show sessions valid",
		"occtl -j show session",
		"occtl -j show cookies all",
		"occtl -j show cookies valid",
		"occtl -j show iroutes",
		"occtl -j show ip ban points",
		"occtl -j show events",
		"occtl show id", // Plain text format
	}

	t.Logf("Validating %d fixtures...", len(expectedFixtures))

	for _, fixture := range expectedFixtures {
		// Skip plain text fixture validation (not JSON)
		if fixture == "occtl show id" {
			fixturesPath := FixturesPath(t)
			fixturePath := filepath.Join(fixturesPath, fixture)
			if _, err := os.Stat(fixturePath); err != nil {
				t.Fatalf("Plain text fixture %s does not exist: %v", fixture, err)
			}
			t.Logf("Fixture %s exists (plain text, skipping JSON validation)", fixture)
			continue
		}

		// Validate JSON fixtures
		ValidateFixture(t, fixture)
	}

	t.Logf("All %d fixtures validated successfully", len(expectedFixtures))
}

// ExpectedUsersCount returns expected number of users in "occtl -j show users" fixture
func ExpectedUsersCount(t *testing.T) int {
	t.Helper()

	data := LoadFixtureJSON(t, "occtl -j show users")

	var users []interface{}
	if err := json.Unmarshal(data, &users); err != nil {
		t.Fatalf("Failed to unmarshal users fixture: %v", err)
	}

	return len(users)
}

// ExpectedSessionsCount returns expected number of sessions in "occtl -j show sessions all" fixture
func ExpectedSessionsCount(t *testing.T) int {
	t.Helper()

	data := LoadFixtureJSON(t, "occtl -j show sessions all")

	var sessions []interface{}
	if err := json.Unmarshal(data, &sessions); err != nil {
		t.Fatalf("Failed to unmarshal sessions fixture: %v", err)
	}

	return len(sessions)
}

// GetFixtureString returns fixture as string (for text fixtures)
func GetFixtureString(t *testing.T, name string) string {
	t.Helper()
	return string(LoadFixture(t, name))
}

// UnmarshalFixture loads and unmarshals a JSON fixture into target
func UnmarshalFixture(t *testing.T, name string, target interface{}) {
	t.Helper()

	data := LoadFixtureJSON(t, name)

	if err := json.Unmarshal(data, target); err != nil {
		t.Fatalf("Failed to unmarshal fixture %s into %T: %v", name, target, err)
	}
}

// CompareJSON compares two JSON byte arrays for equality
func CompareJSON(t *testing.T, expected, actual []byte) {
	t.Helper()

	var expectedJSON, actualJSON interface{}

	if err := json.Unmarshal(expected, &expectedJSON); err != nil {
		t.Fatalf("Failed to unmarshal expected JSON: %v", err)
	}

	if err := json.Unmarshal(actual, &actualJSON); err != nil {
		t.Fatalf("Failed to unmarshal actual JSON: %v", err)
	}

	expectedStr := fmt.Sprintf("%v", expectedJSON)
	actualStr := fmt.Sprintf("%v", actualJSON)

	if expectedStr != actualStr {
		t.Fatalf("JSON mismatch:\nExpected: %s\nActual: %s", expectedStr, actualStr)
	}
}
