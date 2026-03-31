package notification

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPreference_IsChannelEnabled(t *testing.T) {
	tests := []struct {
		name     string
		pref     *Preference
		channel  string
		expected bool
	}{
		{
			name:     "nil channels map returns false",
			pref:     &Preference{Channels: nil},
			channel:  ChannelToast,
			expected: false,
		},
		{
			name:     "empty channels map returns false",
			pref:     &Preference{Channels: map[string]bool{}},
			channel:  ChannelToast,
			expected: false,
		},
		{
			name:     "enabled channel returns true",
			pref:     &Preference{Channels: map[string]bool{ChannelToast: true}},
			channel:  ChannelToast,
			expected: true,
		},
		{
			name:     "disabled channel returns false",
			pref:     &Preference{Channels: map[string]bool{ChannelToast: false}},
			channel:  ChannelToast,
			expected: false,
		},
		{
			name:     "missing channel returns false",
			pref:     &Preference{Channels: map[string]bool{ChannelBrowser: true}},
			channel:  ChannelToast,
			expected: false,
		},
		{
			name:     "browser channel enabled",
			pref:     &Preference{Channels: map[string]bool{ChannelBrowser: true}},
			channel:  ChannelBrowser,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.pref.IsChannelEnabled(tt.channel)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDefaultPreference(t *testing.T) {
	pref := DefaultPreference()

	assert.NotNil(t, pref)
	assert.False(t, pref.IsMuted)
	assert.NotNil(t, pref.Channels)

	// All builtin client channels should be enabled by default
	for ch := range BuiltinClientChannels {
		assert.True(t, pref.IsChannelEnabled(ch), "expected %s to be enabled", ch)
	}

	assert.True(t, pref.IsChannelEnabled(ChannelToast))
	assert.True(t, pref.IsChannelEnabled(ChannelBrowser))
}

func TestChannelsJSON_ScanAndValue(t *testing.T) {
	// Scan from []byte
	var cj ChannelsJSON
	err := cj.Scan([]byte(`{"toast":true,"browser":false}`))
	assert.NoError(t, err)
	assert.True(t, cj["toast"])
	assert.False(t, cj["browser"])

	// Scan from string
	var cj2 ChannelsJSON
	err = cj2.Scan(`{"toast":false}`)
	assert.NoError(t, err)
	assert.False(t, cj2["toast"])

	// Scan from nil
	var cj3 ChannelsJSON
	err = cj3.Scan(nil)
	assert.NoError(t, err)
	assert.Nil(t, cj3)

	// Scan from unsupported type
	var cj4 ChannelsJSON
	err = cj4.Scan(42)
	assert.Error(t, err)

	// Value round-trip
	original := ChannelsJSON{"toast": true, "browser": false}
	val, err := original.Value()
	assert.NoError(t, err)
	assert.NotNil(t, val)

	// Nil value
	var nilCJ ChannelsJSON
	val, err = nilCJ.Value()
	assert.NoError(t, err)
	assert.Nil(t, val)
}

func TestConstants(t *testing.T) {
	// Verify source constants are non-empty strings
	assert.NotEmpty(t, SourceChannelMessage)
	assert.NotEmpty(t, SourceChannelMention)
	assert.NotEmpty(t, SourceTerminalOSC)
	assert.NotEmpty(t, SourceTaskCompleted)

	// Priority constants
	assert.Equal(t, "normal", PriorityNormal)
	assert.Equal(t, "high", PriorityHigh)

	// Builtin channels
	assert.True(t, BuiltinClientChannels[ChannelToast])
	assert.True(t, BuiltinClientChannels[ChannelBrowser])
}
