package runner

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/anthropics/agentsmesh/runner/internal/relay"
	"github.com/anthropics/agentsmesh/runner/internal/terminal"
	"github.com/anthropics/agentsmesh/runner/internal/terminal/aggregator"
	"github.com/anthropics/agentsmesh/runner/internal/terminal/vt"
	"github.com/anthropics/agentsmesh/runner/internal/testutil"
)

func TestAssemblePTYRuntimeWiresSharedComponents(t *testing.T) {
	const marker = "assembled-output-handler"
	command, args := testutil.EchoCommand(marker)
	term, err := terminal.New(terminal.Options{
		Command: command,
		Args:    args,
		Cols:    80,
		Rows:    24,
	})
	require.NoError(t, err)

	virtualTerm := vt.NewVirtualTerminal(80, 24, 100)
	agg := aggregator.NewSmartAggregator(nil)
	localRelay := relay.NewLocalServer(nil)
	pod := &Pod{PodKey: "assembly-pod"}

	pod.installRuntime(assemblePTYRuntime(pod, term, virtualTerm, agg, nil, localRelay))
	t.Cleanup(func() {
		pod.IO.Stop()
		pod.IO.Teardown()
	})

	ptyIO, ok := pod.IO.(*PTYPodIO)
	require.True(t, ok)
	comps := ptyIO.components
	require.Same(t, term, comps.Terminal)
	require.Same(t, virtualTerm, comps.VirtualTerminal)
	require.Same(t, agg, comps.Aggregator)
	require.Same(t, virtualTerm, pod.vtProvider())
	require.Zero(t, term.PID(), "assembly must not start the terminal")

	ptyRelay, ok := pod.Relay.(*PTYPodRelay)
	require.True(t, ok)
	require.Same(t, comps, ptyRelay.components)
	require.Same(t, localRelay, ptyRelay.localServer)
	require.NotNil(t, ptyRelay.localWriter)

	health := agg.OutputHealth()
	require.Len(t, health, 1)
	require.Equal(t, aggregator.OutputDestinationLocal, health[0].Destination)

	require.NoError(t, pod.IO.Start())
	require.Eventually(t, func() bool {
		return strings.Contains(virtualTerm.GetOutput(5), marker)
	}, 2*time.Second, 10*time.Millisecond, "installed output handler did not feed the VT")
}

func TestAssemblePTYRuntimeNormalizesTypedNilLocalRelay(t *testing.T) {
	term, err := terminal.New(terminal.Options{Command: "not-started"})
	require.NoError(t, err)

	virtualTerm := vt.NewVirtualTerminal(80, 24, 100)
	agg := aggregator.NewSmartAggregator(nil)
	t.Cleanup(agg.Stop)
	pod := &Pod{PodKey: "typed-nil-pod"}

	var server *relay.LocalServer
	var broker LocalRelayBroker = server

	pod.installRuntime(assemblePTYRuntime(pod, term, virtualTerm, agg, nil, broker))
	ptyRelay, ok := pod.Relay.(*PTYPodRelay)
	require.True(t, ok)
	if ptyRelay.localServer != nil {
		t.Fatal("typed-nil broker was retained as an active local destination")
	}
	require.Nil(t, ptyRelay.localWriter)
	require.Empty(t, agg.OutputHealth())
}
