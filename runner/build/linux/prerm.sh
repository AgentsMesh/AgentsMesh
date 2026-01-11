#!/bin/bash
# Pre-removal script for AgentsMesh Runner

set -e

# Stop the service if running
if systemctl is-active --quiet agentsmesh-runner; then
    systemctl stop agentsmesh-runner
fi

# Disable the service
if systemctl is-enabled --quiet agentsmesh-runner 2>/dev/null; then
    systemctl disable agentsmesh-runner
fi

echo "AgentsMesh Runner service stopped and disabled."
