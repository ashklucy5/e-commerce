param(
    [string]$BaseUrl = "http://127.0.0.1:8081"
)

$ErrorActionPreference = "Stop"

# ============================================================
# Output / assertions
# ============================================================

function Write-Step {
    param([string]$Message)

    Write-Host ""
    Write-Host "============================================================" -ForegroundColor Cyan
    Write-Host $Message -ForegroundColor Cyan
    Write-Host "============================================================" -ForegroundColor Cyan
    Write-Host ""
}

function Write-Pass {
    param([string]$Message)

    Write-Host "[PASS] $Message" -ForegroundColor Green
}

function Write-Info {
    param([string]$Message)

    Write-Host "[INFO] $Message" -ForegroundColor DarkCyan
}

function Write-Warn {
    param([string]$Message)

    Write-Host "[WARN] $Message" -ForegroundColor Yellow
}

function Assert-True {
    param(
        [bool]$Condition,
        [string]$Message
    )

    if (-not $Condition) {
        throw "ASSERTION FAILED: $Message"
    }

    Write-Pass $Message
}

function Assert-Equal {
    param(
        $Actual,
        $Expected,
        [string]$Message
    )

    if ($Actual -ne $Expected) {
        throw @"
ASSERTION FAILED: $Message
Expected: $Expected
Actual  : $Actual
"@
    }

    Write-Pass $Message
}

function Assert-NullableEqual {
    param(
        $Actual,
        $Expected,
        [string]$Message
    )

    $actualNull =
        $null -eq $Actual

    $expectedNull =
        $null -eq $Expected

    if (
        $actualNull -ne $expectedNull
    ) {
        throw @"
ASSERTION FAILED: $Message
Expected: $Expected
Actual  : $Actual
"@
    }

    if (
        -not $actualNull -and
        $Actual -ne $Expected
    ) {
        throw @"
ASSERTION FAILED: $Message
Expected: $Expected
Actual  : $Actual
"@
    }

    Write-Pass $Message
}

# ============================================================
# Environment / PostgreSQL helpers
# ============================================================

function Get-DotEnvValue {
    param(
        [string]$Path,
        [string]$Name
    )

    if (-not (Test-Path $Path)) {
        throw ".env not found at $Path"
    }

    foreach ($rawLine in Get-Content $Path) {
        $line =
            $rawLine.Trim()

        if (
            $line.Length -eq 0 -or
            $line.StartsWith("#")
        ) {
            continue
        }

        $parts =
            $line -split "=", 2

        if ($parts.Count -ne 2) {
            continue
        }

        if ($parts[0].Trim() -ne $Name) {
            continue
        }

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

        return $value
    }

    return ""
}

function Invoke-DockerPsqlScalar {
    param(
        [string]$DbUser,
        [string]$DbName,
        [string]$Sql
    )

    $output =
        @(
            & docker compose exec -T postgres `
                psql `
                -X `
                -qAt `
                -U $DbUser `
                -d $DbName `
                -v "ON_ERROR_STOP=1" `
                -P "pager=off" `
                -c $Sql
        )

    if ($LASTEXITCODE -ne 0) {
        throw "PostgreSQL command failed"
    }

    $lines =
        @(
            $output |
                Where-Object {
                    -not [string]::IsNullOrWhiteSpace(
                        [string]$_
                    )
                }
        )

    if ($lines.Count -eq 0) {
        return ""
    }

    return [string]$lines[$lines.Count - 1]
}

function Invoke-DockerPsqlNonQuery {
    param(
        [string]$DbUser,
        [string]$DbName,
        [string]$Sql
    )

    & docker compose exec -T postgres `
        psql `
        -X `
        -q `
        -U $DbUser `
        -d $DbName `
        -v "ON_ERROR_STOP=1" `
        -P "pager=off" `
        -c $Sql |
        Out-Null

    if ($LASTEXITCODE -ne 0) {
        throw "PostgreSQL command failed"
    }
}

# ============================================================
# API helpers
# ============================================================

function Get-FinanceViews {
    param(
        [string]$Date,
        [string]$VariantID,
        [string]$BaseUrl,
        [Microsoft.PowerShell.Commands.WebRequestSession]$WebSession
    )

    $query =
        "from=$Date&to=$Date&granularity=day&currency=BDT"

    $profitLossResponse =
        Invoke-RestMethod `
            -Method GET `
            -Uri "$BaseUrl/api/v1/admin/finance/profit-loss?$query" `
            -WebSession $WebSession

    $trendResponse =
        Invoke-RestMethod `
            -Method GET `
            -Uri "$BaseUrl/api/v1/admin/finance/profit-loss/trend?$query" `
            -WebSession $WebSession

    $productsResponse =
        Invoke-RestMethod `
            -Method GET `
            -Uri "$BaseUrl/api/v1/admin/finance/products/profitability?$query&limit=50" `
            -WebSession $WebSession

    $trendPoint =
        @(
            $trendResponse.data |
                Where-Object {
                    [string]$_.bucket_start -eq
                        $Date
                }
        ) |
            Select-Object -First 1

    $product =
        @(
            $productsResponse.data |
                Where-Object {
                    [string]$_.variant_id -eq
                        $VariantID
                }
        ) |
            Select-Object -First 1

    return [PSCustomObject]@{
        ProfitLoss =
            $profitLossResponse.data

        Trend =
            $trendPoint

        Product =
            $product

        Products =
            @(
                $productsResponse.data
            )
    }
}

function Get-BusinessInsights {
    param(
        [string]$Date,
        [string]$BaseUrl,
        [Microsoft.PowerShell.Commands.WebRequestSession]$WebSession
    )

    $query =
        "from=$Date&to=$Date&granularity=day&currency=BDT"

    $response =
        Invoke-RestMethod `
            -Method GET `
            -Uri "$BaseUrl/api/v1/admin/analytics/insights?$query" `
            -WebSession $WebSession

    return $response.data
}

function Add-TestCostHistory {
    param(
        [string]$DbUser,
        [string]$DbName,
        [string]$OrderID,
        [string]$VariantID,
        [Int64]$Amount,
        [string]$Currency,
        [int]$OffsetMinutes,
        [string]$Reference,
        [string]$StaffID
    )

    $id =
        Invoke-DockerPsqlScalar `
            -DbUser $DbUser `
            -DbName $DbName `
            -Sql @"
INSERT INTO finance_variant_cost_history (
    variant_id,
    unit_cost_amount,
    previous_unit_cost_amount,
    currency,
    effective_at,
    source,
    description,
    reference,
    created_by_staff_id,
    created_at
)
SELECT
    oi.variant_id,
    $Amount,
    NULL,
    '$Currency',
    o.created_at + ($OffsetMinutes * interval '1 minute'),
    'manual',
    'Automated finance analytics E2E fixture',
    '$Reference',
    '$StaffID'::uuid,
    now()
FROM orders o
JOIN order_items oi
    ON oi.order_id = o.id
WHERE
    o.id = '$OrderID'::uuid
    AND oi.variant_id = '$VariantID'::uuid
RETURNING id::text;
"@

    if ([string]::IsNullOrWhiteSpace($id)) {
        throw "Unable to insert test cost-history fixture $Reference"
    }

    return $id
}

function Create-TestExpense {
    param(
        [string]$BaseUrl,
        [Microsoft.PowerShell.Commands.WebRequestSession]$WebSession,
        [string]$CsrfToken,
        [string]$Date,
        [Int64]$Amount,
        [string]$Currency,
        [string]$Reference,
        [string]$IdempotencyKey
    )

    $headers =
        @{
            "X-CSRF-Token" =
                $CsrfToken

            "Idempotency-Key" =
                $IdempotencyKey
        }

    $body =
        @{
            category =
                "marketing"

            amount =
                $Amount

            currency =
                $Currency

            occurred_at =
                "$Date`T12:00:00+06:00"

            description =
                "Automated finance analytics E2E expense"

            reference =
                $Reference
        } |
            ConvertTo-Json -Depth 10

    $response =
        Invoke-RestMethod `
            -Method POST `
            -Uri "$BaseUrl/api/v1/admin/finance/expenses" `
            -WebSession $WebSession `
            -Headers $headers `
            -ContentType "application/json" `
            -Body $body

    return [string]$response.data.expense.id
}

function Void-TestExpense {
    param(
        [string]$BaseUrl,
        [Microsoft.PowerShell.Commands.WebRequestSession]$WebSession,
        [string]$CsrfToken,
        [string]$ExpenseID,
        [string]$RunTag
    )

    if ([string]::IsNullOrWhiteSpace($ExpenseID)) {
        return
    }

    $headers =
        @{
            "X-CSRF-Token" =
                $CsrfToken
        }

    Invoke-RestMethod `
        -Method POST `
        -Uri "$BaseUrl/api/v1/admin/finance/expenses/$ExpenseID/void" `
        -WebSession $WebSession `
        -Headers $headers `
        -ContentType "application/json" `
        -Body (
            @{
                reason =
                    "Automated finance analytics E2E cleanup $RunTag"
            } |
                ConvertTo-Json
        ) |
        Out-Null
}

function Test-InsightType {
    param(
        $Insights,
        [string]$Type
    )

    return @(
        $Insights.signals |
            Where-Object {
                [string]$_.type -eq
                    $Type
            }
    ).Count -gt 0
}

# ============================================================
# Local setup
# ============================================================

$projectRoot =
    Split-Path `
        -Parent $PSScriptRoot

$envPath =
    Join-Path `
        $projectRoot `
        ".env"

$dbUser =
    Get-DotEnvValue `
        -Path $envPath `
        -Name "POSTGRES_USER"

$dbName =
    Get-DotEnvValue `
        -Path $envPath `
        -Name "POSTGRES_DB"

if ([string]::IsNullOrWhiteSpace($dbUser)) {
    throw "POSTGRES_USER is missing from .env"
}

if ([string]::IsNullOrWhiteSpace($dbName)) {
    throw "POSTGRES_DB is missing from .env"
}

$runTag =
    "FIN-AN-E2E-" +
    (Get-Date -Format "yyyyMMdd-HHmmss") +
    "-" +
    [Guid]::NewGuid().
        ToString("N").
        Substring(0, 6)

$adminSession = $null
$csrf = $null
$adminStaff = $null

$missingCandidate = $null
$snapshotCandidate = $null

$wrongCurrencyHistoryID = $null
$laterHistoryID = $null
$fallbackHistoryID = $null
$snapshotOverrideHistoryID = $null

$bdtExpenseID = $null
$foreignExpenseID = $null

$voidedExpenseIDs =
    @{}

$locationPushed =
    $false

try {
    Push-Location $projectRoot
    $locationPushed = $true

    Write-Host ""
    Write-Host "Commerce Finance Analytics Runtime Test" -ForegroundColor White
    Write-Host "Run:      $runTag" -ForegroundColor White
    Write-Host "Base URL: $BaseUrl" -ForegroundColor White
    Write-Host ""

    # ========================================================
    # 1. Health
    # ========================================================

    Write-Step "1. Check API health"

    $health =
        Invoke-RestMethod `
            -Method GET `
            -Uri "$BaseUrl/health/ready"

    Assert-Equal `
        $health.status `
        "ready" `
        "API readiness status is ready"

    # ========================================================
    # 2. Admin login
    # ========================================================

    Write-Step "2. Admin password login"

    Write-Host "A credential window will open." -ForegroundColor Yellow
    Write-Host "Use the Commerce Admin email/staff code and Admin password." -ForegroundColor Yellow
    Write-Host ""

    $credential =
        Get-Credential `
            -Message "Commerce Admin Login"

    if ($null -eq $credential) {
        throw "Admin credential entry was cancelled"
    }

    $identifier =
        $credential.UserName

    $password =
        $credential.
            GetNetworkCredential().
            Password

    $adminSession =
        New-Object `
            Microsoft.PowerShell.Commands.WebRequestSession

    $login =
        Invoke-RestMethod `
            -Method POST `
            -Uri "$BaseUrl/api/v1/admin/auth/login" `
            -WebSession $adminSession `
            -ContentType "application/json" `
            -Body (
                @{
                    identifier =
                        $identifier

                    password =
                        $password
                } |
                    ConvertTo-Json
            )

    $password = $null
    $credential = $null

    Assert-True `
        (
            -not [string]::IsNullOrWhiteSpace(
                [string]$login.data.challenge_token
            )
        ) `
        "Admin login challenge was issued"

    if ($login.data.mfa_enrollment_required) {
        throw "Admin account requires MFA enrollment before this test can run"
    }

    # ========================================================
    # 3. MFA
    # ========================================================

    Write-Step "3. Verify Admin MFA"

    $totp = ""

    do {
        $secureTotp =
            Read-Host `
                "Enter current 6-digit Admin TOTP" `
                -AsSecureString

        $totp =
            [System.Net.NetworkCredential]::new(
                "",
                $secureTotp
            ).Password

        $secureTotp = $null
    }
    while ($totp -notmatch '^\d{6}$')

    $mfa =
        Invoke-RestMethod `
            -Method POST `
            -Uri "$BaseUrl/api/v1/admin/auth/mfa/verify" `
            -WebSession $adminSession `
            -ContentType "application/json" `
            -Body (
                @{
                    challenge_token =
                        $login.data.challenge_token

                    method =
                        "totp"

                    code =
                        $totp
                } |
                    ConvertTo-Json
            )

    $totp = $null

    $csrf =
        [string]$mfa.data.csrf_token

    $adminStaff =
        $mfa.data.principal.staff

    Assert-True `
        (
            -not [string]::IsNullOrWhiteSpace(
                $csrf
            )
        ) `
        "Admin CSRF token was issued"

    Assert-True `
        (
            @(
                $adminStaff.permissions
            ) -contains
                "admin.finance.read"
        ) `
        "Admin has admin.finance.read"

    Assert-True `
        (
            @(
                $adminStaff.permissions
            ) -contains
                "admin.finance.manage"
        ) `
        "Admin has admin.finance.manage"

    # ========================================================
    # 4. Find safe historical fallback fixture
    # ========================================================

    Write-Step "4. Find historical order item with missing cost truth"

    $missingCandidateJson =
        Invoke-DockerPsqlScalar `
            -DbUser $dbUser `
            -DbName $dbName `
            -Sql @"
SELECT json_build_object(
    'order_id', o.id::text,
    'order_number', o.order_number,
    'variant_id', oi.variant_id::text,
    'product_id', pv.product_id::text,
    'sku', oi.sku,
    'product_name', oi.product_name,
    'quantity', oi.quantity,
    'order_created_at', o.created_at,
    'paid_at', o.paid_at,
    'business_date',
        to_char(
            o.paid_at AT TIME ZONE 'Asia/Dhaka',
            'YYYY-MM-DD'
        )
)::text
FROM orders o
JOIN order_items oi
    ON oi.order_id = o.id
JOIN product_variants pv
    ON pv.id = oi.variant_id
WHERE
    o.paid_at IS NOT NULL
    AND o.currency = 'BDT'
    AND o.payment_status IN (
        'paid',
        'cod_collected',
        'refunded'
    )
    AND oi.unit_cost_amount IS NULL
    AND oi.line_cost_amount IS NULL
    AND oi.cost_currency IS NULL

    -- The baseline must genuinely be missing. No already-recorded
    -- applicable BDT historical cost may exist.
    AND NOT EXISTS (
        SELECT 1
        FROM finance_variant_cost_history h
        WHERE
            h.variant_id = oi.variant_id
            AND h.currency = 'BDT'
            AND h.effective_at <= o.created_at
    )

    -- Keep the selected SKU isolated within the reporting day so
    -- the expected delta is exactly this order item's quantity.
    AND (
        SELECT COUNT(*)
        FROM orders o2
        JOIN order_items oi2
            ON oi2.order_id = o2.id
        WHERE
            oi2.variant_id = oi.variant_id
            AND o2.paid_at IS NOT NULL
            AND o2.currency = 'BDT'
            AND o2.payment_status IN (
                'paid',
                'cod_collected',
                'refunded'
            )
            AND (
                o2.paid_at AT TIME ZONE 'Asia/Dhaka'
            )::date = (
                o.paid_at AT TIME ZONE 'Asia/Dhaka'
            )::date
    ) = 1

    -- Product profitability is capped at 50 rows.
    AND (
        SELECT COUNT(DISTINCT oi3.variant_id)
        FROM orders o3
        JOIN order_items oi3
            ON oi3.order_id = o3.id
        WHERE
            o3.paid_at IS NOT NULL
            AND o3.currency = 'BDT'
            AND o3.payment_status IN (
                'paid',
                'cod_collected',
                'refunded'
            )
            AND (
                o3.paid_at AT TIME ZONE 'Asia/Dhaka'
            )::date = (
                o.paid_at AT TIME ZONE 'Asia/Dhaka'
            )::date
    ) <= 50

ORDER BY
    o.paid_at DESC,
    o.id,
    oi.id
LIMIT 1;
"@

    if ([string]::IsNullOrWhiteSpace($missingCandidateJson)) {
        throw @"
No safe historical BDT order item with a missing cost snapshot was found.

This test intentionally refuses to NULL or overwrite an immutable order-item
cost snapshot just to manufacture a fixture.

Use a development database containing at least one pre-cost-snapshot historical
order, then rerun this test.
"@
    }

    $missingCandidate =
        $missingCandidateJson |
            ConvertFrom-Json

    $testDate =
        [string]$missingCandidate.business_date

    $variantID =
        [string]$missingCandidate.variant_id

    $orderID =
        [string]$missingCandidate.order_id

    $quantity =
        [Int64]$missingCandidate.quantity

    Write-Info "Order:        $($missingCandidate.order_number)"
    Write-Info "Order ID:     $orderID"
    Write-Info "SKU:          $($missingCandidate.sku)"
    Write-Info "Variant ID:   $variantID"
    Write-Info "Quantity:     $quantity"
    Write-Info "Business day: $testDate"

    Assert-True `
        ($quantity -gt 0) `
        "Historical fixture quantity is positive"

    # ========================================================
    # 5. Baseline finance analytics
    # ========================================================

    Write-Step "5. Capture missing-cost baseline"

    $baseline =
        Get-FinanceViews `
            -Date $testDate `
            -VariantID $variantID `
            -BaseUrl $BaseUrl `
            -WebSession $adminSession

    Assert-True `
        ($null -ne $baseline.Trend) `
        "Profit/loss trend contains the historical business day"

    Assert-True `
        ($null -ne $baseline.Product) `
        "Product profitability contains the historical SKU"

    Assert-True `
        (
            [Int64]$baseline.ProfitLoss.missing_cost_units -ge
                $quantity
        ) `
        "Profit/loss baseline reports the missing-cost units"

    Assert-True `
        (
            [Int64]$baseline.Trend.missing_cost_units -ge
                $quantity
        ) `
        "Trend baseline reports the missing-cost units"

    Assert-True `
        (
            [Int64]$baseline.Product.missing_cost_units -ge
                $quantity
        ) `
        "Product baseline reports the missing-cost units"

    $baselineInsights =
        Get-BusinessInsights `
            -Date $testDate `
            -BaseUrl $BaseUrl `
            -WebSession $adminSession

    Assert-True `
        ($null -ne $baselineInsights.signals) `
        "Business insights endpoint executes against the new finance model"

    if (-not [bool]$baseline.ProfitLoss.profit_complete) {
        Assert-True `
            (
                Test-InsightType `
                    -Insights $baselineInsights `
                    -Type "profitability_coverage"
            ) `
            "Incomplete profitability creates a deterministic coverage insight"
    }

    # ========================================================
    # 6. Insert deliberately ineligible historical costs
    # ========================================================

    Write-Step "6. Verify currency and effective-date boundaries"

    $wrongCurrencyHistoryID =
        Add-TestCostHistory `
            -DbUser $dbUser `
            -DbName $dbName `
            -OrderID $orderID `
            -VariantID $variantID `
            -Amount 910001 `
            -Currency "USD" `
            -OffsetMinutes -120 `
            -Reference "$runTag-WRONG-FX" `
            -StaffID ([string]$adminStaff.id)

    $laterHistoryID =
        Add-TestCostHistory `
            -DbUser $dbUser `
            -DbName $dbName `
            -OrderID $orderID `
            -VariantID $variantID `
            -Amount 920002 `
            -Currency "BDT" `
            -OffsetMinutes 120 `
            -Reference "$runTag-LATER" `
            -StaffID ([string]$adminStaff.id)

    $afterIneligible =
        Get-FinanceViews `
            -Date $testDate `
            -VariantID $variantID `
            -BaseUrl $BaseUrl `
            -WebSession $adminSession

    Assert-Equal `
        ([Int64]$afterIneligible.ProfitLoss.costed_units) `
        ([Int64]$baseline.ProfitLoss.costed_units) `
        "Wrong-currency and later-effective costs do not increase P&L costed units"

    Assert-Equal `
        ([Int64]$afterIneligible.ProfitLoss.missing_cost_units) `
        ([Int64]$baseline.ProfitLoss.missing_cost_units) `
        "Wrong-currency and later-effective costs do not reduce P&L missing units"

    Assert-Equal `
        ([Int64]$afterIneligible.ProfitLoss.gross_cogs_amount) `
        ([Int64]$baseline.ProfitLoss.gross_cogs_amount) `
        "Wrong-currency and later-effective costs do not change P&L COGS"

    Assert-Equal `
        ([Int64]$afterIneligible.Trend.gross_cogs_amount) `
        ([Int64]$baseline.Trend.gross_cogs_amount) `
        "Wrong-currency and later-effective costs do not change trend COGS"

    Assert-Equal `
        ([Int64]$afterIneligible.Product.known_cogs_amount) `
        ([Int64]$baseline.Product.known_cogs_amount) `
        "Wrong-currency and later-effective costs do not change product COGS"

    # ========================================================
    # 7. Insert valid historical fallback
    # ========================================================

    Write-Step "7. Verify historical SKU buying-cost fallback"

    $fallbackCost =
        [Int64]4321

    $fallbackHistoryID =
        Add-TestCostHistory `
            -DbUser $dbUser `
            -DbName $dbName `
            -OrderID $orderID `
            -VariantID $variantID `
            -Amount $fallbackCost `
            -Currency "BDT" `
            -OffsetMinutes -60 `
            -Reference "$runTag-FALLBACK" `
            -StaffID ([string]$adminStaff.id)

    $afterFallback =
        Get-FinanceViews `
            -Date $testDate `
            -VariantID $variantID `
            -BaseUrl $BaseUrl `
            -WebSession $adminSession

    $expectedCostDelta =
        $fallbackCost *
        $quantity

    Assert-Equal `
        ([Int64]$afterFallback.ProfitLoss.costed_units) `
        (
            [Int64]$baseline.ProfitLoss.costed_units +
            $quantity
        ) `
        "P&L converts the historical units from missing to costed"

    Assert-Equal `
        ([Int64]$afterFallback.ProfitLoss.missing_cost_units) `
        (
            [Int64]$baseline.ProfitLoss.missing_cost_units -
            $quantity
        ) `
        "P&L historical fallback reduces missing-cost units exactly"

    Assert-Equal `
        ([Int64]$afterFallback.ProfitLoss.gross_cogs_amount) `
        (
            [Int64]$baseline.ProfitLoss.gross_cogs_amount +
            $expectedCostDelta
        ) `
        "P&L uses the historical buying cost in gross COGS"

    Assert-Equal `
        ([Int64]$afterFallback.Trend.costed_units) `
        (
            [Int64]$baseline.Trend.costed_units +
            $quantity
        ) `
        "Trend uses the same historical fallback"

    Assert-Equal `
        ([Int64]$afterFallback.Trend.missing_cost_units) `
        (
            [Int64]$baseline.Trend.missing_cost_units -
            $quantity
        ) `
        "Trend missing-cost units agree with P&L"

    Assert-Equal `
        ([Int64]$afterFallback.Trend.gross_cogs_amount) `
        (
            [Int64]$baseline.Trend.gross_cogs_amount +
            $expectedCostDelta
        ) `
        "Trend historical COGS agrees with P&L"

    Assert-Equal `
        ([Int64]$afterFallback.Product.costed_units) `
        (
            [Int64]$baseline.Product.costed_units +
            $quantity
        ) `
        "Product profitability uses historical fallback units"

    Assert-Equal `
        ([Int64]$afterFallback.Product.missing_cost_units) `
        (
            [Int64]$baseline.Product.missing_cost_units -
            $quantity
        ) `
        "Product profitability reduces missing units exactly"

    Assert-Equal `
        ([Int64]$afterFallback.Product.known_cogs_amount) `
        (
            [Int64]$baseline.Product.known_cogs_amount +
            $expectedCostDelta
        ) `
        "Product profitability uses the same historical COGS"

    # ========================================================
    # 8. Snapshot precedence fixture
    # ========================================================

    Write-Step "8. Verify immutable order-item snapshot precedence"

    $snapshotCandidateJson =
        Invoke-DockerPsqlScalar `
            -DbUser $dbUser `
            -DbName $dbName `
            -Sql @"
SELECT json_build_object(
    'order_id', o.id::text,
    'order_number', o.order_number,
    'variant_id', oi.variant_id::text,
    'sku', oi.sku,
    'quantity', oi.quantity,
    'unit_cost_amount', oi.unit_cost_amount,
    'business_date',
        to_char(
            o.paid_at AT TIME ZONE 'Asia/Dhaka',
            'YYYY-MM-DD'
        )
)::text
FROM orders o
JOIN order_items oi
    ON oi.order_id = o.id
WHERE
    o.paid_at IS NOT NULL
    AND o.currency = 'BDT'
    AND o.payment_status IN (
        'paid',
        'cod_collected',
        'refunded'
    )
    AND oi.unit_cost_amount IS NOT NULL
    AND oi.line_cost_amount IS NOT NULL
    AND oi.cost_currency = 'BDT'

    -- Make sure a new history row for this SKU cannot change a
    -- different missing-snapshot row in the same reporting day.
    AND NOT EXISTS (
        SELECT 1
        FROM orders o2
        JOIN order_items oi2
            ON oi2.order_id = o2.id
        WHERE
            oi2.variant_id = oi.variant_id
            AND oi2.unit_cost_amount IS NULL
            AND o2.paid_at IS NOT NULL
            AND o2.currency = 'BDT'
            AND o2.payment_status IN (
                'paid',
                'cod_collected',
                'refunded'
            )
            AND (
                o2.paid_at AT TIME ZONE 'Asia/Dhaka'
            )::date = (
                o.paid_at AT TIME ZONE 'Asia/Dhaka'
            )::date
    )

    AND (
        SELECT COUNT(DISTINCT oi3.variant_id)
        FROM orders o3
        JOIN order_items oi3
            ON oi3.order_id = o3.id
        WHERE
            o3.paid_at IS NOT NULL
            AND o3.currency = 'BDT'
            AND o3.payment_status IN (
                'paid',
                'cod_collected',
                'refunded'
            )
            AND (
                o3.paid_at AT TIME ZONE 'Asia/Dhaka'
            )::date = (
                o.paid_at AT TIME ZONE 'Asia/Dhaka'
            )::date
    ) <= 50

ORDER BY
    o.paid_at DESC,
    o.id,
    oi.id
LIMIT 1;
"@

    if (
        [string]::IsNullOrWhiteSpace(
            $snapshotCandidateJson
        )
    ) {
        Write-Warn "No isolated snapshot-backed order item was available; snapshot-precedence subtest was skipped."
    }
    else {
        $snapshotCandidate =
            $snapshotCandidateJson |
                ConvertFrom-Json

        $snapshotDate =
            [string]$snapshotCandidate.business_date

        $snapshotVariantID =
            [string]$snapshotCandidate.variant_id

        Write-Info "Snapshot SKU:  $($snapshotCandidate.sku)"
        Write-Info "Snapshot cost: $($snapshotCandidate.unit_cost_amount)"
        Write-Info "Snapshot date: $snapshotDate"

        $snapshotBefore =
            Get-FinanceViews `
                -Date $snapshotDate `
                -VariantID $snapshotVariantID `
                -BaseUrl $BaseUrl `
                -WebSession $adminSession

        Assert-True `
            ($null -ne $snapshotBefore.Product) `
            "Snapshot-backed SKU is present in product profitability"

        $snapshotOverrideHistoryID =
            Add-TestCostHistory `
                -DbUser $dbUser `
                -DbName $dbName `
                -OrderID ([string]$snapshotCandidate.order_id) `
                -VariantID $snapshotVariantID `
                -Amount 999999 `
                -Currency "BDT" `
                -OffsetMinutes -30 `
                -Reference "$runTag-SNAPSHOT-PRECEDENCE" `
                -StaffID ([string]$adminStaff.id)

        $snapshotAfter =
            Get-FinanceViews `
                -Date $snapshotDate `
                -VariantID $snapshotVariantID `
                -BaseUrl $BaseUrl `
                -WebSession $adminSession

        Assert-Equal `
            ([Int64]$snapshotAfter.ProfitLoss.gross_cogs_amount) `
            ([Int64]$snapshotBefore.ProfitLoss.gross_cogs_amount) `
            "Historical cost does not override immutable P&L snapshot COGS"

        Assert-Equal `
            ([Int64]$snapshotAfter.Trend.gross_cogs_amount) `
            ([Int64]$snapshotBefore.Trend.gross_cogs_amount) `
            "Historical cost does not override immutable trend snapshot COGS"

        Assert-Equal `
            ([Int64]$snapshotAfter.Product.known_cogs_amount) `
            ([Int64]$snapshotBefore.Product.known_cogs_amount) `
            "Historical cost does not override immutable product snapshot COGS"
    }

    # ========================================================
    # 9. Expense currency separation
    # ========================================================

    Write-Step "9. Verify recorded expenses stay separate from COGS"

    $beforeExpenses =
        Get-FinanceViews `
            -Date $testDate `
            -VariantID $variantID `
            -BaseUrl $BaseUrl `
            -WebSession $adminSession

    $foreignExpenseAmount =
        [Int64]5555

    $foreignExpenseID =
        Create-TestExpense `
            -BaseUrl $BaseUrl `
            -WebSession $adminSession `
            -CsrfToken $csrf `
            -Date $testDate `
            -Amount $foreignExpenseAmount `
            -Currency "USD" `
            -Reference "$runTag-USD-EXPENSE" `
            -IdempotencyKey "$runTag-USD-EXPENSE"

    Assert-True `
        (
            -not [string]::IsNullOrWhiteSpace(
                $foreignExpenseID
            )
        ) `
        "Foreign-currency test expense was created"

    $afterForeignExpense =
        Get-FinanceViews `
            -Date $testDate `
            -VariantID $variantID `
            -BaseUrl $BaseUrl `
            -WebSession $adminSession

    Assert-Equal `
        ([Int64]$afterForeignExpense.ProfitLoss.recorded_expenses_amount) `
        ([Int64]$beforeExpenses.ProfitLoss.recorded_expenses_amount) `
        "USD expense is excluded from the BDT recorded-expense amount"

    Assert-Equal `
        ([Int64]$afterForeignExpense.ProfitLoss.gross_cogs_amount) `
        ([Int64]$beforeExpenses.ProfitLoss.gross_cogs_amount) `
        "Foreign-currency expense does not alter COGS"

    Assert-Equal `
        ([Int64]$afterForeignExpense.ProfitLoss.foreign_currency_expense_entries_excluded) `
        (
            [Int64]$beforeExpenses.ProfitLoss.foreign_currency_expense_entries_excluded +
            1
        ) `
        "BDT P&L explicitly counts the excluded foreign-currency expense"

    $bdtExpenseAmount =
        [Int64]7777

    $bdtExpenseID =
        Create-TestExpense `
            -BaseUrl $BaseUrl `
            -WebSession $adminSession `
            -CsrfToken $csrf `
            -Date $testDate `
            -Amount $bdtExpenseAmount `
            -Currency "BDT" `
            -Reference "$runTag-BDT-EXPENSE" `
            -IdempotencyKey "$runTag-BDT-EXPENSE"

    Assert-True `
        (
            -not [string]::IsNullOrWhiteSpace(
                $bdtExpenseID
            )
        ) `
        "BDT test expense was created"

    $afterBDTExpense =
        Get-FinanceViews `
            -Date $testDate `
            -VariantID $variantID `
            -BaseUrl $BaseUrl `
            -WebSession $adminSession

    Assert-Equal `
        ([Int64]$afterBDTExpense.ProfitLoss.recorded_expenses_amount) `
        (
            [Int64]$afterForeignExpense.ProfitLoss.recorded_expenses_amount +
            $bdtExpenseAmount
        ) `
        "BDT expense increases recorded operating expenses exactly"

    Assert-Equal `
        ([Int64]$afterBDTExpense.ProfitLoss.gross_cogs_amount) `
        ([Int64]$afterForeignExpense.ProfitLoss.gross_cogs_amount) `
        "BDT operating expense does not alter gross COGS"

    Assert-Equal `
        ([Int64]$afterBDTExpense.ProfitLoss.net_cogs_amount) `
        ([Int64]$afterForeignExpense.ProfitLoss.net_cogs_amount) `
        "BDT operating expense does not alter net COGS"

    Assert-NullableEqual `
        $afterBDTExpense.ProfitLoss.gross_profit_amount `
        $afterForeignExpense.ProfitLoss.gross_profit_amount `
        "Operating expense does not alter gross profit"

    if (
        $null -ne
            $afterForeignExpense.
                ProfitLoss.
                net_profit_after_recorded_expenses_amount -and
        $null -ne
            $afterBDTExpense.
                ProfitLoss.
                net_profit_after_recorded_expenses_amount
    ) {
        Assert-Equal `
            ([Int64]$afterBDTExpense.ProfitLoss.net_profit_after_recorded_expenses_amount) `
            (
                [Int64]$afterForeignExpense.ProfitLoss.net_profit_after_recorded_expenses_amount -
                $bdtExpenseAmount
            ) `
            "Complete net profit decreases only by the recorded BDT expense"
    }
    else {
        Write-Pass "Net profit remains null because COGS coverage is partial; the test does not manufacture completeness."
    }

    # ========================================================
    # 10. Finance-aware business insights
    # ========================================================

    Write-Step "10. Verify finance-aware deterministic insights"

    $insights =
        Get-BusinessInsights `
            -Date $testDate `
            -BaseUrl $BaseUrl `
            -WebSession $adminSession

    Assert-True `
        ($null -ne $insights.signals) `
        "Business insights endpoint returns deterministic signals"

    Assert-True `
        (
            Test-InsightType `
                -Insights $insights `
                -Type "finance_currency_exclusion"
        ) `
        "Foreign-currency expense exclusion is surfaced as an insight"

    if (
        -not [bool]$afterBDTExpense.ProfitLoss.profit_complete
    ) {
        Assert-True `
            (
                Test-InsightType `
                    -Insights $insights `
                    -Type "profitability_coverage"
            ) `
            "Incomplete cost coverage is surfaced as an insight"
    }

    if (
        [bool]$afterBDTExpense.ProfitLoss.profit_complete -and
        $null -ne
            $afterBDTExpense.
                ProfitLoss.
                net_profit_after_recorded_expenses_amount -and
        [Int64]$afterBDTExpense.
            ProfitLoss.
            net_profit_after_recorded_expenses_amount -lt 0
    ) {
        Assert-True `
            (
                Test-InsightType `
                    -Insights $insights `
                    -Type "recorded_net_loss"
            ) `
            "Negative profit after recorded expenses is surfaced deterministically"
    }

    # ========================================================
    # 11. Void temporary expenses
    # ========================================================

    Write-Step "11. Void temporary finance expenses"

    Void-TestExpense `
        -BaseUrl $BaseUrl `
        -WebSession $adminSession `
        -CsrfToken $csrf `
        -ExpenseID $foreignExpenseID `
        -RunTag $runTag

    $voidedExpenseIDs[$foreignExpenseID] =
        $true

    Write-Pass "Foreign-currency test expense was voided"

    Void-TestExpense `
        -BaseUrl $BaseUrl `
        -WebSession $adminSession `
        -CsrfToken $csrf `
        -ExpenseID $bdtExpenseID `
        -RunTag $runTag

    $voidedExpenseIDs[$bdtExpenseID] =
        $true

    Write-Pass "BDT test expense was voided"

    $afterExpenseCleanup =
        Get-FinanceViews `
            -Date $testDate `
            -VariantID $variantID `
            -BaseUrl $BaseUrl `
            -WebSession $adminSession

    Assert-Equal `
        ([Int64]$afterExpenseCleanup.ProfitLoss.recorded_expenses_amount) `
        ([Int64]$beforeExpenses.ProfitLoss.recorded_expenses_amount) `
        "Voiding restores the original recorded-expense amount"

    Assert-Equal `
        ([Int64]$afterExpenseCleanup.ProfitLoss.foreign_currency_expense_entries_excluded) `
        ([Int64]$beforeExpenses.ProfitLoss.foreign_currency_expense_entries_excluded) `
        "Voiding restores the original foreign-currency exclusion count"

    # ========================================================
    # 12. Remove temporary historical-cost fixtures
    # ========================================================

    Write-Step "12. Remove test-only historical-cost fixtures"

    Invoke-DockerPsqlNonQuery `
        -DbUser $dbUser `
        -DbName $dbName `
        -Sql @"
DELETE FROM finance_variant_cost_history
WHERE reference IN (
    '$runTag-WRONG-FX',
    '$runTag-LATER',
    '$runTag-FALLBACK',
    '$runTag-SNAPSHOT-PRECEDENCE'
);
"@

    $wrongCurrencyHistoryID = $null
    $laterHistoryID = $null
    $fallbackHistoryID = $null
    $snapshotOverrideHistoryID = $null

    $remainingHistoryRows =
        Invoke-DockerPsqlScalar `
            -DbUser $dbUser `
            -DbName $dbName `
            -Sql @"
SELECT COUNT(*)::text
FROM finance_variant_cost_history
WHERE reference LIKE '$runTag-%';
"@

    Assert-Equal `
        ([int]$remainingHistoryRows) `
        0 `
        "No test-only cost-history fixtures remain"

    # ========================================================
    # 13. Verify analytics return to baseline
    # ========================================================

    Write-Step "13. Verify finance analytics returned to baseline"

    $final =
        Get-FinanceViews `
            -Date $testDate `
            -VariantID $variantID `
            -BaseUrl $BaseUrl `
            -WebSession $adminSession

    Assert-Equal `
        ([Int64]$final.ProfitLoss.costed_units) `
        ([Int64]$baseline.ProfitLoss.costed_units) `
        "P&L costed units returned to baseline"

    Assert-Equal `
        ([Int64]$final.ProfitLoss.missing_cost_units) `
        ([Int64]$baseline.ProfitLoss.missing_cost_units) `
        "P&L missing units returned to baseline"

    Assert-Equal `
        ([Int64]$final.ProfitLoss.gross_cogs_amount) `
        ([Int64]$baseline.ProfitLoss.gross_cogs_amount) `
        "P&L gross COGS returned to baseline"

    Assert-Equal `
        ([Int64]$final.Trend.gross_cogs_amount) `
        ([Int64]$baseline.Trend.gross_cogs_amount) `
        "Trend COGS returned to baseline"

    Assert-Equal `
        ([Int64]$final.Product.known_cogs_amount) `
        ([Int64]$baseline.Product.known_cogs_amount) `
        "Product profitability COGS returned to baseline"

    # ========================================================
    # Success
    # ========================================================

    Write-Host ""
    Write-Host "============================================================" -ForegroundColor Green
    Write-Host " FINANCE ANALYTICS RUNTIME TEST PASSED" -ForegroundColor Green
    Write-Host "============================================================" -ForegroundColor Green
    Write-Host ""

    Write-Host "Run:              $runTag"
    Write-Host "Historical order: $($missingCandidate.order_number)"
    Write-Host "Historical SKU:   $($missingCandidate.sku)"
    Write-Host "Business date:    $testDate"
    Write-Host ""

    Write-Host "Validated:" -ForegroundColor White
    Write-Host "  API readiness"
    Write-Host "  Admin finance RBAC"
    Write-Host "  profit/loss endpoint runtime SQL"
    Write-Host "  profit/loss trend runtime SQL"
    Write-Host "  product profitability runtime SQL"
    Write-Host "  business insights runtime SQL"
    Write-Host "  missing historical cost remains partial"
    Write-Host "  wrong-currency SKU cost is ignored"
    Write-Host "  later-effective SKU cost is ignored"
    Write-Host "  latest applicable historical SKU cost fallback"
    Write-Host "  P&L / trend / product COGS agreement"

    if ($null -ne $snapshotCandidate) {
        Write-Host "  immutable order-item snapshot precedence"
    }
    else {
        Write-Host "  immutable order-item snapshot precedence: fixture unavailable"
    }

    Write-Host "  BDT operating expenses remain separate from COGS"
    Write-Host "  foreign-currency expenses are excluded without FX mixing"
    Write-Host "  finance-aware deterministic insights"
    Write-Host "  expense cleanup through normal void lifecycle"
    Write-Host "  test-only historical-cost fixture cleanup"
    Write-Host "  post-cleanup analytics baseline restoration"
    Write-Host ""

    Write-Host "No password, session token, CSRF token, or TOTP was printed." -ForegroundColor DarkGray
    Write-Host ""
}
catch {
    Write-Host ""
    Write-Host "============================================================" -ForegroundColor Red
    Write-Host " FINANCE ANALYTICS RUNTIME TEST FAILED" -ForegroundColor Red
    Write-Host "============================================================" -ForegroundColor Red
    Write-Host ""

    Write-Host $_.Exception.Message -ForegroundColor Red
    Write-Host ""
    Write-Host "Best-effort cleanup will now run." -ForegroundColor Yellow
    Write-Host ""

    exit 1
}
finally {
    # --------------------------------------------------------
    # Best-effort expense cleanup through the normal API.
    # --------------------------------------------------------

    if (
        $null -ne $adminSession -and
        -not [string]::IsNullOrWhiteSpace($csrf)
    ) {
        foreach (
            $expenseID in @(
                $foreignExpenseID,
                $bdtExpenseID
            )
        ) {
            if (
                [string]::IsNullOrWhiteSpace(
                    [string]$expenseID
                )
            ) {
                continue
            }

            if (
                $voidedExpenseIDs.ContainsKey(
                    [string]$expenseID
                )
            ) {
                continue
            }

            try {
                Void-TestExpense `
                    -BaseUrl $BaseUrl `
                    -WebSession $adminSession `
                    -CsrfToken $csrf `
                    -ExpenseID ([string]$expenseID) `
                    -RunTag $runTag

                Write-Warn "Best-effort cleanup voided expense $expenseID."
            }
            catch {
                Write-Warn "Unable to automatically void expense $expenseID."
            }
        }
    }

    # --------------------------------------------------------
    # Test cost-history fixtures were inserted directly only
    # so the read model could be exercised without changing a
    # real immutable order-item snapshot. Remove only rows
    # carrying this run's unique reference prefix.
    # --------------------------------------------------------

    if (
        -not [string]::IsNullOrWhiteSpace($dbUser) -and
        -not [string]::IsNullOrWhiteSpace($dbName) -and
        -not [string]::IsNullOrWhiteSpace($runTag)
    ) {
        try {
            Invoke-DockerPsqlNonQuery `
                -DbUser $dbUser `
                -DbName $dbName `
                -Sql @"
DELETE FROM finance_variant_cost_history
WHERE reference LIKE '$runTag-%';
"@

            Write-Warn "Best-effort cleanup removed any remaining test cost-history fixtures."
        }
        catch {
            Write-Warn "Unable to automatically remove all test cost-history fixtures."
        }
    }

    $csrf = $null
    $totp = $null
    $password = $null
    $credential = $null

    if ($locationPushed) {
        Pop-Location
    }
}