#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
configs=(
  "deploy/selfhost/traefik/dynamic/routes.yml"
  "deploy/onpremise/traefik/dynamic/routes.yml"
)

cd "$repo_root"
ruby -ryaml - "${configs[@]}" <<'RUBY'
EXPECTED_RULE = 'PathPrefix(`/api`) || PathPrefix(`/health`) || PathPrefix(`/proto.`)'

def value_at(document, *keys)
  keys.reduce(document) do |value, key|
    value.is_a?(Hash) ? value[key] : nil
  end
end

def assert_equal(path, field, actual, expected)
  return if actual == expected

  abort "#{path}: #{field}: expected #{expected.inspect}, got #{actual.inspect}"
end

ARGV.each do |path|
  begin
    document = YAML.safe_load_file(path, aliases: false)
  rescue Psych::Exception => error
    abort "#{path}: yaml: #{error.message}"
  end

  api_rule = value_at(document, "http", "routers", "api", "rule")
  api_service = value_at(document, "http", "routers", "api", "service")
  api_priority = value_at(document, "http", "routers", "api", "priority")
  web_priority = value_at(document, "http", "routers", "web", "priority")

  assert_equal(path, "http.routers.api.rule", api_rule, EXPECTED_RULE)
  assert_equal(path, "http.routers.api.service", api_service, "backend")
  assert_equal(path, "http.routers.api.priority", api_priority, 100)

  unless web_priority.is_a?(Numeric) && web_priority < api_priority
    abort "#{path}: http.routers.web.priority: expected a number lower than #{api_priority.inspect}, got #{web_priority.inspect}"
  end

  puts "validated #{path}"
end
RUBY
