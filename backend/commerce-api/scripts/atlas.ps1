$ErrorActionPreference = "Stop"

$projectRoot =
    Split-Path -Parent $PSScriptRoot

$envFile =
    Join-Path $projectRoot ".env"

if (-not (Test-Path $envFile)) {
    throw ".env not found at $envFile"
}

# ------------------------------------------------------------
# Load .env into the current process.
# ------------------------------------------------------------

Get-Content $envFile | ForEach-Object {
    $line =
        $_.Trim()

    if (
        $line.Length -eq 0 -or
        $line.StartsWith("#")
    ) {
        return
    }

    $parts =
        $line -split "=", 2

    if ($parts.Count -ne 2) {
        return
    }

    $key =
        $parts[0].Trim()

    $value =
        $parts[1].Trim()

    if (
        $value.Length -ge 2 -and
        (
            (
                $value.StartsWith('"') -and
                $value.EndsWith('"')
            ) -or
            (
                $value.StartsWith("'") -and
                $value.EndsWith("'")
            )
        )
    ) {
        $value =
            $value.Substring(
                1,
                $value.Length - 2
            )
    }

    [Environment]::SetEnvironmentVariable(
        $key,
        $value,
        "Process"
    )
}

# ------------------------------------------------------------
# Environment helpers.
# ------------------------------------------------------------

function Get-ProcessEnv {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name,

        [string]$Default = ""
    )

    $value =
        [Environment]::GetEnvironmentVariable(
            $Name,
            "Process"
        )

    if (
        [string]::IsNullOrWhiteSpace(
            $value
        )
    ) {
        return $Default
    }

    return $value.Trim()
}

function Require-ProcessEnv {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name
    )

    $value =
        Get-ProcessEnv `
            -Name $Name

    if (
        [string]::IsNullOrWhiteSpace(
            $value
        )
    ) {
        throw "Required environment variable $Name is not set."
    }

    return $value
}

function Validate-Port {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name,

        [Parameter(Mandatory = $true)]
        [string]$Value
    )

    $parsed =
        0

    if (
        -not [int]::TryParse(
            $Value,
            [ref]$parsed
        )
    ) {
        throw "$Name must be a valid TCP port."
    }

    if (
        $parsed -lt 1 -or
        $parsed -gt 65535
    ) {
        throw "$Name must be between 1 and 65535."
    }

    return $parsed
}

function New-PostgresURL {
    param(
        [Parameter(Mandatory = $true)]
        [string]$DbHost,

        [Parameter(Mandatory = $true)]
        [int]$Port,

        [Parameter(Mandatory = $true)]
        [string]$Database,

        [Parameter(Mandatory = $true)]
        [string]$User,

        [Parameter(Mandatory = $true)]
        [string]$Password,

        [Parameter(Mandatory = $true)]
        [string]$SSLMode
    )

    $escapedUser =
        [Uri]::EscapeDataString(
            $User
        )

    $escapedPassword =
        [Uri]::EscapeDataString(
            $Password
        )

    $escapedDatabase =
        [Uri]::EscapeDataString(
            $Database
        )

    $escapedSSLMode =
        [Uri]::EscapeDataString(
            $SSLMode
        )

    return (
        "postgres://${escapedUser}:${escapedPassword}" +
        "@${DbHost}:${Port}/${escapedDatabase}" +
        "?sslmode=${escapedSSLMode}&search_path=public"
    )
}
# ------------------------------------------------------------
# Detect Atlas operation.
# ------------------------------------------------------------

$isMigrateApply = (
    $args.Count -ge 2 -and
    $args[0] -eq "migrate" -and
    $args[1] -eq "apply"
)

$isMigrateStatus = (
    $args.Count -ge 2 -and
    $args[0] -eq "migrate" -and
    $args[1] -eq "status"
)

$isMigrateDiff = (
    $args.Count -ge 2 -and
    $args[0] -eq "migrate" -and
    $args[1] -eq "diff"
)

$isDryRun =
    $args -contains "--dry-run"

# ------------------------------------------------------------
# Atlas disposable development database.
#
# This is used by `atlas migrate diff`.
# It is never the application database.
# ------------------------------------------------------------

$atlasDevContainer =
    "commerce-atlas-dev"

$atlasDevImage =
    "pgvector/pgvector:0.8.6-pg18-bookworm"

$atlasDevHost =
    "127.0.0.1"

$atlasDevPort =
    "55432"

$atlasDevDatabase =
    "dev"

$atlasDevURL =
    "postgres://postgres@${atlasDevHost}:${atlasDevPort}/${atlasDevDatabase}?sslmode=disable&search_path=public"

function Ensure-AtlasDevDatabase {
    Write-Host ""
    Write-Host "============================================================"
    Write-Host " Atlas development database"
    Write-Host "============================================================"
    Write-Host ""
    Write-Host " Container : $atlasDevContainer"
    Write-Host " Image     : $atlasDevImage"
    Write-Host " Host      : $atlasDevHost"
    Write-Host " Port      : $atlasDevPort"
    Write-Host " Database  : $atlasDevDatabase"
    Write-Host ""

    & docker info *> $null

    if ($LASTEXITCODE -ne 0) {
        throw "Docker is not available. Start Docker Desktop and try again."
    }

    $containerName = (
        & docker ps -a `
            --filter "name=^/${atlasDevContainer}$" `
            --format "{{.Names}}"
    )

    if ($LASTEXITCODE -ne 0) {
        throw "Unable to inspect Atlas development container."
    }

    $containerExists =
        -not [string]::IsNullOrWhiteSpace(
            (
                $containerName |
                    Out-String
            ).Trim()
        )

    if ($containerExists) {
        $configuredImage = (
            & docker inspect `
                --format "{{.Config.Image}}" `
                $atlasDevContainer
        ).Trim()

        if ($LASTEXITCODE -ne 0) {
            throw "Unable to inspect Atlas development container."
        }

        if (
            $configuredImage -ne
            $atlasDevImage
        ) {
            throw @"
Atlas development container '$atlasDevContainer'
uses an unexpected image.

Expected:
$atlasDevImage

Actual:
$configuredImage

Remove or rename the container before continuing.
"@
        }

        $isRunning = (
            & docker inspect `
                --format "{{.State.Running}}" `
                $atlasDevContainer
        ).Trim()

        if ($LASTEXITCODE -ne 0) {
            throw "Unable to inspect Atlas development container state."
        }

        if ($isRunning -ne "true") {
            Write-Host "Starting existing Atlas development database..."

            & docker start `
                $atlasDevContainer *> $null

            if ($LASTEXITCODE -ne 0) {
                throw "Unable to start Atlas development container."
            }
        }
        else {
            Write-Host "Atlas development database is already running."
        }
    }
    else {
        Write-Host "Creating Atlas development database..."

        & docker run `
            --rm `
            -d `
            --name $atlasDevContainer `
            -e "POSTGRES_HOST_AUTH_METHOD=trust" `
            -e "POSTGRES_DB=$atlasDevDatabase" `
            -p "${atlasDevHost}:${atlasDevPort}:5432" `
            $atlasDevImage *> $null

        if ($LASTEXITCODE -ne 0) {
            throw "Unable to create Atlas development container."
        }
    }

    $portMapping = (
        & docker port `
            $atlasDevContainer `
            "5432/tcp"
    )

    if ($LASTEXITCODE -ne 0) {
        throw "Unable to inspect Atlas development database port mapping."
    }

    $portMappingText = (
        $portMapping |
            Out-String
    ).Trim()

    if (
        $portMappingText -notmatch
        [regex]::Escape(
            "${atlasDevHost}:${atlasDevPort}"
        )
    ) {
        throw @"
Atlas development container is not mapped to the expected port.

Expected:
${atlasDevHost}:${atlasDevPort}

Actual:
$portMappingText
"@
    }

    Write-Host "Waiting for Atlas development PostgreSQL..."

    $ready =
        $false

    for (
        $attempt = 1
        $attempt -le 30
        $attempt++
    ) {
        & docker exec `
            $atlasDevContainer `
            pg_isready `
            -U postgres `
            -d $atlasDevDatabase *> $null

        if ($LASTEXITCODE -eq 0) {
            $ready =
                $true

            break
        }

        Start-Sleep -Seconds 1
    }

    if (-not $ready) {
        Write-Host ""
        Write-Host "Atlas development database did not become ready."
        Write-Host ""
        Write-Host "Recent container logs:"
        Write-Host ""

        & docker logs `
            --tail 50 `
            $atlasDevContainer

        throw "Atlas development database readiness timeout."
    }

    Write-Host "Atlas development database is ready."
    Write-Host ""
    Write-Host "============================================================"
    Write-Host ""
}

# ------------------------------------------------------------
# Local PostgreSQL target.
# ------------------------------------------------------------

function Get-LocalTarget {
    $hostName =
        Require-ProcessEnv `
            -Name "POSTGRES_HOST"

    $portText =
        Get-ProcessEnv `
            -Name "POSTGRES_PORT" `
            -Default "5432"

    $port =
        Validate-Port `
            -Name "POSTGRES_PORT" `
            -Value $portText

    $database =
        Require-ProcessEnv `
            -Name "POSTGRES_DB"

    $user =
        Require-ProcessEnv `
            -Name "POSTGRES_USER"

    $password =
        Require-ProcessEnv `
            -Name "POSTGRES_PASSWORD"

    $sslMode =
        Get-ProcessEnv `
            -Name "POSTGRES_SSLMODE" `
            -Default "disable"

    return [PSCustomObject]@{
        Name =
            "LOCAL"

        Host =
            $hostName

        Port =
            $port

        Database =
            $database

        User =
            $user

        Password =
            $password

        SSLMode =
            $sslMode

        IsProduction =
            $false
    }
}

# ------------------------------------------------------------
# Docker PostgreSQL target.
#
# Credentials and database name stay aligned with the normal
# local POSTGRES_* database. Only host/port are overridden.
# ------------------------------------------------------------

function Get-DockerTarget {
    $local =
        Get-LocalTarget

    $hostName =
        Get-ProcessEnv `
            -Name "DOCKER_POSTGRES_HOST" `
            -Default "127.0.0.1"

    $portText =
        Get-ProcessEnv `
            -Name "DOCKER_POSTGRES_PORT" `
            -Default "5433"

    $port =
        Validate-Port `
            -Name "DOCKER_POSTGRES_PORT" `
            -Value $portText

    return [PSCustomObject]@{
        Name =
            "DOCKER"

        Host =
            $hostName

        Port =
            $port

        Database =
            $local.Database

        User =
            $local.User

        Password =
            $local.Password

        SSLMode =
            $local.SSLMode

        IsProduction =
            $false
    }
}

# ------------------------------------------------------------
# Direct production PostgreSQL target.
#
# IMPORTANT:
#
# PROD_POSTGRES_* belongs only to Atlas / administrative
# production operations.
#
# APP_POSTGRES_* is NOT used here.
#
# This lets production application traffic use a pooled Neon
# endpoint while migrations use the direct Neon endpoint.
# ------------------------------------------------------------

function Get-ProductionTarget {
    $hostName =
        Require-ProcessEnv `
            -Name "PROD_POSTGRES_HOST"

    if (
        $hostName -match
        "(?i)-pooler\."
    ) {
        throw @"
PROD_POSTGRES_HOST points to a pooled Neon endpoint.

Atlas production migrations must use the DIRECT Neon host.

PROD_POSTGRES_HOST must not contain '-pooler'.
APP_POSTGRES_HOST may use the runtime pooler separately.
"@
    }

    $portText =
        Get-ProcessEnv `
            -Name "PROD_POSTGRES_PORT" `
            -Default "5432"

    $port =
        Validate-Port `
            -Name "PROD_POSTGRES_PORT" `
            -Value $portText

    $database =
        Require-ProcessEnv `
            -Name "PROD_POSTGRES_DB"

    $user =
        Require-ProcessEnv `
            -Name "PROD_POSTGRES_USER"

    $password =
        Require-ProcessEnv `
            -Name "PROD_POSTGRES_PASSWORD"

    $sslMode =
        Get-ProcessEnv `
            -Name "PROD_POSTGRES_SSLMODE" `
            -Default "require"

    switch (
        $sslMode.ToLowerInvariant()
    ) {
        "require" {
        }

        "verify-ca" {
        }

        "verify-full" {
        }

        default {
            throw @"
PROD_POSTGRES_SSLMODE must use a TLS-enabled mode.

Allowed:
require
verify-ca
verify-full
"@
        }
    }

    return [PSCustomObject]@{
        Name =
            "PRODUCTION"

        Host =
            $hostName

        Port =
            $port

        Database =
            $database

        User =
            $user

        Password =
            $password

        SSLMode =
            $sslMode

        IsProduction =
            $true
    }
}

# ------------------------------------------------------------
# Select target.
# ------------------------------------------------------------

$target =
    $null

$needsDatabaseSelection = (
    $isMigrateApply -or
    $isMigrateStatus
)

if ($needsDatabaseSelection) {
    $localHost =
        Get-ProcessEnv `
            -Name "POSTGRES_HOST" `
            -Default "[not configured]"

    $localPort =
        Get-ProcessEnv `
            -Name "POSTGRES_PORT" `
            -Default "5432"

    $localDB =
        Get-ProcessEnv `
            -Name "POSTGRES_DB" `
            -Default "[not configured]"

    $dockerHost =
        Get-ProcessEnv `
            -Name "DOCKER_POSTGRES_HOST" `
            -Default "127.0.0.1"

    $dockerPort =
        Get-ProcessEnv `
            -Name "DOCKER_POSTGRES_PORT" `
            -Default "5433"

    $prodHost =
        Get-ProcessEnv `
            -Name "PROD_POSTGRES_HOST"

    $prodDB =
        Get-ProcessEnv `
            -Name "PROD_POSTGRES_DB"

    $prodConfigured = (
        -not [string]::IsNullOrWhiteSpace(
            $prodHost
        ) -and
        -not [string]::IsNullOrWhiteSpace(
            $prodDB
        )
    )

    Write-Host ""
    Write-Host "============================================================"

    if ($isMigrateStatus) {
        Write-Host " Select migration status database"
    }
    else {
        Write-Host " Select migration database"
    }

    Write-Host "============================================================"
    Write-Host ""

    Write-Host "  [1] LOCAL PostgreSQL"
    Write-Host "      Host : $localHost"
    Write-Host "      Port : $localPort"
    Write-Host "      DB   : $localDB"
    Write-Host ""

    Write-Host "  [2] DOCKER PostgreSQL"
    Write-Host "      Host : $dockerHost"
    Write-Host "      Port : $dockerPort"
    Write-Host "      DB   : $localDB"
    Write-Host ""

    Write-Host "  [3] PRODUCTION PostgreSQL / Neon"

    if ($prodConfigured) {
        Write-Host "      Configured: YES"
        Write-Host "      Host      : $prodHost"
        Write-Host "      DB        : $prodDB"
        Write-Host "      Source    : PROD_POSTGRES_*"
    }
    else {
        Write-Host "      Configured: NO"
        Write-Host "      Source    : PROD_POSTGRES_*"
    }

    Write-Host ""
    Write-Host "============================================================"
    Write-Host ""

    do {
        $selection =
            Read-Host "Select target [1/2/3]"
    }
    while (
        $selection -notin @(
            "1",
            "2",
            "3"
        )
    )

    switch ($selection) {
        "1" {
            $target =
                Get-LocalTarget
        }

        "2" {
            $target =
                Get-DockerTarget
        }

        "3" {
            $target =
                Get-ProductionTarget
        }
    }
}
else {
    # migrate diff and other normal Atlas commands continue
    # to use the local PostgreSQL environment.
    $target =
        Get-LocalTarget
}

# ------------------------------------------------------------
# Build Atlas target URL.
#
# Password and full connection URL are never printed.
# ------------------------------------------------------------

$env:ATLAS_DB_URL =
    New-PostgresURL `
        -DbHost $target.Host `
        -Port $target.Port `
        -Database $target.Database `
        -User $target.User `
        -Password $target.Password `
        -SSLMode $target.SSLMode

# ------------------------------------------------------------
# Prepare Atlas arguments.
# ------------------------------------------------------------

$atlasArguments =
    @($args)

if ($isMigrateDiff) {
    Ensure-AtlasDevDatabase

    $hasDevURL =
        $false

    for (
        $index = 0
        $index -lt $atlasArguments.Count
        $index++
    ) {
        $argument =
            [string]$atlasArguments[$index]

        if (
            $argument -eq "--dev-url" -or
            $argument.StartsWith(
                "--dev-url="
            )
        ) {
            $hasDevURL =
                $true

            break
        }
    }

    if (-not $hasDevURL) {
        $atlasArguments += @(
            "--dev-url",
            $atlasDevURL
        )
    }
}

# ------------------------------------------------------------
# Print selected database without credentials.
# ------------------------------------------------------------

Write-Host ""
Write-Host "============================================================"
Write-Host " Atlas database target"
Write-Host "============================================================"
Write-Host ""

Write-Host " Target   : $($target.Name)"
Write-Host " Host     : $($target.Host)"
Write-Host " Port     : $($target.Port)"
Write-Host " Database : $($target.Database)"
Write-Host " SSL mode : $($target.SSLMode)"
Write-Host ""

if ($target.IsProduction) {
    Write-Host " Source   : PROD_POSTGRES_*"
    Write-Host " Runtime  : APP_POSTGRES_* is intentionally not used"
    Write-Host " Pooler   : NO - direct production database required"
    Write-Host ""
}

if ($isMigrateDiff) {
    Write-Host " Mode     : MIGRATION DIFF"
    Write-Host " Dev DB   : $atlasDevHost`:$atlasDevPort/$atlasDevDatabase"
}
elseif ($isDryRun) {
    Write-Host " Mode     : DRY RUN - database will not be modified"
}
elseif ($isMigrateApply) {
    Write-Host " Mode     : APPLY - database can be modified"
}
elseif ($isMigrateStatus) {
    Write-Host " Mode     : MIGRATION STATUS - database will not be modified"
}
else {
    Write-Host " Mode     : Atlas command"
}

Write-Host ""
Write-Host "============================================================"
Write-Host ""

# ------------------------------------------------------------
# Apply protections.
# ------------------------------------------------------------

if (
    $isMigrateApply -and
    -not $isDryRun -and
    $target.Name -eq "DOCKER"
) {
    Write-Host "WARNING: You selected the Docker PostgreSQL database."
    Write-Host ""

    $confirmation =
        Read-Host "Type YES to apply migrations to DOCKER"

    if ($confirmation -cne "YES") {
        Write-Host ""
        Write-Host "Migration cancelled."
        exit 0
    }

    Write-Host ""
}

if (
    $isMigrateApply -and
    -not $isDryRun -and
    $target.IsProduction
) {
    Write-Host "============================================================"
    Write-Host " PRODUCTION MIGRATION SAFETY CHECK"
    Write-Host "============================================================"
    Write-Host ""
    Write-Host "You selected the PRODUCTION database."
    Write-Host ""
    Write-Host "Host     : $($target.Host)"
    Write-Host "Database : $($target.Database)"
    Write-Host ""
    Write-Host "This command can modify production database structures."
    Write-Host ""

    $confirmation =
        Read-Host "Type APPLY PRODUCTION to continue"

    if (
        $confirmation -cne
        "APPLY PRODUCTION"
    ) {
        Write-Host ""
        Write-Host "Production migration cancelled."
        exit 0
    }

    Write-Host ""
}

# ------------------------------------------------------------
# Run Atlas.
# ------------------------------------------------------------

& atlas @atlasArguments

$atlasExitCode =
    $LASTEXITCODE

# ------------------------------------------------------------
# Result.
# ------------------------------------------------------------

Write-Host ""

if ($atlasExitCode -eq 0) {
    if ($isMigrateDiff) {
        Write-Host "Atlas migration diff completed successfully."
        Write-Host "Development database: $atlasDevContainer"
    }
    elseif ($isMigrateStatus) {
        Write-Host "Atlas migration status completed successfully."
        Write-Host "Checked target: $($target.Name) ($($target.Host):$($target.Port))"
    }
    elseif ($isMigrateApply) {
        if ($isDryRun) {
            Write-Host "Atlas dry-run completed successfully."
            Write-Host "Target checked: $($target.Name) ($($target.Host):$($target.Port))"
        }
        else {
            Write-Host "Atlas migration completed successfully."
            Write-Host "Migrated target: $($target.Name) ($($target.Host):$($target.Port))"
        }
    }
}
else {
    Write-Host "Atlas command failed with exit code $atlasExitCode."
}

exit $atlasExitCode