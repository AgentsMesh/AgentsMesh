#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "$0")" && pwd)"
export PYTHONDONTWRITEBYTECODE=1

python3 "$script_dir/validate_skills_test.py"
python3 "$script_dir/validate_skills.py"
