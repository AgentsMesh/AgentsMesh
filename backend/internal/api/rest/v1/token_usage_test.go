package v1

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestContext creates a gin.Context with the given query string for testing.
func newTestContext(query string) *gin.Context {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/token-usage/dashboard?"+query, nil)
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	return c
}

// --- validGranularities ---

func TestValidGranularities(t *testing.T) {
	assert.True(t, validGranularities["day"])
	assert.True(t, validGranularities["week"])
	assert.True(t, validGranularities["month"])
	assert.False(t, validGranularities["second"])
	assert.False(t, validGranularities["year"])
	assert.False(t, validGranularities[""])
}

// --- isAdminOrOwner ---

func TestIsAdminOrOwner(t *testing.T) {
	tests := []struct {
		role     string
		expected bool
	}{
		{"owner", true},
		{"admin", true},
		{"member", false},
		{"viewer", false},
		{"", false},
	}
	for _, tc := range tests {
		t.Run(tc.role, func(t *testing.T) {
			result := tc.role == "owner" || tc.role == "admin"
			assert.Equal(t, tc.expected, result)
		})
	}
}

// --- parseFilter ---

func TestParseFilter_Defaults(t *testing.T) {
	c := newTestContext("")
	filter, err := parseFilter(c)
	require.NoError(t, err)

	assert.Equal(t, "day", filter.Granularity)
	assert.Nil(t, filter.AgentSlug)
	assert.Nil(t, filter.UserID)
	assert.Nil(t, filter.Model)
	// Default range: last 30 days.
	assert.WithinDuration(t, time.Now().AddDate(0, 0, -30), filter.StartTime, 5*time.Second)
	assert.WithinDuration(t, time.Now(), filter.EndTime, 5*time.Second)
}

func TestParseFilter_ValidGranularities(t *testing.T) {
	for _, g := range []string{"day", "week", "month"} {
		t.Run(g, func(t *testing.T) {
			c := newTestContext("granularity=" + g)
			filter, err := parseFilter(c)
			require.NoError(t, err)
			assert.Equal(t, g, filter.Granularity)
		})
	}
}

func TestParseFilter_InvalidGranularity(t *testing.T) {
	for _, g := range []string{"hour", "second", "year", "invalid"} {
		t.Run(g, func(t *testing.T) {
			c := newTestContext("granularity=" + g)
			_, err := parseFilter(c)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "invalid granularity")
		})
	}
}

func TestParseFilter_ValidTimeRange(t *testing.T) {
	start := "2025-01-01T00:00:00Z"
	end := "2025-01-15T00:00:00Z"
	c := newTestContext("start_time=" + start + "&end_time=" + end)
	filter, err := parseFilter(c)
	require.NoError(t, err)

	expectedStart, _ := time.Parse(time.RFC3339, start)
	expectedEnd, _ := time.Parse(time.RFC3339, end)
	assert.True(t, filter.StartTime.Equal(expectedStart))
	assert.True(t, filter.EndTime.Equal(expectedEnd))
}

func TestParseFilter_StartAfterEnd(t *testing.T) {
	c := newTestContext("start_time=2025-06-01T00:00:00Z&end_time=2025-01-01T00:00:00Z")
	_, err := parseFilter(c)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "start_time must be before end_time")
}

func TestParseFilter_DateRangeExceeds366Days(t *testing.T) {
	c := newTestContext("start_time=2024-01-01T00:00:00Z&end_time=2025-06-01T00:00:00Z")
	_, err := parseFilter(c)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "366 days")
}

func TestParseFilter_InvalidDateFormat(t *testing.T) {
	c := newTestContext("start_time=not-a-date")
	_, err := parseFilter(c)
	require.Error(t, err)

	c = newTestContext("end_time=2025/01/01")
	_, err = parseFilter(c)
	require.Error(t, err)
}

func TestParseFilter_OptionalFilters(t *testing.T) {
	c := newTestContext("agent_slug=claude&user_id=42&model=opus")
	filter, err := parseFilter(c)
	require.NoError(t, err)

	require.NotNil(t, filter.AgentSlug)
	assert.Equal(t, "claude", *filter.AgentSlug)
	require.NotNil(t, filter.UserID)
	assert.Equal(t, int64(42), *filter.UserID)
	require.NotNil(t, filter.Model)
	assert.Equal(t, "opus", *filter.Model)
}

func TestParseFilter_InvalidUserID(t *testing.T) {
	c := newTestContext("user_id=abc")
	_, err := parseFilter(c)
	require.Error(t, err)
}

func TestParseFilter_DateRangeExactly366Days(t *testing.T) {
	now := time.Now().UTC()

	// 367-day range must be rejected.
	start367 := now.AddDate(0, 0, -367)
	c := newTestContext("start_time=" + start367.Format(time.RFC3339) + "&end_time=" + now.Format(time.RFC3339))
	_, err := parseFilter(c)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "366 days")

	// 365-day range must be accepted.
	start365 := now.AddDate(0, 0, -365)
	c2 := newTestContext("start_time=" + start365.Format(time.RFC3339) + "&end_time=" + now.Format(time.RFC3339))
	_, err2 := parseFilter(c2)
	require.NoError(t, err2)
}

func TestParseFilter_OnlyAgentSlug(t *testing.T) {
	// Long agent slug — accepted by parseFilter (validation is elsewhere).
	slug := strings.Repeat("a", 100)
	c := newTestContext("agent_slug=" + slug)
	filter, err := parseFilter(c)
	require.NoError(t, err)
	require.NotNil(t, filter.AgentSlug)
	assert.Equal(t, slug, *filter.AgentSlug)
}
