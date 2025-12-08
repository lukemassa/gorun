package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValueFromEnv(t *testing.T) {
	cases := []struct {
		description   string
		env           []string
		key           string
		expectedValue string
	}{
		{
			description: "value is present",
			env: []string{
				"key=value",
			},
			key:           "key",
			expectedValue: "value",
		},
		{
			description:   "value is absent",
			env:           []string{},
			key:           "key",
			expectedValue: "",
		},
		{
			description: "key does not have equal sign",
			env: []string{
				"key",
			},
			key:           "key",
			expectedValue: "",
		},
		{
			description: "env has multiple equal signs",
			env: []string{
				"key=value=foo",
			},
			key:           "key",
			expectedValue: "value=foo",
		},
	}
	for _, tc := range cases {
		t.Run(tc.description, func(t *testing.T) {
			actualValue := valueFromEnv(tc.key, tc.env)
			assert.Equal(t, tc.expectedValue, actualValue)
		})
	}
}
