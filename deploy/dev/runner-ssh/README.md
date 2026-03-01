# Runner SSH Keys

This directory contains SSH keys used by the dev runner to access Git repositories.

## Files

- `id_ed25519` - Private key (**do not commit to Git**)
- `id_ed25519.pub` - Public key (can be committed, for reference)
- `config` - SSH client configuration
- `known_hosts` - Known hosts list

## Generate New Keys

If the private key is lost or needs to be regenerated:

```bash
# Generate a new ED25519 key
ssh-keygen -t ed25519 -C "agentsmesh-dev-runner@local" -f ./id_ed25519 -N ""

# Generate known_hosts
ssh-keyscan -p 2222 gitlab.corp.signalrender.com > known_hosts
```

## Configure on GitLab

Add the public key as a Deploy Key for the project:

```bash
PUBKEY=$(cat id_ed25519.pub)
GITLAB_HOST=gitlab.corp.signalrender.com glab api -X POST projects/12/deploy_keys \
  -f title="AgentsMesh Dev Runner" \
  -f key="$PUBKEY" \
  -f can_push=true
```

Or via the GitLab Web UI:
1. Go to project Settings > Repository > Deploy Keys
2. Add the public key content
3. Check "Grant write permissions to this key"

## Test Connection

```bash
# Test inside the runner container
docker compose exec runner ssh -T git@gitlab.corp.signalrender.com
```
