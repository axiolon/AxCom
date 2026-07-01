<#
 Copyright 2026 Axiolon Labs
 SPDX-License-Identifier: Apache-2.0
#>

# Define SonarQube configuration
$SONAR_URL = "http://localhost:9000"
$SONAR_TOKEN = "sqp_770f20135ea4e236e775d15910a2614fbfbbd1e8" # User's token
$PROJECT_KEY = "ecom-engine"
$GATE_NAME = "Production-Gate"

# Encode authorization header
$bytes = [System.Text.Encoding]::UTF8.GetBytes("${SONAR_TOKEN}:")
$base64 = [System.Convert]::ToBase64String($bytes)
$headers = @{
    "Authorization" = "Basic $base64"
}

Write-Host "Checking if Quality Gate '$GATE_NAME' exists..."
try {
    $gates = Invoke-RestMethod -Uri "$SONAR_URL/api/qualitygates/show?name=$GATE_NAME" -Headers $headers -Method Get
    Write-Host "Quality Gate '$GATE_NAME' already exists."
} catch {
    Write-Host "Creating Quality Gate '$GATE_NAME'..."
    try {
        $result = Invoke-RestMethod -Uri "$SONAR_URL/api/qualitygates/create" -Headers $headers -Method Post -Body @{ name = $GATE_NAME }
        Write-Host "Successfully created Quality Gate '$GATE_NAME'."
    } catch {
        Write-Error "Failed to create Quality Gate: $_"
        exit 1
    }
}

# Define production-grade conditions
# Metrics list: 
# - coverage (Overall Code Coverage)
# - duplicated_lines_density (Duplicated Lines %)
# - reliability_rating (1 = A, 2 = B, 3 = C, etc.)
# - security_rating (1 = A, 2 = B, 3 = C, etc.)
# - sqale_rating (Maintainability: 1 = A, 2 = B, etc.)
# - security_hotspots_reviewed (Percentage of security hotspots reviewed)
$conditions = @(
    @{ metric = "coverage"; op = "LT"; error = "80" },
    @{ metric = "duplicated_lines_density"; op = "GT"; error = "3" },
    @{ metric = "reliability_rating"; op = "GT"; error = "1" },
    @{ metric = "security_rating"; op = "GT"; error = "1" },
    @{ metric = "sqale_rating"; op = "GT"; error = "1" },
    @{ metric = "security_hotspots_reviewed"; op = "LT"; error = "100" }
)

Write-Host "Configuring conditions for Quality Gate '$GATE_NAME'..."
foreach ($cond in $conditions) {
    try {
        $body = @{
            gateName = $GATE_NAME
            metric = $cond.metric
            op = $cond.op
            error = $cond.error
        }
        $res = Invoke-RestMethod -Uri "$SONAR_URL/api/qualitygates/create_condition" -Headers $headers -Method Post -Body $body
        Write-Host "Added condition: $($cond.metric) $($cond.op) $($cond.error)"
    } catch {
        # Condition might already exist, log warning but continue
        Write-Host "Note: Could not add condition for $($cond.metric) (it may already exist or metric not available)." -ForegroundColor Yellow
    }
}

Write-Host "Associating project '$PROJECT_KEY' with Quality Gate '$GATE_NAME'..."
try {
    $body = @{
        gateName = $GATE_NAME
        projectKey = $PROJECT_KEY
    }
    Invoke-RestMethod -Uri "$SONAR_URL/api/qualitygates/select" -Headers $headers -Method Post -Body $body
    Write-Host "Successfully associated project '$PROJECT_KEY' with '$GATE_NAME'."
} catch {
    Write-Error "Failed to associate project with Quality Gate: $_"
}
