//go:build e2e
// +build e2e

package e2e

import "testing"

func TestReadFilePlaceholder(t *testing.T) {
	t.Skip("e2e tests require Files service + Gateway deployments")
}
