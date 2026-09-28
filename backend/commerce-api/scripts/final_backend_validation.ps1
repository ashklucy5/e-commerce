$ErrorActionPreference = "Stop"

Set-StrictMode -Version Latest

# ============================================================
# Final Commerce Backend Validation
#
# Expected workflow:
#
#   1. Stop Air before doing a mass gofmt.
#   2. Format all Go files once:
#
#      Get-ChildItem -Recurse -Filter *.go -File |
#          ForEach-Object { gofmt -w $_.FullName }
#
#   3. Start Air again and make sure exactly one API instance
#      is listening on the configured E2E_BASE_URL.
#
#   4. Run:
#
#      .\scripts\final_backend_validation.ps1
#
# This script intentionally VERIFYs formatting instead of
# rewriting source files, preventing an Air restart storm.
# ============================================================

function Write-Step {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name
    )

    Write-Host ""
    Write-Host "============================================================"
    Write-Host "==> $Name"
    Write-Host "============================================================"
}

function Invoke-GoStep {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name,

        [Parameter(Mandatory = $true)]
        [scriptblock]$Command
    )

    Write-Step $Name

    & $Command

    if ($LASTEXITCODE -ne 0) {
        throw "$Name failed with exit code $LASTEXITCODE"
    }

    Write-Host "PASS: $Name"
}

function Restore-EnvironmentVariable {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name,

        [AllowNull()]
        [string]$PreviousValue
    )

    if ($null -eq $PreviousValue) {
        Remove-Item `
            -Path "Env:$Name" `
            -ErrorAction SilentlyContinue

        return
    }

    Set-Item `
        -Path "Env:$Name" `
        -Value $PreviousValue
}

function Test-APIReady {
    param(
        [Parameter(Mandatory = $true)]
        [string]$BaseURL
    )

    $healthURL =
        $BaseURL.TrimEnd("/") +
        "/health/ready"

    Write-Step "live API readiness"

    Write-Host "Checking:"
    Write-Host "  $healthURL"

    try {
        $response =
            Invoke-WebRequest `
                -Uri $healthURL `
                -Method GET `
                -TimeoutSec 10 `
                -UseBasicParsing

        if ($response.StatusCode -ne 200) {
            throw "health endpoint returned HTTP $($response.StatusCode)"
        }

        Write-Host "PASS: live API is ready"
    }
    catch {
        Write-Host ""
        Write-Host "The external black-box E2E requires one running API instance."
        Write-Host ""
        Write-Host "Make sure Air is running and the API is listening at:"
        Write-Host "  $BaseURL"
        Write-Host ""
        Write-Host "If another process owns port 8081, inspect it with:"
        Write-Host ""
        Write-Host "  Get-NetTCPConnection -LocalPort 8081 -State Listen"
        Write-Host ""

        throw "live API readiness check failed: $($_.Exception.Message)"
    }
}

# ------------------------------------------------------------
# Resolve repository root.
# scripts/final_backend_validation.ps1 -> repository root
# ------------------------------------------------------------

$root =
    Resolve-Path(
        Join-Path `
            $PSScriptRoot `
            ".."
    )

$validationDirectory =
    Join-Path `
        $root `
        "tmp\final-backend-validation"

Push-Location $root

try {
    Write-Host ""
    Write-Host "Commerce backend final validation"
    Write-Host "Repository:"
    Write-Host "  $root"
    Write-Host ""

    # ========================================================
    # 1. Verify formatting across ALL Go source files
    # ========================================================

    Write-Step "gofmt verification for all Go files"

    $goFiles =
        Get-ChildItem `
            -Path $root `
            -Recurse `
            -Filter "*.go" `
            -File |
        Where-Object {
            $_.FullName -notmatch "\\tmp\\"
        }

    if ($goFiles.Count -eq 0) {
        throw "no Go source files found"
    }

    $unformatted =
        New-Object `
            System.Collections.Generic.List[string]

    foreach ($file in $goFiles) {
        $result =
            & gofmt -l $file.FullName

        if ($LASTEXITCODE -ne 0) {
            throw "gofmt failed while checking $($file.FullName)"
        }

        foreach ($line in $result) {
            if (
                -not [string]::IsNullOrWhiteSpace(
                    $line
                )
            ) {
                $unformatted.Add(
                    $line
                )
            }
        }
    }

    if ($unformatted.Count -gt 0) {
        Write-Host ""
        Write-Host "The following Go files are not formatted:"

        foreach ($file in $unformatted) {
            Write-Host "  $file"
        }

        Write-Host ""
        Write-Host "Stop Air first, then run:"
        Write-Host ""
        Write-Host '  Get-ChildItem -Recurse -Filter *.go -File | ForEach-Object { gofmt -w $_.FullName }'
        Write-Host ""
        Write-Host "Then restart Air and rerun this validator."

        throw "gofmt verification failed"
    }

    Write-Host "PASS: all $($goFiles.Count) Go files are formatted"

    # ========================================================
    # 2. Module graph consistency
    # ========================================================

    Invoke-GoStep `
        "go mod tidy consistency" `
        {
            go mod tidy -diff
        }

    # ========================================================
    # 3. Static analysis
    # ========================================================

    Invoke-GoStep `
        "go vet ./..." `
        {
            go vet ./...
        }

    # ========================================================
    # 4. Compile and run ALL package tests
    #
    # Live opt-in tests remain skipped here because their
    # environment variables are enabled later individually.
    # ========================================================

    Invoke-GoStep `
        "go test ./..." `
        {
            go test ./... -count=1
        }

    # ========================================================
    # 5. Build ALL cmd/* programs
    #
    # This dynamically finds every direct Go command under cmd
    # instead of maintaining a fragile hand-written list.
    # ========================================================

    Write-Step "build every cmd program"

    if (
        Test-Path `
            -Path $validationDirectory
    ) {
        Remove-Item `
            -Path $validationDirectory `
            -Recurse `
            -Force
    }

    New-Item `
        -ItemType Directory `
        -Path $validationDirectory `
        -Force |
        Out-Null

    $commandDirectories =
        Get-ChildItem `
            -Path (
                Join-Path `
                    $root `
                    "cmd"
            ) `
            -Directory |
        Sort-Object Name

    if ($commandDirectories.Count -eq 0) {
        throw "no cmd programs found"
    }

    foreach ($commandDirectory in $commandDirectories) {
        $goSource =
            Get-ChildItem `
                -Path $commandDirectory.FullName `
                -Filter "*.go" `
                -File `
                -ErrorAction SilentlyContinue

        if ($null -eq $goSource) {
            continue
        }

        $packagePath =
            "./cmd/" +
            $commandDirectory.Name

        $outputPath =
            Join-Path `
                $validationDirectory `
                (
                    $commandDirectory.Name +
                    ".exe"
                )

        Write-Host ""
        Write-Host "Building $packagePath"

        & go build `
            -o $outputPath `
            $packagePath

        if ($LASTEXITCODE -ne 0) {
            throw "build failed: $packagePath"
        }

        Write-Host "PASS: $packagePath"
    }

    # ========================================================
    # 6. Automatic migration runtime validation
    # ========================================================

    $oldMigrationFlag =
        [Environment]::GetEnvironmentVariable(
            "RUN_DB_MIGRATION_TEST",
            "Process"
        )

    $env:RUN_DB_MIGRATION_TEST =
        "1"

    try {
        Invoke-GoStep `
            "live automatic migration validation" `
            {
                go test `
                    ./internal/platform/database `
                    -run "TestApplyMigrationsLive|TestEmbeddedMigrationsValidate|TestMigrationPostgresURLUsesPublicSearchPath" `
                    -v `
                    -count=1
            }
    }
    finally {
        Restore-EnvironmentVariable `
            -Name "RUN_DB_MIGRATION_TEST" `
            -PreviousValue $oldMigrationFlag
    }

    # ========================================================
    # 7. PostgreSQL pgvector provisioning
    #
    # OpenAI is NOT required for this test.
    # This only verifies PostgreSQL extension provisioning.
    # ========================================================

    $oldPgvectorFlag =
        [Environment]::GetEnvironmentVariable(
            "RUN_PGVECTOR_TEST",
            "Process"
        )

    $env:RUN_PGVECTOR_TEST =
        "1"

    try {
        Invoke-GoStep `
            "PostgreSQL pgvector provisioning" `
            {
                go test `
                    ./internal/search `
                    -run "TestPgvectorProvisioning" `
                    -v `
                    -count=1
            }
    }
    finally {
        Restore-EnvironmentVariable `
            -Name "RUN_PGVECTOR_TEST" `
            -PreviousValue $oldPgvectorFlag
    }

    # ========================================================
    # 8. Scheduler -> Redis -> worker integration
    # ========================================================

    $oldQueuePipelineFlag =
        [Environment]::GetEnvironmentVariable(
            "RUN_QUEUE_PIPELINE_TEST",
            "Process"
        )

    $env:RUN_QUEUE_PIPELINE_TEST =
        "1"

    try {
        Invoke-GoStep `
            "scheduler -> Redis -> worker queue pipeline" `
            {
                go test `
                    ./tests/integration/queuepipeline `
                    -run "TestSchedulerWorkerQueuePipeline" `
                    -v `
                    -count=1
            }
    }
    finally {
        Restore-EnvironmentVariable `
            -Name "RUN_QUEUE_PIPELINE_TEST" `
            -PreviousValue $oldQueuePipelineFlag
    }

    # ========================================================
    # 9. External live API E2E
    #
    # IMPORTANT:
    # The API must already be running.
    # The validator does NOT launch a second API process.
    # This prevents the :8081 collision we saw with Air.
    # ========================================================

    $baseURL =
        [Environment]::GetEnvironmentVariable(
            "E2E_BASE_URL",
            "Process"
        )

    if (
        [string]::IsNullOrWhiteSpace(
            $baseURL
        )
    ) {
        $baseURL =
            "http://127.0.0.1:8081"
    }

    $baseURL =
        $baseURL.TrimEnd("/")

    Test-APIReady `
        -BaseURL $baseURL

    # --------------------------------------------------------
    # Basic route preflight
    #
    # This specifically catches a missing #14 route before
    # running the longer black-box flow.
    # A 400 here is acceptable because the route exists but
    # the query is intentionally empty.
    #
    # 404 means the route was not registered.
    # --------------------------------------------------------

    Write-Step "storefront route preflight"

    $searchProductsURL =
        $baseURL +
        "/api/v1/search/products"

    try {
        $response =
            Invoke-WebRequest `
                -Uri $searchProductsURL `
                -Method GET `
                -TimeoutSec 10 `
                -UseBasicParsing

        $statusCode =
            [int]$response.StatusCode
    }
    catch {
        if (
            $null -ne $_.Exception.Response
        ) {
            $statusCode =
                [int]$_.Exception.Response.StatusCode
        }
        else {
            throw "unable to check $searchProductsURL : $($_.Exception.Message)"
        }
    }

    if ($statusCode -eq 404) {
        throw "GET /api/v1/search/products is not registered"
    }

    Write-Host "PASS: GET /api/v1/search/products is registered (HTTP $statusCode)"

    # --------------------------------------------------------
    # Promotion route preflight
    # --------------------------------------------------------

    $promotionURL =
        $baseURL +
        "/api/v1/promotions/active?currency=BDT"

    try {
        $response =
            Invoke-WebRequest `
                -Uri $promotionURL `
                -Method GET `
                -TimeoutSec 10 `
                -UseBasicParsing

        if ($response.StatusCode -ne 200) {
            throw "promotion endpoint returned HTTP $($response.StatusCode)"
        }

        Write-Host "PASS: GET /api/v1/promotions/active"
    }
    catch {
        throw "storefront promotion endpoint failed: $($_.Exception.Message)"
    }

    # ========================================================
    # 10. Full external black-box E2E
    # ========================================================

    $oldBlackboxFlag =
        [Environment]::GetEnvironmentVariable(
            "RUN_BLACKBOX_E2E",
            "Process"
        )

    $oldE2EBaseURL =
        [Environment]::GetEnvironmentVariable(
            "E2E_BASE_URL",
            "Process"
        )

    $env:RUN_BLACKBOX_E2E =
        "1"

    $env:E2E_BASE_URL =
        $baseURL

    try {
        Invoke-GoStep `
            "external storefront/customer/guest black-box E2E" `
            {
                go test `
                    ./tests/e2e `
                    -run "TestBackendBlackBox" `
                    -v `
                    -count=1
            }
    }
    finally {
        Restore-EnvironmentVariable `
            -Name "RUN_BLACKBOX_E2E" `
            -PreviousValue $oldBlackboxFlag

        Restore-EnvironmentVariable `
            -Name "E2E_BASE_URL" `
            -PreviousValue $oldE2EBaseURL
    }

    # ========================================================
    # Success
    # ========================================================

    Write-Host ""
    Write-Host "============================================================"
    Write-Host " FINAL BACKEND VALIDATION: PASS"
    Write-Host "============================================================"
    Write-Host ""
    Write-Host "Validated:"
    Write-Host "  [PASS] all Go source formatting"
    Write-Host "  [PASS] go.mod/go.sum consistency"
    Write-Host "  [PASS] go vet across repository"
    Write-Host "  [PASS] all Go tests"
    Write-Host "  [PASS] every cmd/* program builds"
    Write-Host "  [PASS] automatic database migrations"
    Write-Host "  [PASS] PostgreSQL pgvector provisioning"
    Write-Host "  [PASS] scheduler -> Redis -> worker pipeline"
    Write-Host "  [PASS] storefront promotion route"
    Write-Host "  [PASS] filtered/faceted search route"
    Write-Host "  [PASS] anonymous storefront browsing"
    Write-Host "  [PASS] authenticated account boundaries"
    Write-Host "  [PASS] secure guest order/tracking access"
    Write-Host "  [PASS] external HTTP black-box flow"
    Write-Host ""
}
catch {
    Write-Host ""
    Write-Host "============================================================"
    Write-Host " FINAL BACKEND VALIDATION: FAILED"
    Write-Host "============================================================"
    Write-Host ""
    Write-Host $_.Exception.Message
    Write-Host ""

    exit 1
}
finally {
    if (
        Test-Path `
            -Path $validationDirectory
    ) {
        Remove-Item `
            -Path $validationDirectory `
            -Recurse `
            -Force `
            -ErrorAction SilentlyContinue
    }

    Pop-Location
}