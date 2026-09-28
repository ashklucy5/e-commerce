param(
    [string]$BaseUrl = "http://127.0.0.1:8081"
)

$ErrorActionPreference = "Stop"

Add-Type -AssemblyName System.Net.Http

# ============================================================
# Helpers
# ============================================================

function Write-Step {
    param(
        [string]$Message
    )

    Write-Host ""
    Write-Host "============================================================" -ForegroundColor Cyan
    Write-Host $Message -ForegroundColor Cyan
    Write-Host "============================================================" -ForegroundColor Cyan
    Write-Host ""
}

function Write-Pass {
    param(
        [string]$Message
    )

    Write-Host "[PASS] $Message" -ForegroundColor Green
}

function Write-Info {
    param(
        [string]$Message
    )

    Write-Host "[INFO] $Message" -ForegroundColor DarkCyan
}

function Write-Warn {
    param(
        [string]$Message
    )

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

function Get-HttpFailure {
    param(
        [System.Management.Automation.ErrorRecord]$Record
    )

    $status = 0
    $body = ""
    $json = $null

    if ($null -ne $Record.Exception.Response) {
        try {
            $status =
                [int]$Record.Exception.Response.StatusCode
        }
        catch {
            $status = 0
        }

        try {
            $stream =
                $Record.Exception.Response.GetResponseStream()

            if ($null -ne $stream) {
                $reader =
                    New-Object System.IO.StreamReader(
                        $stream
                    )

                try {
                    $body =
                        $reader.ReadToEnd()
                }
                finally {
                    $reader.Dispose()
                }
            }
        }
        catch {
        }
    }

    if (
        [string]::IsNullOrWhiteSpace($body) -and
        $null -ne $Record.ErrorDetails -and
        -not [string]::IsNullOrWhiteSpace(
            $Record.ErrorDetails.Message
        )
    ) {
        $body =
            $Record.ErrorDetails.Message
    }

    if (-not [string]::IsNullOrWhiteSpace($body)) {
        try {
            $json =
                $body |
                    ConvertFrom-Json
        }
        catch {
        }
    }

    return [PSCustomObject]@{
        Status = $status
        Body   = $body
        Json   = $json
    }
}

function Expect-ApiFailure {
    param(
        [Parameter(Mandatory = $true)]
        [scriptblock]$Action,

        [Parameter(Mandatory = $true)]
        [int]$ExpectedStatus,

        [Parameter(Mandatory = $true)]
        [string]$ExpectedCode,

        [Parameter(Mandatory = $true)]
        [string]$Label
    )

    $caught = $null

    try {
        & $Action | Out-Null
    }
    catch {
        $caught = $_
    }

    if ($null -eq $caught) {
        throw @"
$Label succeeded unexpectedly.
Expected HTTP $ExpectedStatus / $ExpectedCode.
"@
    }

    $failure =
        Get-HttpFailure `
            -Record $caught

    if ($failure.Status -ne $ExpectedStatus) {
        throw @"
$Label returned wrong HTTP status.
Expected: $ExpectedStatus
Actual  : $($failure.Status)
Body    : $($failure.Body)
"@
    }

    if (
        $null -eq $failure.Json -or
        $null -eq $failure.Json.error -or
        [string]$failure.Json.error.code -ne
            $ExpectedCode
    ) {
        throw @"
$Label returned wrong API error.
Expected: $ExpectedCode
Body    : $($failure.Body)
"@
    }

    Write-Pass "$Label returned expected HTTP $ExpectedStatus / $ExpectedCode"
}

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
                -U $DbUser `
                -d $DbName `
                -tA `
                -P "pager=off" `
                -v "ON_ERROR_STOP=1" `
                -c $Sql
        )

    if ($LASTEXITCODE -ne 0) {
        throw "PostgreSQL verification command failed"
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

function Invoke-MultipartWorkbookUpload {
    param(
        [string]$Uri,
        [Microsoft.PowerShell.Commands.WebRequestSession]$WebSession,
        [string]$CsrfToken,
        [string]$FilePath
    )

    $handler = $null
    $client = $null
    $multipart = $null
    $fileStream = $null
    $fileContent = $null
    $request = $null
    $response = $null

    try {
        $handler =
            New-Object System.Net.Http.HttpClientHandler

        $handler.UseCookies =
            $true

        $handler.CookieContainer =
            $WebSession.Cookies

        $client =
            New-Object System.Net.Http.HttpClient(
                $handler
            )

        $client.Timeout =
            [TimeSpan]::FromSeconds(
                90
            )

        $multipart =
            New-Object System.Net.Http.MultipartFormDataContent

        $fileStream =
            [System.IO.File]::OpenRead(
                $FilePath
            )

        $fileContent =
            New-Object System.Net.Http.StreamContent(
                $fileStream
            )

        $fileContent.Headers.ContentType =
            [System.Net.Http.Headers.MediaTypeHeaderValue]::Parse(
                "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
            )

        $multipart.Add(
            $fileContent,
            "file",
            [System.IO.Path]::GetFileName(
                $FilePath
            )
        )

        $request =
            New-Object System.Net.Http.HttpRequestMessage(
                [System.Net.Http.HttpMethod]::Post,
                $Uri
            )

        $request.Headers.Add(
            "X-CSRF-Token",
            $CsrfToken
        )

        $request.Content =
            $multipart

        $response =
            $client.
                SendAsync(
                    $request
                ).
                GetAwaiter().
                GetResult()

        $body =
            $response.
                Content.
                ReadAsStringAsync().
                GetAwaiter().
                GetResult()

        $json = $null

        if (-not [string]::IsNullOrWhiteSpace($body)) {
            try {
                $json =
                    $body |
                        ConvertFrom-Json
            }
            catch {
            }
        }

        return [PSCustomObject]@{
            Status = [int]$response.StatusCode
            Body   = $body
            Json   = $json
        }
    }
    finally {
        if ($null -ne $response) {
            $response.Dispose()
        }

        if ($null -ne $request) {
            $request.Dispose()
        }

        if ($null -ne $multipart) {
            $multipart.Dispose()
        }

        if ($null -ne $fileContent) {
            $fileContent.Dispose()
        }

        if ($null -ne $fileStream) {
            $fileStream.Dispose()
        }

        if ($null -ne $client) {
            $client.Dispose()
        }

        if ($null -ne $handler) {
            $handler.Dispose()
        }
    }
}

function Invoke-GoWorkbookHelper {
    param(
        [string]$HelperPath,
        [string[]]$Arguments
    )

    $goArguments =
        @(
            "run",
            $HelperPath
        ) + $Arguments

    & go $goArguments

    if ($LASTEXITCODE -ne 0) {
        throw "Finance workbook helper failed"
    }
}

function Void-FinanceExpense {
    param(
        [string]$ExpenseID,
        [string]$BaseUrl,
        [Microsoft.PowerShell.Commands.WebRequestSession]$WebSession,
        [hashtable]$Headers,
        [string]$Reason
    )

    if ([string]::IsNullOrWhiteSpace($ExpenseID)) {
        return
    }

    Invoke-RestMethod `
        -Method POST `
        -Uri "$BaseUrl/api/v1/admin/finance/expenses/$ExpenseID/void" `
        -WebSession $WebSession `
        -Headers $Headers `
        -ContentType "application/json" `
        -Body (
            @{
                reason = $Reason
            } |
                ConvertTo-Json
        ) |
        Out-Null
}

function Restore-VariantCost {
    param(
        [string]$BaseUrl,
        [Microsoft.PowerShell.Commands.WebRequestSession]$WebSession,
        [hashtable]$Headers,
        [string]$VariantID,
        [string]$SKU,
        [Int64]$OriginalCost,
        [string]$Currency,
        [string]$Reference
    )

    $body =
        @{
            variant_id =
                $VariantID

            sku =
                $SKU

            unit_cost_amount =
                $OriginalCost

            currency =
                $Currency

            effective_at =
                [DateTime]::UtcNow.ToString(
                    "o"
                )

            description =
                "Automated finance E2E current-cost restoration"

            reference =
                $Reference

            set_current =
                $true
        } |
            ConvertTo-Json -Depth 10

    return Invoke-RestMethod `
        -Method POST `
        -Uri "$BaseUrl/api/v1/admin/finance/product-costs" `
        -WebSession $WebSession `
        -Headers $Headers `
        -ContentType "application/json" `
        -Body $body
}

# ============================================================
# Setup
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
    "FIN-E2E-" +
    (Get-Date -Format "yyyyMMdd-HHmmss") +
    "-" +
    [Guid]::NewGuid().
        ToString("N").
        Substring(0, 6)

$tempRoot =
    Join-Path `
        ([System.IO.Path]::GetTempPath()) `
        $runTag

New-Item `
    -ItemType Directory `
    -Path $tempRoot `
    -Force |
    Out-Null

$templatePath =
    Join-Path `
        $tempRoot `
        "finance-template.xlsx"

$invalidWorkbookPath =
    Join-Path `
        $tempRoot `
        "finance-invalid-$runTag.xlsx"

$validWorkbookPath =
    Join-Path `
        $tempRoot `
        "finance-valid-$runTag.xlsx"

$goHelperPath =
    Join-Path `
        $projectRoot `
        (
            "zz_finance_workbook_" +
            [Guid]::NewGuid().
                ToString("N") +
            ".go"
        )

$utf8NoBom =
    New-Object System.Text.UTF8Encoding(
        $false
    )

$goHelperSource = @'
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/xuri/excelize/v2"
)

var headers = []string{
	"record_type",
	"date",
	"category",
	"scope",
	"amount",
	"unit_cost_amount",
	"currency",
	"product_code",
	"sku",
	"order_id",
	"staff_email",
	"warehouse_code",
	"description",
	"reference",
	"set_current",
}

func main() {
	mode := flag.String("mode", "", "check-template, invalid, or valid")
	in := flag.String("in", "", "input workbook")
	out := flag.String("out", "", "output workbook")
	date := flag.String("date", "", "business date")
	tag := flag.String("tag", "", "test tag")
	staffEmail := flag.String("staff-email", "", "staff email")
	sku := flag.String("sku", "", "variant SKU")
	currency := flag.String("currency", "", "variant currency")
	newCost := flag.Int64("new-cost", 0, "new unit cost")
	flag.Parse()

	switch *mode {
	case "check-template":
		if err := checkTemplate(*in); err != nil {
			fatal(err)
		}
		fmt.Println("finance template OK")

	case "invalid":
		rows := [][]any{
			{
				"expense",
				*date,
				"salary",
				"staff",
				int64(99101),
				"",
				"BDT",
				"",
				"",
				"",
				"missing-" + strings.ToLower(*tag) + "@example.invalid",
				"",
				"Intentional invalid finance E2E salary row",
				*tag + "-INVALID",
				"",
			},
		}

		if err := writeWorkbook(*out, rows); err != nil {
			fatal(err)
		}

	case "valid":
		if strings.TrimSpace(*staffEmail) == "" {
			fatal(fmt.Errorf("staff-email is required"))
		}

		if strings.TrimSpace(*sku) == "" {
			fatal(fmt.Errorf("sku is required"))
		}

		if strings.TrimSpace(*currency) == "" {
			fatal(fmt.Errorf("currency is required"))
		}

		rows := [][]any{
			{
				"expense",
				*date,
				"salary",
				"staff",
				int64(91001),
				"",
				"BDT",
				"",
				"",
				"",
				*staffEmail,
				"",
				"Automated finance E2E salary expense",
				*tag + "-SALARY",
				"",
			},
			{
				"expense",
				*date,
				"marketing",
				"business",
				int64(92002),
				"",
				"BDT",
				"",
				"",
				"",
				"",
				"",
				"Automated finance E2E marketing expense",
				*tag + "-MARKETING",
				"",
			},
			{
				"expense",
				*date,
				"packaging",
				"variant",
				int64(93003),
				"",
				*currency,
				"",
				*sku,
				"",
				"",
				"",
				"Automated finance E2E SKU expense",
				*tag + "-SKU-EXPENSE",
				"",
			},
			{
				"variant_cost",
				*date,
				"",
				"",
				"",
				*newCost,
				*currency,
				"",
				*sku,
				"",
				"",
				"",
				"Automated finance E2E buying cost",
				*tag + "-COST",
				"TRUE",
			},
		}

		if err := writeWorkbook(*out, rows); err != nil {
			fatal(err)
		}

	default:
		fatal(fmt.Errorf("unsupported mode %q", *mode))
	}
}

func checkTemplate(path string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("template path is required")
	}

	book, err := excelize.OpenFile(path)
	if err != nil {
		return fmt.Errorf("open template: %w", err)
	}
	defer book.Close()

	rows, err := book.GetRows("Finance")
	if err != nil {
		return fmt.Errorf("read Finance sheet: %w", err)
	}

	if len(rows) == 0 {
		return fmt.Errorf("Finance sheet has no header")
	}

	if len(rows[0]) < len(headers) {
		return fmt.Errorf(
			"Finance sheet has %d headers, expected at least %d",
			len(rows[0]),
			len(headers),
		)
	}

	for i, expected := range headers {
		actual := strings.TrimSpace(rows[0][i])

		if actual != expected {
			return fmt.Errorf(
				"header %d = %q, expected %q",
				i+1,
				actual,
				expected,
			)
		}
	}

	return nil
}

func writeWorkbook(path string, rows [][]any) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("output path is required")
	}

	book := excelize.NewFile()
	defer book.Close()

	defaultSheet := book.GetSheetName(0)

	if defaultSheet == "" {
		return fmt.Errorf("default sheet missing")
	}

	if err := book.SetSheetName(defaultSheet, "Finance"); err != nil {
		return err
	}

	for columnIndex, header := range headers {
		cell, err := excelize.CoordinatesToCellName(
			columnIndex+1,
			1,
		)
		if err != nil {
			return err
		}

		if err := book.SetCellValue(
			"Finance",
			cell,
			header,
		); err != nil {
			return err
		}
	}

	for rowIndex, row := range rows {
		for columnIndex, value := range row {
			cell, err := excelize.CoordinatesToCellName(
				columnIndex+1,
				rowIndex+2,
			)
			if err != nil {
				return err
			}

			if err := book.SetCellValue(
				"Finance",
				cell,
				value,
			); err != nil {
				return err
			}
		}
	}

	if err := book.SetPanes(
		"Finance",
		&excelize.Panes{
			Freeze:      true,
			Split:       false,
			YSplit:      1,
			TopLeftCell: "A2",
			ActivePane:  "bottomLeft",
		},
	); err != nil {
		return err
	}

	return book.SaveAs(path)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
'@

[System.IO.File]::WriteAllText(
    $goHelperPath,
    $goHelperSource,
    $utf8NoBom
)

$adminSession = $null
$adminHeaders = $null
$csrf = $null
$adminStaff = $null

$manualExpenseID = $null
$invalidBatchID = $null
$validBatchID = $null

$importExpenseIDs =
    @()

$voidedExpenseIDs =
    @{}

$variantID = $null
$variantSKU = $null
$variantCurrency = $null
$originalCost = $null
$newCost = $null

$currentCostChanged =
    $false

$costRestored =
    $false

$locationPushed =
    $false

try {
    Push-Location $projectRoot
    $locationPushed = $true

    Write-Host ""
    Write-Host "Commerce Finance Intelligence Integration Test" -ForegroundColor White
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

    $adminCredential =
        Get-Credential `
            -Message "Commerce Admin Login"

    if ($null -eq $adminCredential) {
        throw "Admin credential entry was cancelled"
    }

    $adminIdentifier =
        $adminCredential.UserName

    $adminPassword =
        $adminCredential.
            GetNetworkCredential().
            Password

    $adminSession =
        New-Object `
            Microsoft.PowerShell.Commands.WebRequestSession

    $adminLoginBody =
        @{
            identifier =
                $adminIdentifier

            password =
                $adminPassword
        } |
            ConvertTo-Json

    $adminLogin =
        Invoke-RestMethod `
            -Method POST `
            -Uri "$BaseUrl/api/v1/admin/auth/login" `
            -WebSession $adminSession `
            -ContentType "application/json" `
            -Body $adminLoginBody

    $adminPassword = $null
    $adminCredential = $null

    Assert-True `
        (
            -not [string]::IsNullOrWhiteSpace(
                [string]$adminLogin.data.challenge_token
            )
        ) `
        "Admin login challenge was issued"

    if ($adminLogin.data.mfa_enrollment_required) {
        throw @"
Admin account requires MFA enrollment.
Complete MFA enrollment first, then rerun the finance test.
"@
    }

    Write-Pass "Admin MFA is already enrolled"

    # ========================================================
    # 3. MFA
    # ========================================================

    Write-Step "3. Verify Admin MFA"

    $totp = ""

    do {
        $totpSecure =
            Read-Host `
                "Enter current 6-digit Admin TOTP" `
                -AsSecureString

        $totp =
            [System.Net.NetworkCredential]::new(
                "",
                $totpSecure
            ).Password

        $totpSecure = $null
    }
    while ($totp -notmatch '^\d{6}$')

    $adminMfaBody =
        @{
            challenge_token =
                $adminLogin.data.challenge_token

            method =
                "totp"

            code =
                $totp
        } |
            ConvertTo-Json

    $adminMfa =
        Invoke-RestMethod `
            -Method POST `
            -Uri "$BaseUrl/api/v1/admin/auth/mfa/verify" `
            -WebSession $adminSession `
            -ContentType "application/json" `
            -Body $adminMfaBody

    $totp = $null

    $csrf =
        [string]$adminMfa.data.csrf_token

    Assert-True `
        (
            -not [string]::IsNullOrWhiteSpace(
                $csrf
            )
        ) `
        "Admin CSRF token was issued"

    $adminStaff =
        $adminMfa.data.principal.staff

    Assert-True `
        (
            -not [string]::IsNullOrWhiteSpace(
                [string]$adminStaff.id
            )
        ) `
        "Authenticated Admin staff ID is available"

    Assert-True `
        (
            -not [string]::IsNullOrWhiteSpace(
                [string]$adminStaff.email
            )
        ) `
        "Authenticated Admin staff email is available"

    $permissions =
        @(
            $adminStaff.permissions
        )

    Assert-True `
        (
            $permissions -contains
                "admin.finance.read"
        ) `
        "Admin has admin.finance.read"

    Assert-True `
        (
            $permissions -contains
                "admin.finance.manage"
        ) `
        "Admin has admin.finance.manage"

    $adminHeaders =
        @{
            "X-CSRF-Token" =
                $csrf
        }

    Write-Info "Admin staff code: $($adminStaff.staff_code)"
    Write-Info "Admin name:       $($adminStaff.full_name)"

    # ========================================================
    # 4. Select reversible SKU fixture
    # ========================================================

    Write-Step "4. Select existing SKU with current buying cost"

    $variantJson =
        Invoke-DockerPsqlScalar `
            -DbUser $dbUser `
            -DbName $dbName `
            -Sql @"
SELECT json_build_object(
    'id', id::text,
    'product_id', product_id::text,
    'sku', sku,
    'currency', currency,
    'cost_amount', cost_amount
)::text
FROM product_variants
WHERE
    cost_amount IS NOT NULL
    AND length(trim(sku)) > 0
ORDER BY
    is_active DESC,
    updated_at DESC,
    id
LIMIT 1;
"@

    Assert-True `
        (
            -not [string]::IsNullOrWhiteSpace(
                $variantJson
            )
        ) `
        "A reversible variant with existing cost_amount exists"

    $variantFixture =
        $variantJson |
            ConvertFrom-Json

    $variantID =
        [string]$variantFixture.id

    $variantSKU =
        [string]$variantFixture.sku

    $variantCurrency =
        [string]$variantFixture.currency

    $originalCost =
        [Int64]$variantFixture.cost_amount

    $newCost =
        $originalCost +
        [Int64](
            Get-Random `
                -Minimum 31 `
                -Maximum 97
        )

    Assert-True `
        ($newCost -ne $originalCost) `
        "Test buying cost differs from current buying cost"

    Write-Info "Test SKU:           $variantSKU"
    Write-Info "Variant ID:         $variantID"
    Write-Info "Currency:           $variantCurrency"
    Write-Info "Original unit cost: $originalCost"
    Write-Info "Test unit cost:     $newCost"

    # ========================================================
    # 5. Manual scoped expense + idempotency
    # ========================================================

    Write-Step "5. Test manual staff-scoped finance expense"

    $manualIdempotencyKey =
        "$runTag-MANUAL-EXPENSE"

    $occurredAt =
        [DateTime]::UtcNow.
            ToString("o")

    $manualExpenseBody =
        @{
            category =
                "salary"

            amount =
                [Int64]94004

            currency =
                "BDT"

            occurred_at =
                $occurredAt

            description =
                "Automated finance E2E manual staff expense"

            staff_account_id =
                [string]$adminStaff.id

            reference =
                "$runTag-MANUAL-SALARY"
        } |
            ConvertTo-Json -Depth 10

    $manualHeaders =
        @{
            "X-CSRF-Token" =
                $csrf

            "Idempotency-Key" =
                $manualIdempotencyKey
        }

    $manualCreate =
        Invoke-RestMethod `
            -Method POST `
            -Uri "$BaseUrl/api/v1/admin/finance/expenses" `
            -WebSession $adminSession `
            -Headers $manualHeaders `
            -ContentType "application/json" `
            -Body $manualExpenseBody

    $manualExpenseID =
        [string]$manualCreate.data.expense.id

    Assert-True `
        ([bool]$manualCreate.data.inserted) `
        "Manual finance expense was inserted"

    Assert-Equal `
        $manualCreate.data.expense.scope `
        "staff" `
        "Manual salary expense is staff-scoped"

    Assert-Equal `
        $manualCreate.data.expense.staff_account_id `
        $adminStaff.id `
        "Manual salary expense targets authenticated staff"

    $manualReplay =
        Invoke-RestMethod `
            -Method POST `
            -Uri "$BaseUrl/api/v1/admin/finance/expenses" `
            -WebSession $adminSession `
            -Headers $manualHeaders `
            -ContentType "application/json" `
            -Body $manualExpenseBody

    Assert-True `
        (-not [bool]$manualReplay.data.inserted) `
        "Exact manual expense replay is idempotent"

    Assert-Equal `
        $manualReplay.data.expense.id `
        $manualExpenseID `
        "Exact replay returns the original expense"

    $conflictingExpenseBody =
        @{
            category =
                "salary"

            amount =
                [Int64]94005

            currency =
                "BDT"

            occurred_at =
                $occurredAt

            description =
                "Automated finance E2E manual staff expense"

            staff_account_id =
                [string]$adminStaff.id

            reference =
                "$runTag-MANUAL-SALARY"
        } |
            ConvertTo-Json -Depth 10

    Expect-ApiFailure `
        -Label "Reusing expense Idempotency-Key with changed amount" `
        -ExpectedStatus 409 `
        -ExpectedCode "IDEMPOTENCY_KEY_CONFLICT" `
        -Action {
            Invoke-RestMethod `
                -Method POST `
                -Uri "$BaseUrl/api/v1/admin/finance/expenses" `
                -WebSession $adminSession `
                -Headers $manualHeaders `
                -ContentType "application/json" `
                -Body $conflictingExpenseBody
        }

    # ========================================================
    # 6. Download and validate template
    # ========================================================

    Write-Step "6. Download finance Excel template"

    Invoke-WebRequest `
        -Method GET `
        -Uri "$BaseUrl/api/v1/admin/finance/imports/template" `
        -WebSession $adminSession `
        -UseBasicParsing `
        -OutFile $templatePath

    Assert-True `
        (Test-Path $templatePath) `
        "Finance Excel template was downloaded"

    Assert-True `
        (
            (Get-Item $templatePath).Length -gt 0
        ) `
        "Finance Excel template is non-empty"

    Invoke-GoWorkbookHelper `
        -HelperPath $goHelperPath `
        -Arguments @(
            "-mode",
            "check-template",
            "-in",
            $templatePath
        )

    Write-Pass "Finance Excel template contains the expected Finance headers"

    # ========================================================
    # 7. Build invalid and valid workbooks
    # ========================================================

    Write-Step "7. Build finance import fixtures"

    $businessDate =
        [DateTime]::UtcNow.
            AddHours(6).
            ToString("yyyy-MM-dd")

    Invoke-GoWorkbookHelper `
        -HelperPath $goHelperPath `
        -Arguments @(
            "-mode",
            "invalid",
            "-out",
            $invalidWorkbookPath,
            "-date",
            $businessDate,
            "-tag",
            $runTag
        )

    Invoke-GoWorkbookHelper `
        -HelperPath $goHelperPath `
        -Arguments @(
            "-mode",
            "valid",
            "-out",
            $validWorkbookPath,
            "-date",
            $businessDate,
            "-tag",
            $runTag,
            "-staff-email",
            [string]$adminStaff.email,
            "-sku",
            $variantSKU,
            "-currency",
            $variantCurrency,
            "-new-cost",
            [string]$newCost
        )

    Assert-True `
        (Test-Path $invalidWorkbookPath) `
        "Invalid workbook fixture was created"

    Assert-True `
        (Test-Path $validWorkbookPath) `
        "Valid workbook fixture was created"

    # ========================================================
    # 8. Invalid workbook preview
    # ========================================================

    Write-Step "8. Validate rejection of invalid workbook"

    $invalidUpload =
        Invoke-MultipartWorkbookUpload `
            -Uri "$BaseUrl/api/v1/admin/finance/imports" `
            -WebSession $adminSession `
            -CsrfToken $csrf `
            -FilePath $invalidWorkbookPath

    Assert-Equal `
        $invalidUpload.Status `
        201 `
        "Invalid workbook is accepted for validation preview"

    Assert-True `
        ($null -ne $invalidUpload.Json) `
        "Invalid workbook response is JSON"

    $invalidBatchID =
        [string]$invalidUpload.Json.data.batch.id

    Assert-Equal `
        $invalidUpload.Json.data.batch.status `
        "failed" `
        "Invalid workbook batch is marked failed"

    Assert-Equal `
        ([int]$invalidUpload.Json.data.batch.total_rows) `
        1 `
        "Invalid workbook records one data row"

    Assert-Equal `
        ([int]$invalidUpload.Json.data.batch.invalid_rows) `
        1 `
        "Invalid workbook records one invalid row"

    $invalidPreview =
        Invoke-RestMethod `
            -Method GET `
            -Uri "$BaseUrl/api/v1/admin/finance/imports/$invalidBatchID" `
            -WebSession $adminSession

    Assert-Equal `
        @($invalidPreview.data.rows).Count `
        1 `
        "Invalid import preview exposes one row"

    Assert-Equal `
        $invalidPreview.data.rows[0].status `
        "invalid" `
        "Invalid import row status is invalid"

    Assert-True `
        (
            @(
                $invalidPreview.
                    data.
                    rows[0].
                    error_details.
                    errors
            ).Count -gt 0
        ) `
        "Invalid row exposes validation errors"

    Expect-ApiFailure `
        -Label "Applying invalid finance workbook" `
        -ExpectedStatus 409 `
        -ExpectedCode "FINANCE_IMPORT_NOT_READY" `
        -Action {
            Invoke-RestMethod `
                -Method POST `
                -Uri "$BaseUrl/api/v1/admin/finance/imports/$invalidBatchID/apply" `
                -WebSession $adminSession `
                -Headers $adminHeaders
        }

    # ========================================================
    # 9. Valid workbook preview
    # ========================================================

    Write-Step "9. Upload valid workbook"

    $validUpload =
        Invoke-MultipartWorkbookUpload `
            -Uri "$BaseUrl/api/v1/admin/finance/imports" `
            -WebSession $adminSession `
            -CsrfToken $csrf `
            -FilePath $validWorkbookPath

    Assert-Equal `
        $validUpload.Status `
        201 `
        "Valid workbook creates a new finance import batch"

    Assert-True `
        (-not [bool]$validUpload.Json.data.duplicate) `
        "First valid workbook upload is not a duplicate"

    $validBatchID =
        [string]$validUpload.Json.data.batch.id

    Assert-Equal `
        $validUpload.Json.data.batch.status `
        "ready" `
        "Valid workbook reaches ready status"

    Assert-Equal `
        ([int]$validUpload.Json.data.batch.total_rows) `
        4 `
        "Valid workbook has four rows"

    Assert-Equal `
        ([int]$validUpload.Json.data.batch.valid_rows) `
        4 `
        "All valid workbook rows pass validation"

    Assert-Equal `
        ([int]$validUpload.Json.data.batch.invalid_rows) `
        0 `
        "Valid workbook has zero invalid rows"

    $validPreview =
        Invoke-RestMethod `
            -Method GET `
            -Uri "$BaseUrl/api/v1/admin/finance/imports/$validBatchID" `
            -WebSession $adminSession

    Assert-Equal `
        @($validPreview.data.rows).Count `
        4 `
        "Valid import preview exposes all four rows"

    Assert-Equal `
        @(
            $validPreview.data.rows |
                Where-Object {
                    $_.status -eq "valid"
                }
        ).Count `
        4 `
        "All preview rows are marked valid"

    # ========================================================
    # 10. Duplicate workbook protection
    # ========================================================

    Write-Step "10. Verify duplicate workbook protection"

    $duplicateUpload =
        Invoke-MultipartWorkbookUpload `
            -Uri "$BaseUrl/api/v1/admin/finance/imports" `
            -WebSession $adminSession `
            -CsrfToken $csrf `
            -FilePath $validWorkbookPath

    Assert-Equal `
        $duplicateUpload.Status `
        200 `
        "Exact duplicate workbook returns HTTP 200"

    Assert-True `
        ([bool]$duplicateUpload.Json.data.duplicate) `
        "Exact duplicate workbook reports duplicate=true"

    Assert-Equal `
        $duplicateUpload.Json.data.batch.id `
        $validBatchID `
        "Duplicate upload returns the existing batch"

    # ========================================================
    # 11. Apply valid workbook
    # ========================================================

    Write-Step "11. Apply valid finance workbook"

    $applyResult =
        Invoke-RestMethod `
            -Method POST `
            -Uri "$BaseUrl/api/v1/admin/finance/imports/$validBatchID/apply" `
            -WebSession $adminSession `
            -Headers $adminHeaders

    Assert-Equal `
        $applyResult.data.batch.status `
        "completed" `
        "Valid finance import completes"

    Assert-Equal `
        ([int]$applyResult.data.batch.applied_expenses) `
        3 `
        "Three expense rows were applied"

    Assert-Equal `
        ([int]$applyResult.data.batch.applied_cost_updates) `
        1 `
        "One buying-cost row was applied"

    $currentCostChanged =
        $true

    $appliedPreview =
        Invoke-RestMethod `
            -Method GET `
            -Uri "$BaseUrl/api/v1/admin/finance/imports/$validBatchID" `
            -WebSession $adminSession

    Assert-Equal `
        @(
            $appliedPreview.data.rows |
                Where-Object {
                    $_.status -eq "applied"
                }
        ).Count `
        4 `
        "All four finance import rows are applied"

    $importExpenseIDs =
        @(
            $appliedPreview.data.rows |
                ForEach-Object {
                    [string]$_.applied_expense_id
                } |
                Where-Object {
                    -not [string]::IsNullOrWhiteSpace($_)
                }
        )

    Assert-Equal `
        $importExpenseIDs.Count `
        3 `
        "Three imported expense IDs were persisted"

    $costHistoryID =
        [string](
            $appliedPreview.data.rows |
                Where-Object {
                    -not [string]::IsNullOrWhiteSpace(
                        [string]$_.applied_cost_history_id
                    )
                } |
                Select-Object -First 1
        ).applied_cost_history_id

    Assert-True `
        (
            -not [string]::IsNullOrWhiteSpace(
                $costHistoryID
            )
        ) `
        "Imported buying-cost history ID was persisted"

    # ========================================================
    # 12. Verify imported expense semantics through API
    # ========================================================

    Write-Step "12. Verify imported expenses through API"

    $expenseByReference =
        @{}

    foreach ($expenseID in $importExpenseIDs) {
        $response =
            Invoke-RestMethod `
                -Method GET `
                -Uri "$BaseUrl/api/v1/admin/finance/expenses/$expenseID" `
                -WebSession $adminSession

        $expenseByReference[
            [string]$response.data.reference
        ] =
            $response.data
    }

    $salaryExpense =
        $expenseByReference[
            "$runTag-SALARY"
        ]

    $marketingExpense =
        $expenseByReference[
            "$runTag-MARKETING"
        ]

    $skuExpense =
        $expenseByReference[
            "$runTag-SKU-EXPENSE"
        ]

    Assert-True `
        ($null -ne $salaryExpense) `
        "Imported salary expense exists"

    Assert-Equal `
        $salaryExpense.category `
        "salary" `
        "Imported salary category persisted"

    Assert-Equal `
        $salaryExpense.scope `
        "staff" `
        "Imported salary is staff-scoped"

    Assert-Equal `
        $salaryExpense.staff_account_id `
        $adminStaff.id `
        "Imported salary resolves staff email to staff account"

    Assert-True `
        ($null -ne $marketingExpense) `
        "Imported marketing expense exists"

    Assert-Equal `
        $marketingExpense.scope `
        "business" `
        "Imported marketing expense is business-scoped"

    Assert-True `
        ($null -ne $skuExpense) `
        "Imported SKU expense exists"

    Assert-Equal `
        $skuExpense.scope `
        "variant" `
        "Imported SKU expense is variant-scoped"

    Assert-Equal `
        $skuExpense.variant_id `
        $variantID `
        "Imported SKU expense resolves SKU to correct variant"

    # ========================================================
    # 13. Verify buying-cost update
    # ========================================================

    Write-Step "13. Verify historical buying cost and current cost"

    $costHistory =
        Invoke-RestMethod `
            -Method GET `
            -Uri "$BaseUrl/api/v1/admin/finance/product-costs?variant_id=$variantID&source=import" `
            -WebSession $adminSession

    $importedCost =
        $costHistory.data |
            Where-Object {
                [string]$_.id -eq
                    $costHistoryID
            } |
            Select-Object -First 1

    Assert-True `
        ($null -ne $importedCost) `
        "Imported buying-cost history is readable through API"

    Assert-Equal `
        ([Int64]$importedCost.unit_cost_amount) `
        $newCost `
        "Imported historical buying cost matches workbook"

    Assert-Equal `
        $importedCost.source `
        "import" `
        "Buying-cost source is import"

    Assert-Equal `
        $importedCost.reference `
        "$runTag-COST" `
        "Buying-cost import reference persisted"

    Assert-Equal `
        ([Int64]$importedCost.current_unit_cost_amount) `
        $newCost `
        "set_current updated product_variants.cost_amount"

    # ========================================================
    # 14. Re-apply completed batch
    # ========================================================

    Write-Step "14. Verify completed import re-apply is idempotent"

    $firstAppliedExpenseIDs =
        @(
            $appliedPreview.data.rows |
                ForEach-Object {
                    [string]$_.applied_expense_id
                } |
                Where-Object {
                    -not [string]::IsNullOrWhiteSpace($_)
                } |
                Sort-Object
        )

    $firstAppliedCostIDs =
        @(
            $appliedPreview.data.rows |
                ForEach-Object {
                    [string]$_.applied_cost_history_id
                } |
                Where-Object {
                    -not [string]::IsNullOrWhiteSpace($_)
                } |
                Sort-Object
        )

    $reapply =
        Invoke-RestMethod `
            -Method POST `
            -Uri "$BaseUrl/api/v1/admin/finance/imports/$validBatchID/apply" `
            -WebSession $adminSession `
            -Headers $adminHeaders

    Assert-Equal `
        $reapply.data.batch.status `
        "completed" `
        "Re-applying completed batch remains completed"

    $reappliedPreview =
        Invoke-RestMethod `
            -Method GET `
            -Uri "$BaseUrl/api/v1/admin/finance/imports/$validBatchID" `
            -WebSession $adminSession

    $secondAppliedExpenseIDs =
        @(
            $reappliedPreview.data.rows |
                ForEach-Object {
                    [string]$_.applied_expense_id
                } |
                Where-Object {
                    -not [string]::IsNullOrWhiteSpace($_)
                } |
                Sort-Object
        )

    $secondAppliedCostIDs =
        @(
            $reappliedPreview.data.rows |
                ForEach-Object {
                    [string]$_.applied_cost_history_id
                } |
                Where-Object {
                    -not [string]::IsNullOrWhiteSpace($_)
                } |
                Sort-Object
        )

    Assert-Equal `
        ($secondAppliedExpenseIDs -join ",") `
        ($firstAppliedExpenseIDs -join ",") `
        "Re-apply does not create duplicate expenses"

    Assert-Equal `
        ($secondAppliedCostIDs -join ",") `
        ($firstAppliedCostIDs -join ",") `
        "Re-apply does not create duplicate buying-cost history"

    # ========================================================
    # 15. PostgreSQL verification
    # ========================================================

    Write-Step "15. Verify persistent PostgreSQL state"

    $batchState =
        Invoke-DockerPsqlScalar `
            -DbUser $dbUser `
            -DbName $dbName `
            -Sql @"
SELECT concat_ws(
    '|',
    status,
    total_rows::text,
    valid_rows::text,
    invalid_rows::text,
    applied_expenses::text,
    applied_cost_updates::text
)
FROM finance_import_batches
WHERE id = '$validBatchID'::uuid;
"@

    Assert-Equal `
        $batchState `
        "completed|4|4|0|3|1" `
        "Finance import batch persisted completed counters"

    $appliedRowCount =
        Invoke-DockerPsqlScalar `
            -DbUser $dbUser `
            -DbName $dbName `
            -Sql @"
SELECT count(*)::text
FROM finance_import_rows
WHERE
    batch_id = '$validBatchID'::uuid
    AND status = 'applied';
"@

    Assert-Equal `
        ([int]$appliedRowCount) `
        4 `
        "PostgreSQL contains four applied import rows"

    $salaryCount =
        Invoke-DockerPsqlScalar `
            -DbUser $dbUser `
            -DbName $dbName `
            -Sql @"
SELECT count(*)::text
FROM finance_expenses e
JOIN finance_import_rows r
    ON r.applied_expense_id = e.id
WHERE
    r.batch_id = '$validBatchID'::uuid
    AND e.category = 'salary'
    AND e.staff_account_id = '$($adminStaff.id)'::uuid
    AND e.reference = '$runTag-SALARY'
    AND e.status = 'active';
"@

    Assert-Equal `
        ([int]$salaryCount) `
        1 `
        "PostgreSQL contains one active staff-linked imported salary"

    $variantExpenseCount =
        Invoke-DockerPsqlScalar `
            -DbUser $dbUser `
            -DbName $dbName `
            -Sql @"
SELECT count(*)::text
FROM finance_expenses e
JOIN finance_import_rows r
    ON r.applied_expense_id = e.id
WHERE
    r.batch_id = '$validBatchID'::uuid
    AND e.variant_id = '$variantID'::uuid
    AND e.reference = '$runTag-SKU-EXPENSE'
    AND e.status = 'active';
"@

    Assert-Equal `
        ([int]$variantExpenseCount) `
        1 `
        "PostgreSQL contains one active SKU-linked imported expense"

    $costHistoryCount =
        Invoke-DockerPsqlScalar `
            -DbUser $dbUser `
            -DbName $dbName `
            -Sql @"
SELECT count(*)::text
FROM finance_variant_cost_history h
JOIN finance_import_rows r
    ON r.applied_cost_history_id = h.id
WHERE
    r.batch_id = '$validBatchID'::uuid
    AND h.variant_id = '$variantID'::uuid
    AND h.unit_cost_amount = $newCost
    AND h.source = 'import'
    AND h.reference = '$runTag-COST';
"@

    Assert-Equal `
        ([int]$costHistoryCount) `
        1 `
        "PostgreSQL contains exactly one imported buying-cost history row"

    $databaseCurrentCost =
        Invoke-DockerPsqlScalar `
            -DbUser $dbUser `
            -DbName $dbName `
            -Sql @"
SELECT cost_amount::text
FROM product_variants
WHERE id = '$variantID'::uuid;
"@

    Assert-Equal `
        ([Int64]$databaseCurrentCost) `
        $newCost `
        "PostgreSQL current variant cost matches set_current import"

    # ========================================================
    # 16. Restore current cost
    # ========================================================

    Write-Step "16. Restore original SKU current cost"

    $restore =
        Restore-VariantCost `
            -BaseUrl $BaseUrl `
            -WebSession $adminSession `
            -Headers $adminHeaders `
            -VariantID $variantID `
            -SKU $variantSKU `
            -OriginalCost $originalCost `
            -Currency $variantCurrency `
            -Reference "$runTag-RESTORE"

    Assert-True `
        ([bool]$restore.data.current_cost_updated) `
        "Current SKU cost restoration was applied"

    Assert-Equal `
        ([Int64]$restore.data.cost.current_unit_cost_amount) `
        $originalCost `
        "Current SKU cost returned to original value"

    $costRestored =
        $true

    $currentCostChanged =
        $false

    # ========================================================
    # 17. Void test expenses
    # ========================================================

    Write-Step "17. Void test expenses"

    $allExpenseIDs =
        @(
            $manualExpenseID
        ) +
        @(
            $importExpenseIDs
        )

    foreach ($expenseID in $allExpenseIDs) {
        if ([string]::IsNullOrWhiteSpace($expenseID)) {
            continue
        }

        Void-FinanceExpense `
            -ExpenseID $expenseID `
            -BaseUrl $BaseUrl `
            -WebSession $adminSession `
            -Headers $adminHeaders `
            -Reason "Automated finance E2E cleanup $runTag"

        $voidedExpenseIDs[$expenseID] =
            $true

        Write-Pass "Voided test finance expense $expenseID"
    }

    # ========================================================
    # 18. Verify cleanup
    # ========================================================

    Write-Step "18. Verify financial cleanup"

    $restoredDatabaseCost =
        Invoke-DockerPsqlScalar `
            -DbUser $dbUser `
            -DbName $dbName `
            -Sql @"
SELECT cost_amount::text
FROM product_variants
WHERE id = '$variantID'::uuid;
"@

    Assert-Equal `
        ([Int64]$restoredDatabaseCost) `
        $originalCost `
        "Database current SKU cost is restored"

    $activeTestExpenseCount =
        Invoke-DockerPsqlScalar `
            -DbUser $dbUser `
            -DbName $dbName `
            -Sql @"
SELECT count(*)::text
FROM finance_expenses
WHERE
    reference IN (
        '$runTag-MANUAL-SALARY',
        '$runTag-SALARY',
        '$runTag-MARKETING',
        '$runTag-SKU-EXPENSE'
    )
    AND status = 'active';
"@

    Assert-Equal `
        ([int]$activeTestExpenseCount) `
        0 `
        "No automated E2E expenses remain active"

    # ========================================================
    # Success
    # ========================================================

    Write-Host ""
    Write-Host "============================================================" -ForegroundColor Green
    Write-Host " FINANCE INTELLIGENCE RUNTIME TEST PASSED" -ForegroundColor Green
    Write-Host "============================================================" -ForegroundColor Green
    Write-Host ""

    Write-Host "Run:              $runTag"
    Write-Host "Invalid batch:    $invalidBatchID"
    Write-Host "Completed batch:  $validBatchID"
    Write-Host "Variant ID:       $variantID"
    Write-Host "SKU:              $variantSKU"
    Write-Host ""

    Write-Host "Validated:" -ForegroundColor White
    Write-Host "  Admin finance RBAC"
    Write-Host "  manual scoped expense creation"
    Write-Host "  expense idempotent replay"
    Write-Host "  expense idempotency conflict"
    Write-Host "  Excel template download"
    Write-Host "  Excel template structure"
    Write-Host "  invalid workbook validation preview"
    Write-Host "  invalid import apply rejection"
    Write-Host "  valid workbook validation preview"
    Write-Host "  duplicate workbook SHA-256 protection"
    Write-Host "  atomic finance import apply"
    Write-Host "  staff salary resolution"
    Write-Host "  business expense import"
    Write-Host "  SKU-scoped expense resolution"
    Write-Host "  historical SKU buying-cost import"
    Write-Host "  set_current catalog cost update"
    Write-Host "  completed-batch reapply idempotency"
    Write-Host "  PostgreSQL import persistence"
    Write-Host "  current buying-cost restoration"
    Write-Host "  expense cleanup through voiding"
    Write-Host ""

    Write-Host "No password, session token, CSRF token, or TOTP was printed." -ForegroundColor DarkGray
    Write-Host ""
}
catch {
    Write-Host ""
    Write-Host "============================================================" -ForegroundColor Red
    Write-Host " FINANCE INTELLIGENCE RUNTIME TEST FAILED" -ForegroundColor Red
    Write-Host "============================================================" -ForegroundColor Red
    Write-Host ""

    Write-Host $_.Exception.Message -ForegroundColor Red
    Write-Host ""

    if (-not [string]::IsNullOrWhiteSpace($invalidBatchID)) {
        Write-Host "Invalid batch created before failure:   $invalidBatchID"
    }

    if (-not [string]::IsNullOrWhiteSpace($validBatchID)) {
        Write-Host "Valid batch created before failure:     $validBatchID"
    }

    if (-not [string]::IsNullOrWhiteSpace($variantID)) {
        Write-Host "Variant used before failure:            $variantID"
    }

    Write-Host ""
    Write-Host "Best-effort cleanup will now run." -ForegroundColor Yellow
    Write-Host ""

    exit 1
}
finally {
    # --------------------------------------------------------
    # Best-effort restore if failure happened after set_current
    # --------------------------------------------------------

    if (
        $currentCostChanged -and
        -not $costRestored -and
        $null -ne $adminSession -and
        $null -ne $adminHeaders -and
        -not [string]::IsNullOrWhiteSpace($variantID) -and
        $null -ne $originalCost
    ) {
        try {
            Restore-VariantCost `
                -BaseUrl $BaseUrl `
                -WebSession $adminSession `
                -Headers $adminHeaders `
                -VariantID $variantID `
                -SKU $variantSKU `
                -OriginalCost $originalCost `
                -Currency $variantCurrency `
                -Reference "$runTag-FAILURE-RESTORE" |
                Out-Null

            Write-Warn "Best-effort cleanup restored original SKU current cost."
        }
        catch {
            Write-Warn "Unable to automatically restore SKU current cost: $($_.Exception.Message)"
        }
    }

    # --------------------------------------------------------
    # Best-effort void any uncleaned test expenses
    # --------------------------------------------------------

    if (
        $null -ne $adminSession -and
        $null -ne $adminHeaders
    ) {
        $cleanupExpenseIDs =
            @()

        if (
            -not [string]::IsNullOrWhiteSpace(
                $manualExpenseID
            )
        ) {
            $cleanupExpenseIDs +=
                $manualExpenseID
        }

        $cleanupExpenseIDs +=
            @(
                $importExpenseIDs
            )

        foreach ($expenseID in $cleanupExpenseIDs) {
            if (
                [string]::IsNullOrWhiteSpace(
                    $expenseID
                )
            ) {
                continue
            }

            if ($voidedExpenseIDs.ContainsKey($expenseID)) {
                continue
            }

            try {
                Void-FinanceExpense `
                    -ExpenseID $expenseID `
                    -BaseUrl $BaseUrl `
                    -WebSession $adminSession `
                    -Headers $adminHeaders `
                    -Reason "Automated finance E2E failure cleanup $runTag"

                Write-Warn "Best-effort cleanup voided expense $expenseID."
            }
            catch {
                Write-Warn "Unable to automatically void expense $expenseID."
            }
        }
    }

    $csrf = $null
    $totp = $null
    $adminPassword = $null
    $adminCredential = $null

    if (Test-Path $goHelperPath) {
        Remove-Item `
            -Path $goHelperPath `
            -Force `
            -ErrorAction SilentlyContinue
    }

    if (Test-Path $tempRoot) {
        Remove-Item `
            -Path $tempRoot `
            -Recurse `
            -Force `
            -ErrorAction SilentlyContinue
    }

    if ($locationPushed) {
        Pop-Location
    }
}