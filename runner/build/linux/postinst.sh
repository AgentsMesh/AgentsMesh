#!/bin/bash
# Post-installation script for AgentsMesh Runner

set -e

# Create agentsmesh user if it doesn't exist
if ! id "agentsmesh" &>/dev/null; then
    useradd --system --no-create-home --shell /bin/false agentsmesh
fi

# Create directories
mkdir -p /var/lib/agentsmesh
mkdir -p /var/log/agentsmesh
mkdir -p /etc/agentsmesh

# Set permissions
chown -R agentsmesh:agentsmesh /var/lib/agentsmesh
chown -R agentsmesh:agentsmesh /var/log/agentsmesh
chmod 755 /var/lib/agentsmesh
chmod 755 /var/log/agentsmesh

# Reload systemd
systemctl daemon-reload

echo ""
echo "AgentsMesh Runner has been installed."
echo ""
echo "Next steps:"
echo "  1. Register the runner:"
echo "     sudo -u agentsmesh runner register --server <URL> --token <TOKEN>"
echo ""
echo "  2. Start the service:"
echo "     sudo systemctl start agentsmesh-runner"
echo ""
echo "  3. Enable auto-start at boot:"
echo "     sudo systemctl enable agentsmesh-runner"
echo ""
