Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$RepositoryRoot = Split-Path -Parent $PSScriptRoot
Push-Location $RepositoryRoot

function Write-Section {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Title
    )

    Write-Host ""
    Write-Host "============================================================"
    Write-Host "==> $Title"
    Write-Host "============================================================"
}

function Assert-NativeSuccess {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Message
    )

    if ($LASTEXITCODE -ne 0) {
        throw "$Message (exit code $LASTEXITCODE)"
    }
}

function Get-DockerApiPort {
    $port = 8081

    if (-not (Test-Path ".env")) {
        return $port
    }

    $line = Get-Content ".env" |
        Where-Object {
            $_ -match '^\s*DOCKER_API_PORT\s*='
        } |
        Select-Object -Last 1

    if ($null -eq $line) {
        return $port
    }

    $value = ($line -split '=', 2)[1].Trim().Trim('"').Trim("'")
    $parsed = 0

    if (
        [int]::TryParse($value, [ref]$parsed) -and
        $parsed -gt 0 -and
        $parsed -le 65535
    ) {
        return $parsed
    }

    throw "Invalid DOCKER_API_PORT in .env: $value"
}

function Wait-ForApiReady {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Url,

        [int]$Attempts = 60,

        [int]$DelaySeconds = 2
    )

    for ($attempt = 1; $attempt -le $Attempts; $attempt++) {
        try {
            $response = Invoke-RestMethod `
                -Uri $Url `
                -Method Get `
                -TimeoutSec 3

            if (
                $response.status -eq "ready" -and
                $response.dependencies.postgres -eq "ok" -and
                $response.dependencies.redis -eq "ok"
            ) {
                Write-Host "PASS: $Url"
                return
            }
        }
        catch {
            # Services may still be starting.
        }

        Start-Sleep -Seconds $DelaySeconds
    }

    throw "API did not become ready at $Url"
}

function Get-ContainerEnvironmentValue {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Service,

        [Parameter(Mandatory = $true)]
        [string]$Name
    )

    $output = @(
        & docker compose exec -T $Service printenv $Name
    )

    Assert-NativeSuccess "Could not read $Name from $Service"

    $value = $output |
        ForEach-Object { [string]$_ } |
        Where-Object {
            -not [string]::IsNullOrWhiteSpace($_)
        } |
        Select-Object -Last 1

    if ([string]::IsNullOrWhiteSpace($value)) {
        throw "$Name is empty or unavailable in $Service"
    }

    return $value.Trim()
}

function Invoke-PostgresScalar {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Sql
    )

    $pgUser = Get-ContainerEnvironmentValue `
        -Service "postgres" `
        -Name "POSTGRES_USER"

    $pgDatabase = Get-ContainerEnvironmentValue `
        -Service "postgres" `
        -Name "POSTGRES_DB"

    $output = @(
        & docker compose exec `
            -T `
            postgres `
            psql `
            -X `
            -v ON_ERROR_STOP=1 `
            -U $pgUser `
            -d $pgDatabase `
            -Atc $Sql
    )

    Assert-NativeSuccess "PostgreSQL query failed"

    $value = $output |
        ForEach-Object { [string]$_ } |
        Where-Object {
            -not [string]::IsNullOrWhiteSpace($_)
        } |
        Select-Object -Last 1

    if ([string]::IsNullOrWhiteSpace($value)) {
        throw "PostgreSQL query returned no scalar value"
    }

    return $value.Trim()
}

try {
    Write-Host "Commerce backend Docker validation"
    Write-Host "Repository:"
    Write-Host "  $RepositoryRoot"


    Write-Section "Docker / Compose availability"

    & docker version
    Assert-NativeSuccess "docker version failed"

    & docker compose version
    Assert-NativeSuccess "docker compose version failed"

    Write-Host "PASS: Docker and Compose are available"


    if (-not (Test-Path ".env")) {
        throw ".env is required for docker compose."
    }


    Write-Section "Compose configuration validation"

    & docker compose config --quiet
    Assert-NativeSuccess "docker compose config failed"

    Write-Host "PASS: docker compose configuration"


    Write-Section "Required services are running"

    $runningServices = @(
        & docker compose ps --status running --services
    )

    Assert-NativeSuccess "docker compose ps failed"

    $requiredServices = @(
        "postgres",
        "redis",
        "api",
        "scheduler",
        "worker"
    )

    foreach ($service in $requiredServices) {
        if ($runningServices -notcontains $service) {
            throw "Required service is not running: $service"
        }

        Write-Host "PASS: $service running"
    }


    Write-Section "API readiness"

    $apiPort = Get-DockerApiPort
    $readyUrl = "http://127.0.0.1:$apiPort/health/ready"

    Wait-ForApiReady -Url $readyUrl


    Write-Section "PostgreSQL extensions"

    $pgVector = Invoke-PostgresScalar `
        -Sql "SELECT CASE WHEN EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'vector') THEN 't' ELSE 'f' END;"

    if ($pgVector -ne "t") {
        throw "pgvector extension is not enabled"
    }

    Write-Host "PASS: pgvector enabled"


    $pgTrgm = Invoke-PostgresScalar `
        -Sql "SELECT CASE WHEN EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'pg_trgm') THEN 't' ELSE 'f' END;"

    if ($pgTrgm -ne "t") {
        throw "pg_trgm extension is not enabled"
    }

    Write-Host "PASS: pg_trgm enabled"


    Write-Section "Atlas migration state"

    $revisionCountRaw = Invoke-PostgresScalar `
        -Sql "SELECT count(*)::text FROM public.atlas_schema_revisions;"

    $revisionCount = 0

    if (
        -not [int]::TryParse(
            $revisionCountRaw,
            [ref]$revisionCount
        ) -or
        $revisionCount -le 0
    ) {
        throw "Atlas revision table is empty or unreadable: '$revisionCountRaw'"
    }

    Write-Host "PASS: Atlas revision rows = $revisionCount"


    Write-Section "Redis connectivity / queue inspection"

    & docker compose run `
        --rm `
        --no-deps `
        api `
        /app/bin/queue-inspect

    Assert-NativeSuccess "queue-inspect failed"

    Write-Host "PASS: application image can connect to Redis queue"


    Write-Section "Application container restart / recovery"

    & docker compose restart api scheduler worker

    Assert-NativeSuccess "application service restart failed"

    Wait-ForApiReady -Url $readyUrl

    $runningAfterRestart = @(
        & docker compose ps --status running --services
    )

    Assert-NativeSuccess "docker compose ps after restart failed"

    foreach ($service in @("api", "scheduler", "worker")) {
        if ($runningAfterRestart -notcontains $service) {
            throw "Service did not recover after restart: $service"
        }

        Write-Host "PASS: $service recovered"
    }


    Write-Section "Final container state"

    & docker compose ps
    Assert-NativeSuccess "docker compose ps failed"


    Write-Section "DOCKER BACKEND VALIDATION: PASS"

    Write-Host ""
    Write-Host "Validated:"
    Write-Host "  [PASS] Compose configuration"
    Write-Host "  [PASS] existing commerce runtime image"
    Write-Host "  [PASS] PostgreSQL"
    Write-Host "  [PASS] pgvector"
    Write-Host "  [PASS] pg_trgm"
    Write-Host "  [PASS] Redis"
    Write-Host "  [PASS] Atlas revision state"
    Write-Host "  [PASS] API readiness"
    Write-Host "  [PASS] scheduler process"
    Write-Host "  [PASS] worker process"
    Write-Host "  [PASS] application-to-Redis queue connectivity"
    Write-Host "  [PASS] API/scheduler/worker restart recovery"
    Write-Host ""
    Write-Host "PostgreSQL, Redis and MinIO were not rebuilt."
}
catch {
    Write-Host ""
    Write-Host "DOCKER BACKEND VALIDATION: FAIL" -ForegroundColor Red
    Write-Host $_.Exception.Message -ForegroundColor Red
    Write-Host ""

    Write-Host "Recent application container logs:"

    try {
        & docker compose logs `
            --no-color `
            --tail 200 `
            api `
            scheduler `
            worker
    }
    catch {
        Write-Host "Could not collect Docker logs."
    }

    exit 1
}
finally {
    Pop-Location
}