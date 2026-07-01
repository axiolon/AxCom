#!/usr/bin/env bash
# Copyright 2026 Axiolon Labs
# SPDX-License-Identifier: Apache-2.0


# Exit on error
set -e

SONAR_URL="http://localhost:9000"
SONAR_TOKEN="squ_1f0963b0fa6e6e168c16d4e5b07f7fba6231934d"
PROJECT_KEY="ecom-engine"
GATE_NAME="Production-Gate"

echo "Checking if Quality Gate '$GATE_NAME' exists..."
if curl -s -u "${SONAR_TOKEN}:" "${SONAR_URL}/api/qualitygates/show?name=${GATE_NAME}" | grep -q '"name"'; then
  echo "Quality Gate '$GATE_NAME' already exists."
else
  echo "Creating Quality Gate '$GATE_NAME'..."
  curl -s -u "${SONAR_TOKEN}:" -X POST "${SONAR_URL}/api/qualitygates/create" -d "name=${GATE_NAME}"
  echo "Successfully created Quality Gate '$GATE_NAME'."
fi

# Define production-grade conditions
# Metrics list:
# - coverage (Overall Code Coverage)
# - duplicated_lines_density (Duplicated Lines %)
# - reliability_rating (1 = A, 2 = B, 3 = C, etc.)
# - security_rating (1 = A, 2 = B, 3 = C, etc.)
# - sqale_rating (Maintainability: 1 = A, 2 = B, etc.)
# - security_hotspots_reviewed (Percentage of security hotspots reviewed)
declare -A conditions=(
  ["coverage"]="LT:80"
  ["duplicated_lines_density"]="GT:3"
  ["reliability_rating"]="GT:1"
  ["security_rating"]="GT:1"
  ["sqale_rating"]="GT:1"
  ["security_hotspots_reviewed"]="LT:100"
)

echo "Configuring conditions for Quality Gate '$GATE_NAME'..."
for metric in "${!conditions[@]}"; do
  val=${conditions[$metric]}
  op=$(echo $val | cut -d: -f1)
  error=$(echo $val | cut -d: -f2)

  echo "Adding condition: $metric $op $error"
  curl -s -u "${SONAR_TOKEN}:" -X POST "${SONAR_URL}/api/qualitygates/create_condition" \
    -d "gateName=${GATE_NAME}" \
    -d "metric=${metric}" \
    -d "op=${op}" \
    -d "error=${error}" || echo "Note: Could not add condition for $metric (it may already exist)."
done

echo "Associating project '$PROJECT_KEY' with Quality Gate '$GATE_NAME'..."
curl -s -u "${SONAR_TOKEN}:" -X POST "${SONAR_URL}/api/qualitygates/select" \
  -d "gateName=${GATE_NAME}" \
  -d "projectKey=${PROJECT_KEY}"

echo "Successfully associated project '$PROJECT_KEY' with '$GATE_NAME'."
