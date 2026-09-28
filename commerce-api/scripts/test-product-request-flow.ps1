param(
    [string]$BaseUrl = "http://127.0.0.1:8081",

    [switch]$SkipDatabaseVerification
)

$ErrorActionPreference = "Stop"

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

function Assert-False {
    param(
        [bool]$Condition,

        [string]$Message
    )

    if ($Condition) {
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

function Read-SecretText {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Prompt
    )

    $secure =
        Read-Host `
            $Prompt `
            -AsSecureString

    $pointer =
        [Runtime.InteropServices.Marshal]::
            SecureStringToBSTR(
                $secure
            )

    try {
        return [Runtime.InteropServices.Marshal]::
            PtrToStringBSTR(
                $pointer
            )
    }
    finally {
        [Runtime.InteropServices.Marshal]::
            ZeroFreeBSTR(
                $pointer
            )
    }
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
                    New-Object `
                        System.IO.StreamReader(
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

    if (
        -not [string]::IsNullOrWhiteSpace(
            $body
        )
    ) {
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

        [string]$ExpectedCode = "",

        [Parameter(Mandatory = $true)]
        [string]$Label
    )

    $caught = $null

    try {
        & $Action |
            Out-Null
    }
    catch {
        $caught = $_
    }

    if ($null -eq $caught) {
        throw "${Label}: request succeeded but HTTP $ExpectedStatus failure was expected"
    }

    $failure =
        Get-HttpFailure `
            $caught

    if ($failure.Status -ne $ExpectedStatus) {
        throw @"
${Label}: wrong HTTP status
Expected: $ExpectedStatus
Actual  : $($failure.Status)
Body    : $($failure.Body)
"@
    }

    if (
        -not [string]::IsNullOrWhiteSpace(
            $ExpectedCode
        )
    ) {
        if (
            $null -eq $failure.Json -or
            $null -eq $failure.Json.error -or
            [string]$failure.Json.error.code -ne
                $ExpectedCode
        ) {
            throw @"
${Label}: wrong API error code
Expected: $ExpectedCode
Body    : $($failure.Body)
"@
        }
    }

    Write-Pass "$Label returned expected HTTP $ExpectedStatus / $ExpectedCode"
}

function Import-ProjectDotEnv {
    param(
        [string]$Path
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

        [Environment]::
            SetEnvironmentVariable(
                $key,
                $value,
                "Process"
            )
    }
}

function Get-ItemById {
    param(
        $Items,

        [string]$ID
    )

    return @(
        $Items |
            Where-Object {
                [string]$_.id -eq $ID
            }
    ) |
        Select-Object -First 1
}

# ============================================================
# Start
# ============================================================

Write-Host ""
Write-Host "Commerce Product Request + Sourcing Offer Integration Test" -ForegroundColor White
Write-Host "Base URL: $BaseUrl" -ForegroundColor White
Write-Host ""

$runTag =
    "E2E-" +
    (Get-Date -Format "yyyyMMdd-HHmmss")

$requestAId = $null
$requestBId = $null

$offerAId = $null
$offerBId = $null
$confirmationId = $null

$customerToken = $null
$csrf = $null
$totp = $null
$customerOtp = $null

try {
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

    Write-Info "PostgreSQL: $($health.dependencies.postgres)"
    Write-Info "Redis:      $($health.dependencies.redis)"

    # ========================================================
    # 2. Create isolated customer
    # ========================================================

    Write-Step "2. Create test customer"

    $randomPhoneSuffix =
        Get-Random `
            -Minimum 10000000 `
            -Maximum 99999999

    $customerPhone =
        "+88019$randomPhoneSuffix"

    $customerPassword =
        "Tmp!" +
        [Guid]::
            NewGuid().
            ToString("N").
            Substring(
                0,
                24
            )

    $registerBody = @{
        phone =
            $customerPhone

        email =
            ""

        password =
            $customerPassword

        full_name =
            "Product Request E2E Customer"
    } |
        ConvertTo-Json `
            -Depth 10

    $register =
        Invoke-RestMethod `
            -Method POST `
            -Uri "$BaseUrl/api/v1/auth/register" `
            -ContentType "application/json" `
            -Body $registerBody

    if ($register.data.verification_required) {
        Write-Warn "Customer phone verification is enabled."

        do {
            $customerOtp =
                Read-SecretText `
                    "Enter customer registration OTP shown by the API/Air log"
        }
        while (
            $customerOtp -notmatch
            '^\d{4,8}$'
        )

        $verifyBody = @{
            verification_id =
                $register.data.verification.verification_id

            code =
                $customerOtp
        } |
            ConvertTo-Json

        $verified =
            Invoke-RestMethod `
                -Method POST `
                -Uri "$BaseUrl/api/v1/auth/register/verify" `
                -ContentType "application/json" `
                -Body $verifyBody

        $customerOtp = $null

        $customerToken =
            $verified.data.tokens.access_token

        $customerId =
            $verified.data.customer.id
    }
    else {
        $customerToken =
            $register.data.tokens.access_token

        $customerId =
            $register.data.customer.id
    }

    $customerPassword = $null

    Assert-True `
        (-not [string]::IsNullOrWhiteSpace(
            $customerToken
        )) `
        "Customer access token was issued"

    Assert-True `
        (-not [string]::IsNullOrWhiteSpace(
            $customerId
        )) `
        "Customer account was created"

    Write-Info "Customer ID: $customerId"

    $customerHeaders = @{
        Authorization =
            "Bearer $customerToken"
    }

    # ========================================================
    # 3. Initial customer list
    # ========================================================

    Write-Step "3. Test customer product-request list"

    $customerInitialList =
        Invoke-RestMethod `
            -Method GET `
            -Uri "$BaseUrl/api/v1/product-requests" `
            -Headers $customerHeaders

    Assert-Equal `
        $customerInitialList.meta.limit `
        20 `
        "Customer request list default limit is 20"

    # ========================================================
    # 4. Create primary request
    # ========================================================

    Write-Step "4. Create primary product request"

    $productAName =
        "Sony WH-1000XM6 $runTag"

    $createABody = @{
        requested_product_name =
            $productAName

        description =
            "Primary automated sourcing request for $runTag."

        requested_quantity =
            2

        customer_requirements = @{
            color =
                "black"

            condition =
                "new"

            region =
                "international"

            test_run =
                $runTag
        }

        external_url =
            "https://www.sony.com/"

        attachments =
            @()
    } |
        ConvertTo-Json `
            -Depth 10

    $createA =
        Invoke-RestMethod `
            -Method POST `
            -Uri "$BaseUrl/api/v1/product-requests" `
            -Headers $customerHeaders `
            -ContentType "application/json" `
            -Body $createABody

    $requestA =
        $createA.data

    $requestAId =
        $requestA.id

    Assert-True `
        (-not [string]::IsNullOrWhiteSpace(
            $requestAId
        )) `
        "Primary product request ID was returned"

    Assert-Equal `
        $requestA.status `
        "pending_review" `
        "New request starts as pending_review"

    Assert-Equal `
        $requestA.requested_quantity `
        2 `
        "Requested quantity persisted"

    Assert-True `
        ([bool]$requestA.can_message) `
        "New request allows customer messages"

    Assert-False `
        ($requestA.PSObject.Properties.Name -contains "case_id") `
        "Customer response does not expose internal case_id"

    Write-Info "Primary request ID: $requestAId"
    Write-Info "Request number:     $($requestA.request_number)"

    # ========================================================
    # 5. Customer detail + follow-up
    # ========================================================

    Write-Step "5. Test customer detail and messaging"

    $customerDetailA =
        Invoke-RestMethod `
            -Method GET `
            -Uri "$BaseUrl/api/v1/product-requests/$requestAId" `
            -Headers $customerHeaders

    Assert-Equal `
        $customerDetailA.data.id `
        $requestAId `
        "Customer can load own request"

    $customerFollowupText =
        "Customer follow-up message for $runTag"

    $customerFollowupBody = @{
        message =
            $customerFollowupText

        attachments =
            @()
    } |
        ConvertTo-Json `
            -Depth 10

    $customerFollowup =
        Invoke-RestMethod `
            -Method POST `
            -Uri "$BaseUrl/api/v1/product-requests/$requestAId/messages" `
            -Headers $customerHeaders `
            -ContentType "application/json" `
            -Body $customerFollowupBody

    Assert-Equal `
        $customerFollowup.data.author_type `
        "customer" `
        "Customer follow-up author is customer"

    $customerMessagesA =
        Invoke-RestMethod `
            -Method GET `
            -Uri "$BaseUrl/api/v1/product-requests/$requestAId/messages" `
            -Headers $customerHeaders

    Assert-Equal `
        @($customerMessagesA.data).Count `
        2 `
        "Customer conversation initially contains two messages"

    # ========================================================
    # 6. Admin login
    # ========================================================

    Write-Step "6. Admin login"

    Write-Host "A credential window will open." -ForegroundColor Yellow
    Write-Host "Use the Commerce Admin email/staff code and Admin password." -ForegroundColor Yellow
    Write-Host "Do NOT use Windows credentials." -ForegroundColor Yellow
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

    $adminLoginBody = @{
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
        (-not [string]::IsNullOrWhiteSpace(
            [string]$adminLogin.data.challenge_token
        )) `
        "Admin login challenge was issued"

    if ($adminLogin.data.mfa_enrollment_required) {
        throw @"
Admin account requires MFA enrollment.

Enroll MFA first, then rerun this script.
"@
    }

    Write-Pass "Admin MFA is already enrolled"

    # ========================================================
    # 7. Admin MFA
    # ========================================================

    Write-Step "7. Verify Admin MFA"

    do {
        $totp =
            Read-SecretText `
                "Enter current 6-digit Admin TOTP"
    }
    while (
        $totp -notmatch
        '^\d{6}$'
    )

    $adminMfaBody = @{
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
        (-not [string]::IsNullOrWhiteSpace(
            $csrf
        )) `
        "Admin CSRF token was issued"

    $adminStaff =
        $adminMfa.data.principal.staff

    Write-Info "Admin staff code: $($adminStaff.staff_code)"
    Write-Info "Admin name:       $($adminStaff.full_name)"

    $adminPermissions =
        @($adminStaff.permissions)

    Assert-True `
        ($adminPermissions -contains "admin.sourcing.read") `
        "Admin has admin.sourcing.read"

    Assert-True `
        ($adminPermissions -contains "admin.sourcing.review") `
        "Admin has admin.sourcing.review"

    Assert-True `
        ($adminPermissions -contains "admin.sourcing.offer.manage") `
        "Admin has admin.sourcing.offer.manage"

    Assert-True `
        ($adminPermissions -contains "admin.sourcing.finalize") `
        "Admin has admin.sourcing.finalize"

    $adminHeaders = @{
        "X-CSRF-Token" =
            $csrf
    }

    # ========================================================
    # 8. Admin session
    # ========================================================

    Write-Step "8. Verify Admin session"

    $adminMe =
        Invoke-RestMethod `
            -Method GET `
            -Uri "$BaseUrl/api/v1/admin/auth/me" `
            -WebSession $adminSession

    Assert-Equal `
        $adminMe.data.principal.staff.id `
        $adminStaff.id `
        "Admin /auth/me returns authenticated staff account"

    # ========================================================
    # 9. Admin request list
    # ========================================================

    Write-Step "9. Test Admin product-request list"

    $encodedRunTag =
        [Uri]::
            EscapeDataString(
                $runTag
            )

    $adminList =
        Invoke-RestMethod `
            -Method GET `
            -Uri "$BaseUrl/api/v1/admin/product-requests?q=$encodedRunTag" `
            -WebSession $adminSession

    $adminListIds =
        @(
            $adminList.data |
                ForEach-Object {
                    $_.id
                }
        )

    Assert-True `
        ($adminListIds -contains $requestAId) `
        "Admin list contains primary request"

    # ========================================================
    # 10. Admin detail
    # ========================================================

    Write-Step "10. Test Admin product-request detail"

    $adminDetailA =
        Invoke-RestMethod `
            -Method GET `
            -Uri "$BaseUrl/api/v1/admin/product-requests/$requestAId" `
            -WebSession $adminSession

    Assert-Equal `
        $adminDetailA.data.id `
        $requestAId `
        "Admin can load request detail"

    Assert-Equal `
        $adminDetailA.data.status `
        "pending_review" `
        "Admin sees pending_review"

    Assert-True `
        (-not [string]::IsNullOrWhiteSpace(
            [string]$adminDetailA.data.case_id
        )) `
        "Admin response exposes internal case_id"

    Assert-Equal `
        $adminDetailA.data.assignment.queue_code `
        "bulk_sales" `
        "Product request is assigned to bulk_sales queue"

    # ========================================================
    # 11. Admin conversation
    # ========================================================

    Write-Step "11. Test Admin conversation"

    $adminMessagesBefore =
        Invoke-RestMethod `
            -Method GET `
            -Uri "$BaseUrl/api/v1/admin/product-requests/$requestAId/messages" `
            -WebSession $adminSession

    Assert-Equal `
        @($adminMessagesBefore.data).Count `
        2 `
        "Admin initially sees two CRM messages"

    # ========================================================
    # 12. Internal note
    # ========================================================

    Write-Step "12. Add internal Admin note"

    $internalNoteText =
        "Internal sourcing note for $runTag"

    $internalNoteBody = @{
        message =
            $internalNoteText

        visibility =
            "internal"

        attachments =
            @()
    } |
        ConvertTo-Json `
            -Depth 10

    $internalNote =
        Invoke-RestMethod `
            -Method POST `
            -Uri "$BaseUrl/api/v1/admin/product-requests/$requestAId/messages" `
            -WebSession $adminSession `
            -Headers $adminHeaders `
            -ContentType "application/json" `
            -Body $internalNoteBody

    Assert-Equal `
        $internalNote.data.visibility `
        "internal" `
        "Internal Admin note was created"

    $customerMessagesAfterInternal =
        Invoke-RestMethod `
            -Method GET `
            -Uri "$BaseUrl/api/v1/product-requests/$requestAId/messages" `
            -Headers $customerHeaders

    $customerBodiesAfterInternal =
        @(
            $customerMessagesAfterInternal.data |
                ForEach-Object {
                    $_.body
                }
        )

    Assert-False `
        ($customerBodiesAfterInternal -contains $internalNoteText) `
        "Internal Admin note is hidden from customer"

    Assert-Equal `
        @($customerMessagesAfterInternal.data).Count `
        2 `
        "Internal note does not increase customer-visible message count"

    # ========================================================
    # 13. Customer-visible Admin reply
    # ========================================================

    Write-Step "13. Send customer-visible Admin reply"

    $adminReplyText =
        "We are reviewing sourcing options for $runTag."

    $adminReplyBody = @{
        message =
            $adminReplyText

        visibility =
            "customer"

        attachments =
            @()
    } |
        ConvertTo-Json `
            -Depth 10

    $adminReply =
        Invoke-RestMethod `
            -Method POST `
            -Uri "$BaseUrl/api/v1/admin/product-requests/$requestAId/messages" `
            -WebSession $adminSession `
            -Headers $adminHeaders `
            -ContentType "application/json" `
            -Body $adminReplyBody

    Assert-Equal `
        $adminReply.data.visibility `
        "customer" `
        "Customer-visible Admin reply was created"

    Assert-Equal `
        $adminReply.data.author_type `
        "support" `
        "Admin reply is attributed to support"

    Assert-Equal `
        $adminReply.data.support_actor.actor_type `
        "human" `
        "Admin was resolved/provisioned as human support actor"

    $customerMessagesAfterReply =
        Invoke-RestMethod `
            -Method GET `
            -Uri "$BaseUrl/api/v1/product-requests/$requestAId/messages" `
            -Headers $customerHeaders

    $customerBodiesAfterReply =
        @(
            $customerMessagesAfterReply.data |
                ForEach-Object {
                    $_.body
                }
        )

    Assert-True `
        ($customerBodiesAfterReply -contains $adminReplyText) `
        "Customer can see customer-visible Admin reply"

    Assert-False `
        ($customerBodiesAfterReply -contains $internalNoteText) `
        "Customer still cannot see internal note"

    Assert-Equal `
        @($customerMessagesAfterReply.data).Count `
        3 `
        "Customer sees exactly three visible messages"

    $adminMessagesAfterReply =
        Invoke-RestMethod `
            -Method GET `
            -Uri "$BaseUrl/api/v1/admin/product-requests/$requestAId/messages" `
            -WebSession $adminSession

    Assert-Equal `
        @($adminMessagesAfterReply.data).Count `
        4 `
        "Admin sees customer messages plus internal note plus Admin reply"

    # ========================================================
    # 14. Lifecycle protections
    # ========================================================

    Write-Step "14. Verify lifecycle protections"

    Expect-ApiFailure `
        -Label "Direct pending_review -> negotiating transition" `
        -ExpectedStatus 400 `
        -ExpectedCode "INVALID_PRODUCT_REQUEST" `
        -Action {
            Invoke-RestMethod `
                -Method PATCH `
                -Uri "$BaseUrl/api/v1/admin/product-requests/$requestAId/status" `
                -WebSession $adminSession `
                -Headers $adminHeaders `
                -ContentType "application/json" `
                -Body (@{
                    status =
                        "negotiating"
                } | ConvertTo-Json)
        }

    Expect-ApiFailure `
        -Label "on_hold without reason" `
        -ExpectedStatus 400 `
        -ExpectedCode "INVALID_PRODUCT_REQUEST" `
        -Action {
            Invoke-RestMethod `
                -Method PATCH `
                -Uri "$BaseUrl/api/v1/admin/product-requests/$requestAId/status" `
                -WebSession $adminSession `
                -Headers $adminHeaders `
                -ContentType "application/json" `
                -Body (@{
                    status =
                        "on_hold"
                } | ConvertTo-Json)
        }

    # ========================================================
    # 15. on_hold
    # ========================================================

    Write-Step "15. Move primary request to on_hold"

    $holdReason =
        "Searching approved suppliers for $runTag"

    $holdResponse =
        Invoke-RestMethod `
            -Method PATCH `
            -Uri "$BaseUrl/api/v1/admin/product-requests/$requestAId/status" `
            -WebSession $adminSession `
            -Headers $adminHeaders `
            -ContentType "application/json" `
            -Body (@{
                status =
                    "on_hold"

                reason =
                    $holdReason
            } | ConvertTo-Json)

    Assert-Equal `
        $holdResponse.data.status `
        "on_hold" `
        "Request moved to on_hold"

    Assert-Equal `
        $holdResponse.data.status_reason `
        $holdReason `
        "on_hold reason persisted"

    Assert-Equal `
        $holdResponse.data.reviewed_by.actor_type `
        "human" `
        "Review action is attributed to human support actor"

    $customerHold =
        Invoke-RestMethod `
            -Method GET `
            -Uri "$BaseUrl/api/v1/product-requests/$requestAId" `
            -Headers $customerHeaders

    Assert-Equal `
        $customerHold.data.status `
        "on_hold" `
        "Customer sees on_hold status"

    Assert-Equal `
        $customerHold.data.status_reason `
        $holdReason `
        "Customer sees on_hold reason"

    Assert-True `
        ([bool]$customerHold.data.can_message) `
        "Customer may still message while request is on_hold"

    # ========================================================
    # 16. accepted
    # ========================================================

    Write-Step "16. Accept primary request"

    $acceptedResponse =
        Invoke-RestMethod `
            -Method PATCH `
            -Uri "$BaseUrl/api/v1/admin/product-requests/$requestAId/status" `
            -WebSession $adminSession `
            -Headers $adminHeaders `
            -ContentType "application/json" `
            -Body (@{
                status =
                    "accepted"
            } | ConvertTo-Json)

    Assert-Equal `
        $acceptedResponse.data.status `
        "accepted" `
        "on_hold request moved to accepted"

    $customerAccepted =
        Invoke-RestMethod `
            -Method GET `
            -Uri "$BaseUrl/api/v1/product-requests/$requestAId" `
            -Headers $customerHeaders

    Assert-Equal `
        $customerAccepted.data.status `
        "accepted" `
        "Customer sees accepted status"

    Assert-True `
        ([bool]$customerAccepted.data.can_message) `
        "Accepted request remains open for negotiation"

    # ========================================================
    # 17. Invalid commercial offer
    # ========================================================

    Write-Step "17. Verify sourcing offer validation"

    $offerExpiry =
        (Get-Date).
            ToUniversalTime().
            AddDays(7).
            ToString(
                "yyyy-MM-ddTHH:mm:ss'Z'"
            )

    Expect-ApiFailure `
        -Label "Offer quantity below MOQ" `
        -ExpectedStatus 400 `
        -ExpectedCode "INVALID_SOURCING_OFFER" `
        -Action {
            Invoke-RestMethod `
                -Method POST `
                -Uri "$BaseUrl/api/v1/admin/product-requests/$requestAId/offers" `
                -WebSession $adminSession `
                -Headers $adminHeaders `
                -ContentType "application/json" `
                -Body (@{
                    product_name =
                        $productAName

                    description =
                        "Invalid MOQ test offer"

                    attachments =
                        @()

                    offered_specifications = @{
                        color =
                            "black"
                    }

                    unit_price =
                        43000

                    shipping_price =
                        3000

                    currency =
                        "BDT"

                    quoted_quantity =
                        1

                    minimum_order_quantity =
                        2

                    expires_at =
                        $offerExpiry
                } | ConvertTo-Json -Depth 10)
        }

    # ========================================================
    # 18. Create first draft
    # ========================================================

    Write-Step "18. Create first draft sourcing offer"

    $firstUnitPrice = 43000
    $firstShippingPrice = 3000
    $firstQuantity = 2
    $firstMOQ = 1

    $firstOfferBody = @{
        product_name =
            $productAName

        description =
            "Initial supplier quotation for $runTag."

        attachments =
            @()

        offered_specifications = @{
            color =
                "black"

            condition =
                "new"

            warranty =
                "supplier warranty"

            revision =
                1
        }

        unit_price =
            $firstUnitPrice

        shipping_price =
            $firstShippingPrice

        currency =
            "BDT"

        quoted_quantity =
            $firstQuantity

        minimum_order_quantity =
            $firstMOQ

        expires_at =
            $offerExpiry
    } |
        ConvertTo-Json `
            -Depth 10

    $firstOffer =
        Invoke-RestMethod `
            -Method POST `
            -Uri "$BaseUrl/api/v1/admin/product-requests/$requestAId/offers" `
            -WebSession $adminSession `
            -Headers $adminHeaders `
            -ContentType "application/json" `
            -Body $firstOfferBody

    $offerAId =
        [string]$firstOffer.data.id

    Assert-True `
        (-not [string]::IsNullOrWhiteSpace(
            $offerAId
        )) `
        "First draft offer ID was returned"

    Assert-Equal `
        $firstOffer.data.status `
        "draft" `
        "New sourcing offer starts as draft"

    Assert-Equal `
        $firstOffer.data.created_by.actor_type `
        "human" `
        "Draft offer is attributed to human support actor"

    Assert-Equal `
        $firstOffer.data.currency `
        "BDT" `
        "Draft currency persisted"

    Assert-Equal `
        $firstOffer.data.quoted_quantity `
        $firstQuantity `
        "Draft quoted quantity persisted"

    Write-Info "First offer ID: $offerAId"

    # ========================================================
    # 19. Draft visibility
    # ========================================================

    Write-Step "19. Verify draft offer visibility"

    $adminDraftList =
        Invoke-RestMethod `
            -Method GET `
            -Uri "$BaseUrl/api/v1/admin/product-requests/$requestAId/offers" `
            -WebSession $adminSession

    $adminDraft =
        Get-ItemById `
            $adminDraftList.data `
            $offerAId

    Assert-True `
        ($null -ne $adminDraft) `
        "Admin offer list contains draft"

    $customerDraftList =
        Invoke-RestMethod `
            -Method GET `
            -Uri "$BaseUrl/api/v1/product-requests/$requestAId/offers" `
            -Headers $customerHeaders

    Assert-Equal `
        @($customerDraftList.data).Count `
        0 `
        "Customer cannot see draft offers"

    Expect-ApiFailure `
        -Label "Customer direct access to draft offer" `
        -ExpectedStatus 404 `
        -ExpectedCode "SOURCING_OFFER_NOT_FOUND" `
        -Action {
            Invoke-RestMethod `
                -Method GET `
                -Uri "$BaseUrl/api/v1/product-requests/$requestAId/offers/$offerAId" `
                -Headers $customerHeaders
        }

    # ========================================================
    # 20. Edit first draft
    # ========================================================

    Write-Step "20. Edit first draft sourcing offer"

    $firstEditedUnitPrice = 42500

    $firstOfferUpdate =
        Invoke-RestMethod `
            -Method PATCH `
            -Uri "$BaseUrl/api/v1/admin/product-requests/$requestAId/offers/$offerAId" `
            -WebSession $adminSession `
            -Headers $adminHeaders `
            -ContentType "application/json" `
            -Body (@{
                description =
                    "Updated initial quotation for $runTag."

                unit_price =
                    $firstEditedUnitPrice

                offered_specifications = @{
                    color =
                        "black"

                    condition =
                        "new"

                    warranty =
                        "supplier warranty"

                    revision =
                        1

                    note =
                        "updated before send"
                }
            } | ConvertTo-Json -Depth 10)

    Assert-Equal `
        $firstOfferUpdate.data.status `
        "draft" `
        "Editing offer keeps it in draft"

    Assert-Equal `
        $firstOfferUpdate.data.unit_price `
        $firstEditedUnitPrice `
        "Draft unit price was updated"

    Assert-Equal `
        $firstOfferUpdate.data.offered_specifications.note `
        "updated before send" `
        "Draft structured specifications were updated"

    # ========================================================
    # 21. Send first offer
    # ========================================================

    Write-Step "21. Send first sourcing offer"

    $sentFirstOffer =
        Invoke-RestMethod `
            -Method POST `
            -Uri "$BaseUrl/api/v1/admin/product-requests/$requestAId/offers/$offerAId/send" `
            -WebSession $adminSession `
            -Headers $adminHeaders

    Assert-Equal `
        $sentFirstOffer.data.status `
        "sent" `
        "First offer moved from draft to sent"

    Assert-True `
        (-not [string]::IsNullOrWhiteSpace(
            [string]$sentFirstOffer.data.sent_at
        )) `
        "Sent offer has sent_at timestamp"

    $customerNegotiating =
        Invoke-RestMethod `
            -Method GET `
            -Uri "$BaseUrl/api/v1/product-requests/$requestAId" `
            -Headers $customerHeaders

    Assert-Equal `
        $customerNegotiating.data.status `
        "negotiating" `
        "Sending first offer moves request to negotiating"

    $customerSentOffers =
        Invoke-RestMethod `
            -Method GET `
            -Uri "$BaseUrl/api/v1/product-requests/$requestAId/offers" `
            -Headers $customerHeaders

    Assert-Equal `
        @($customerSentOffers.data).Count `
        1 `
        "Customer sees sent offer"

    $customerFirstOffer =
        Get-ItemById `
            $customerSentOffers.data `
            $offerAId

    Assert-True `
        ($null -ne $customerFirstOffer) `
        "Customer offer list contains first offer"

    Assert-Equal `
        $customerFirstOffer.status `
        "sent" `
        "Customer sees first offer as sent"

    Assert-False `
        ($customerFirstOffer.PSObject.Properties.Name -contains "created_by") `
        "Customer offer does not expose internal support actor"

    Assert-Equal `
        $customerFirstOffer.offered_specifications.color `
        "black" `
        "Customer sees structured offered specifications"

    Expect-ApiFailure `
        -Label "Finalize offer before customer acceptance" `
        -ExpectedStatus 409 `
        -ExpectedCode "SOURCING_OFFER_NOT_ACTIONABLE" `
        -Action {
            Invoke-RestMethod `
                -Method POST `
                -Uri "$BaseUrl/api/v1/admin/product-requests/$requestAId/offers/$offerAId/finalize" `
                -WebSession $adminSession `
                -Headers $adminHeaders
        }

    # ========================================================
    # 22. Customer rejects first offer
    # ========================================================

    Write-Step "22. Customer rejects first sourcing offer"

    $rejectedFirstOffer =
        Invoke-RestMethod `
            -Method POST `
            -Uri "$BaseUrl/api/v1/product-requests/$requestAId/offers/$offerAId/reject" `
            -Headers $customerHeaders

    Assert-Equal `
        $rejectedFirstOffer.data.status `
        "customer_rejected" `
        "Customer rejection changes first offer to customer_rejected"

    Assert-True `
        (-not [string]::IsNullOrWhiteSpace(
            [string]$rejectedFirstOffer.data.customer_responded_at
        )) `
        "Rejected offer records customer response time"

    $afterRejectRequest =
        Invoke-RestMethod `
            -Method GET `
            -Uri "$BaseUrl/api/v1/product-requests/$requestAId" `
            -Headers $customerHeaders

    Assert-Equal `
        $afterRejectRequest.data.status `
        "negotiating" `
        "Rejected offer keeps request in negotiating"

    # ========================================================
    # 23. Rejected offer protections
    # ========================================================

    Write-Step "23. Verify rejected offer protections"

    Expect-ApiFailure `
        -Label "Accept already rejected offer" `
        -ExpectedStatus 409 `
        -ExpectedCode "SOURCING_OFFER_NOT_ACTIONABLE" `
        -Action {
            Invoke-RestMethod `
                -Method POST `
                -Uri "$BaseUrl/api/v1/product-requests/$requestAId/offers/$offerAId/accept" `
                -Headers $customerHeaders
        }

    Expect-ApiFailure `
        -Label "Edit already rejected offer" `
        -ExpectedStatus 409 `
        -ExpectedCode "SOURCING_OFFER_NOT_ACTIONABLE" `
        -Action {
            Invoke-RestMethod `
                -Method PATCH `
                -Uri "$BaseUrl/api/v1/admin/product-requests/$requestAId/offers/$offerAId" `
                -WebSession $adminSession `
                -Headers $adminHeaders `
                -ContentType "application/json" `
                -Body (@{
                    unit_price =
                        40000
                } | ConvertTo-Json)
        }

    # ========================================================
    # 24. Create revised offer
    # ========================================================

    Write-Step "24. Create revised sourcing offer"

    $revisedUnitPrice = 41000
    $revisedShippingPrice = 2000
    $revisedQuantity = 2
    $revisedMOQ = 1

    $revisedOfferBody = @{
        product_name =
            $productAName

        description =
            "Revised commercial quotation after customer feedback for $runTag."

        attachments =
            @()

        offered_specifications = @{
            color =
                "black"

            condition =
                "new"

            warranty =
                "supplier warranty"

            revision =
                2

            negotiated =
                $true
        }

        unit_price =
            $revisedUnitPrice

        shipping_price =
            $revisedShippingPrice

        currency =
            "BDT"

        quoted_quantity =
            $revisedQuantity

        minimum_order_quantity =
            $revisedMOQ

        expires_at =
            $offerExpiry
    } |
        ConvertTo-Json `
            -Depth 10

    $revisedOffer =
        Invoke-RestMethod `
            -Method POST `
            -Uri "$BaseUrl/api/v1/admin/product-requests/$requestAId/offers" `
            -WebSession $adminSession `
            -Headers $adminHeaders `
            -ContentType "application/json" `
            -Body $revisedOfferBody

    $offerBId =
        [string]$revisedOffer.data.id

    Assert-True `
        (-not [string]::IsNullOrWhiteSpace(
            $offerBId
        )) `
        "Revised offer ID was returned"

    Assert-True `
        ($offerBId -ne $offerAId) `
        "Revised offer is a new commercial revision"

    Assert-Equal `
        $revisedOffer.data.status `
        "draft" `
        "Revised offer starts as draft"

    Write-Info "Revised offer ID: $offerBId"

    # ========================================================
    # 25. Send revised offer
    # ========================================================

    Write-Step "25. Send revised sourcing offer"

    $sentRevisedOffer =
        Invoke-RestMethod `
            -Method POST `
            -Uri "$BaseUrl/api/v1/admin/product-requests/$requestAId/offers/$offerBId/send" `
            -WebSession $adminSession `
            -Headers $adminHeaders

    Assert-Equal `
        $sentRevisedOffer.data.status `
        "sent" `
        "Revised offer moved to sent"

    $customerRevisionList =
        Invoke-RestMethod `
            -Method GET `
            -Uri "$BaseUrl/api/v1/product-requests/$requestAId/offers" `
            -Headers $customerHeaders

    Assert-Equal `
        @($customerRevisionList.data).Count `
        2 `
        "Customer sees first response history and revised offer"

    $customerFirstHistory =
        Get-ItemById `
            $customerRevisionList.data `
            $offerAId

    $customerRevision =
        Get-ItemById `
            $customerRevisionList.data `
            $offerBId

    Assert-Equal `
        $customerFirstHistory.status `
        "customer_rejected" `
        "First rejected revision remains in history"

    Assert-Equal `
        $customerRevision.status `
        "sent" `
        "Customer sees revised offer as actionable sent offer"

    Assert-Equal `
        $customerRevision.unit_price `
        $revisedUnitPrice `
        "Customer sees revised unit price"

    Assert-Equal `
        $customerRevision.shipping_price `
        $revisedShippingPrice `
        "Customer sees revised shipping price"

    Assert-Equal `
        $customerRevision.offered_specifications.revision `
        2 `
        "Customer sees revision 2 specifications"

    # ========================================================
    # 26. Customer accepts revised offer
    # ========================================================

    Write-Step "26. Customer accepts revised sourcing offer"

    $acceptedOffer =
        Invoke-RestMethod `
            -Method POST `
            -Uri "$BaseUrl/api/v1/product-requests/$requestAId/offers/$offerBId/accept" `
            -Headers $customerHeaders

    Assert-Equal `
        $acceptedOffer.data.status `
        "customer_accepted" `
        "Customer acceptance changes revised offer to customer_accepted"

    Assert-True `
        (-not [string]::IsNullOrWhiteSpace(
            [string]$acceptedOffer.data.customer_responded_at
        )) `
        "Accepted offer records customer response time"

    $afterOfferAcceptance =
        Invoke-RestMethod `
            -Method GET `
            -Uri "$BaseUrl/api/v1/product-requests/$requestAId" `
            -Headers $customerHeaders

    Assert-Equal `
        $afterOfferAcceptance.data.status `
        "negotiating" `
        "Customer acceptance does not prematurely mark request agreed"

    # ========================================================
    # 27. Accepted offer protections
    # ========================================================

    Write-Step "27. Verify accepted offer protections"

    Expect-ApiFailure `
        -Label "Edit customer-accepted offer" `
        -ExpectedStatus 409 `
        -ExpectedCode "SOURCING_OFFER_NOT_ACTIONABLE" `
        -Action {
            Invoke-RestMethod `
                -Method PATCH `
                -Uri "$BaseUrl/api/v1/admin/product-requests/$requestAId/offers/$offerBId" `
                -WebSession $adminSession `
                -Headers $adminHeaders `
                -ContentType "application/json" `
                -Body (@{
                    unit_price =
                        1
                } | ConvertTo-Json)
        }

    Expect-ApiFailure `
        -Label "Resend customer-accepted offer" `
        -ExpectedStatus 409 `
        -ExpectedCode "SOURCING_OFFER_NOT_ACTIONABLE" `
        -Action {
            Invoke-RestMethod `
                -Method POST `
                -Uri "$BaseUrl/api/v1/admin/product-requests/$requestAId/offers/$offerBId/send" `
                -WebSession $adminSession `
                -Headers $adminHeaders
        }

    Expect-ApiFailure `
        -Label "Create new draft after customer acceptance" `
        -ExpectedStatus 409 `
        -ExpectedCode "SOURCING_OFFER_NOT_ACTIONABLE" `
        -Action {
            Invoke-RestMethod `
                -Method POST `
                -Uri "$BaseUrl/api/v1/admin/product-requests/$requestAId/offers" `
                -WebSession $adminSession `
                -Headers $adminHeaders `
                -ContentType "application/json" `
                -Body $revisedOfferBody
        }

    # ========================================================
    # 28. Finalize accepted offer
    # ========================================================

    Write-Step "28. Finalize customer-accepted sourcing offer"

    $expectedTotal =
        ([int64]$revisedUnitPrice *
            [int64]$revisedQuantity) +
        [int64]$revisedShippingPrice

    $finalizeResponse =
        Invoke-RestMethod `
            -Method POST `
            -Uri "$BaseUrl/api/v1/admin/product-requests/$requestAId/offers/$offerBId/finalize" `
            -WebSession $adminSession `
            -Headers $adminHeaders

    $confirmation =
        $finalizeResponse.data

    $confirmationId =
        [string]$confirmation.id

    Assert-True `
        (-not [string]::IsNullOrWhiteSpace(
            $confirmationId
        )) `
        "Finalization created sourcing confirmation"

    Assert-Equal `
        $confirmation.request_id `
        $requestAId `
        "Confirmation belongs to primary request"

    Assert-Equal `
        $confirmation.offer_id `
        $offerBId `
        "Confirmation references customer-accepted offer"

    Assert-Equal `
        $confirmation.status `
        "confirmed" `
        "New confirmation starts confirmed"

    Assert-Equal `
        $confirmation.quantity `
        $revisedQuantity `
        "Confirmation snapshots agreed quantity"

    Assert-Equal `
        $confirmation.minimum_order_quantity `
        $revisedMOQ `
        "Confirmation snapshots MOQ"

    Assert-Equal `
        $confirmation.unit_price_snapshot `
        $revisedUnitPrice `
        "Confirmation snapshots unit price"

    Assert-Equal `
        $confirmation.shipping_price_snapshot `
        $revisedShippingPrice `
        "Confirmation snapshots shipping price"

    Assert-Equal `
        $confirmation.currency `
        "BDT" `
        "Confirmation snapshots currency"

    Assert-Equal `
        ([int64]$confirmation.total_amount) `
        $expectedTotal `
        "Confirmation total equals unit price x quantity plus shipping"

    Assert-Equal `
        $confirmation.accepted_specifications.revision `
        2 `
        "Confirmation snapshots accepted specifications"

    Assert-Equal `
        $confirmation.finalized_by.actor_type `
        "human" `
        "Confirmation is attributed to human finalizer"

    Write-Info "Confirmation ID: $confirmationId"

    # ========================================================
    # 29. Verify agreed state and confirmation visibility
    # ========================================================

    Write-Step "29. Verify agreed request and confirmation visibility"

    $customerAgreed =
        Invoke-RestMethod `
            -Method GET `
            -Uri "$BaseUrl/api/v1/product-requests/$requestAId" `
            -Headers $customerHeaders

    Assert-Equal `
        $customerAgreed.data.status `
        "agreed" `
        "Finalization moves request to agreed"

    $adminFinalOffers =
        Invoke-RestMethod `
            -Method GET `
            -Uri "$BaseUrl/api/v1/admin/product-requests/$requestAId/offers" `
            -WebSession $adminSession

    $adminFirstFinal =
        Get-ItemById `
            $adminFinalOffers.data `
            $offerAId

    $adminSecondFinal =
        Get-ItemById `
            $adminFinalOffers.data `
            $offerBId

    Assert-Equal `
        $adminFirstFinal.status `
        "customer_rejected" `
        "Rejected first offer remains immutable history"

    Assert-Equal `
        $adminSecondFinal.status `
        "finalized" `
        "Accepted revised offer becomes finalized"

    Assert-True `
        (-not [string]::IsNullOrWhiteSpace(
            [string]$adminSecondFinal.finalized_at
        )) `
        "Finalized offer has finalized_at timestamp"

    $adminConfirmation =
        Invoke-RestMethod `
            -Method GET `
            -Uri "$BaseUrl/api/v1/admin/product-requests/$requestAId/confirmation" `
            -WebSession $adminSession

    Assert-Equal `
        $adminConfirmation.data.id `
        $confirmationId `
        "Admin can load finalized confirmation"

    Assert-Equal `
        $adminConfirmation.data.finalized_by.actor_type `
        "human" `
        "Admin confirmation exposes finalizer attribution"

    $customerConfirmation =
        Invoke-RestMethod `
            -Method GET `
            -Uri "$BaseUrl/api/v1/product-requests/$requestAId/confirmation" `
            -Headers $customerHeaders

    Assert-Equal `
        $customerConfirmation.data.id `
        $confirmationId `
        "Customer can load finalized confirmation"

    Assert-Equal `
        ([int64]$customerConfirmation.data.total_amount) `
        $expectedTotal `
        "Customer sees confirmed total amount"

    Assert-False `
        ($customerConfirmation.data.PSObject.Properties.Name -contains "finalized_by") `
        "Customer confirmation does not expose internal finalizer"

    Assert-False `
        ($customerConfirmation.data.PSObject.Properties.Name -contains "created_product_id") `
        "Confirmation has not created catalog product yet"

    Assert-False `
        ($customerConfirmation.data.PSObject.Properties.Name -contains "created_variant_id") `
        "Confirmation has not created catalog variant yet"

    Assert-False `
        ($customerConfirmation.data.PSObject.Properties.Name -contains "created_order_id") `
        "Confirmation has not created order yet"

    # ========================================================
    # 30. Finalization protections
    # ========================================================

    Write-Step "30. Verify finalization protections"

    Expect-ApiFailure `
        -Label "Finalize same offer twice" `
        -ExpectedStatus 409 `
        -ExpectedCode "SOURCING_OFFER_NOT_ACTIONABLE" `
        -Action {
            Invoke-RestMethod `
                -Method POST `
                -Uri "$BaseUrl/api/v1/admin/product-requests/$requestAId/offers/$offerBId/finalize" `
                -WebSession $adminSession `
                -Headers $adminHeaders
        }

    Expect-ApiFailure `
        -Label "Respond to finalized offer again" `
        -ExpectedStatus 409 `
        -ExpectedCode "SOURCING_OFFER_NOT_ACTIONABLE" `
        -Action {
            Invoke-RestMethod `
                -Method POST `
                -Uri "$BaseUrl/api/v1/product-requests/$requestAId/offers/$offerBId/reject" `
                -Headers $customerHeaders
        }

    # ========================================================
    # 31. Final customer offer history
    # ========================================================

    Write-Step "31. Verify final customer offer history"

    $customerFinalOffers =
        Invoke-RestMethod `
            -Method GET `
            -Uri "$BaseUrl/api/v1/product-requests/$requestAId/offers" `
            -Headers $customerHeaders

    Assert-Equal `
        @($customerFinalOffers.data).Count `
        2 `
        "Customer offer history contains exactly two revisions"

    $customerHistoricalOffer =
        Get-ItemById `
            $customerFinalOffers.data `
            $offerAId

    $customerFinalizedOffer =
        Get-ItemById `
            $customerFinalOffers.data `
            $offerBId

    Assert-Equal `
        $customerHistoricalOffer.status `
        "customer_rejected" `
        "Customer sees rejected first revision"

    Assert-Equal `
        $customerFinalizedOffer.status `
        "finalized" `
        "Customer sees finalized accepted revision"

    # ========================================================
    # 32. Create cancellation request
    # ========================================================

    Write-Step "32. Create second request for cancellation path"

    $productBName =
        "Cancellation Test Product $runTag"

    $createBBody = @{
        requested_product_name =
            $productBName

        description =
            "Cancellation lifecycle test for $runTag."

        requested_quantity =
            1

        customer_requirements = @{
            test_run =
                $runTag

            variant =
                "cancellation"
        }

        attachments =
            @()
    } |
        ConvertTo-Json `
            -Depth 10

    $createB =
        Invoke-RestMethod `
            -Method POST `
            -Uri "$BaseUrl/api/v1/product-requests" `
            -Headers $customerHeaders `
            -ContentType "application/json" `
            -Body $createBBody

    $requestBId =
        $createB.data.id

    Assert-Equal `
        $createB.data.status `
        "pending_review" `
        "Cancellation test request starts pending_review"

    Write-Info "Cancellation request ID: $requestBId"

    # ========================================================
    # 33. Cancellation reason
    # ========================================================

    Write-Step "33. Verify cancellation reason requirement"

    Expect-ApiFailure `
        -Label "cancelled without reason" `
        -ExpectedStatus 400 `
        -ExpectedCode "INVALID_PRODUCT_REQUEST" `
        -Action {
            Invoke-RestMethod `
                -Method PATCH `
                -Uri "$BaseUrl/api/v1/admin/product-requests/$requestBId/status" `
                -WebSession $adminSession `
                -Headers $adminHeaders `
                -ContentType "application/json" `
                -Body (@{
                    status =
                        "cancelled"
                } | ConvertTo-Json)
        }

    # ========================================================
    # 34. Cancel request
    # ========================================================

    Write-Step "34. Cancel second request"

    $cancelReason =
        "Unavailable from approved sourcing channels"

    $cancelResponse =
        Invoke-RestMethod `
            -Method PATCH `
            -Uri "$BaseUrl/api/v1/admin/product-requests/$requestBId/status" `
            -WebSession $adminSession `
            -Headers $adminHeaders `
            -ContentType "application/json" `
            -Body (@{
                status =
                    "cancelled"

                reason =
                    $cancelReason
            } | ConvertTo-Json)

    Assert-Equal `
        $cancelResponse.data.status `
        "cancelled" `
        "Second request moved to cancelled"

    Assert-Equal `
        $cancelResponse.data.status_reason `
        $cancelReason `
        "Cancellation reason persisted"

    $customerCancelled =
        Invoke-RestMethod `
            -Method GET `
            -Uri "$BaseUrl/api/v1/product-requests/$requestBId" `
            -Headers $customerHeaders

    Assert-Equal `
        $customerCancelled.data.status `
        "cancelled" `
        "Customer sees cancelled status"

    Assert-Equal `
        $customerCancelled.data.status_reason `
        $cancelReason `
        "Customer sees cancellation reason"

    Assert-False `
        ([bool]$customerCancelled.data.can_message) `
        "Cancelled request no longer accepts customer messages"

    # ========================================================
    # 35. Closed conversation
    # ========================================================

    Write-Step "35. Verify cancelled conversation is closed"

    Expect-ApiFailure `
        -Label "Customer message after cancellation" `
        -ExpectedStatus 409 `
        -ExpectedCode "PRODUCT_REQUEST_CONVERSATION_CLOSED" `
        -Action {
            Invoke-RestMethod `
                -Method POST `
                -Uri "$BaseUrl/api/v1/product-requests/$requestBId/messages" `
                -Headers $customerHeaders `
                -ContentType "application/json" `
                -Body (@{
                    message =
                        "This message must be rejected."

                    attachments =
                        @()
                } | ConvertTo-Json -Depth 10)
        }

    Expect-ApiFailure `
        -Label "Admin message after cancellation" `
        -ExpectedStatus 409 `
        -ExpectedCode "PRODUCT_REQUEST_CONVERSATION_CLOSED" `
        -Action {
            Invoke-RestMethod `
                -Method POST `
                -Uri "$BaseUrl/api/v1/admin/product-requests/$requestBId/messages" `
                -WebSession $adminSession `
                -Headers $adminHeaders `
                -ContentType "application/json" `
                -Body (@{
                    message =
                        "This Admin message must also be rejected."

                    visibility =
                        "customer"

                    attachments =
                        @()
                } | ConvertTo-Json -Depth 10)
        }

    # ========================================================
    # 36. Admin filters
    # ========================================================

    Write-Step "36. Verify Admin status filters"

    $agreedList =
        Invoke-RestMethod `
            -Method GET `
            -Uri "$BaseUrl/api/v1/admin/product-requests?status=agreed&q=$encodedRunTag" `
            -WebSession $adminSession

    $agreedIds =
        @(
            $agreedList.data |
                ForEach-Object {
                    $_.id
                }
        )

    Assert-True `
        ($agreedIds -contains $requestAId) `
        "agreed filter contains finalized primary request"

    Assert-False `
        ($agreedIds -contains $requestBId) `
        "agreed filter excludes cancelled request"

    $acceptedList =
        Invoke-RestMethod `
            -Method GET `
            -Uri "$BaseUrl/api/v1/admin/product-requests?status=accepted&q=$encodedRunTag" `
            -WebSession $adminSession

    $acceptedIds =
        @(
            $acceptedList.data |
                ForEach-Object {
                    $_.id
                }
        )

    Assert-False `
        ($acceptedIds -contains $requestAId) `
        "accepted filter excludes request after agreement"

    $cancelledList =
        Invoke-RestMethod `
            -Method GET `
            -Uri "$BaseUrl/api/v1/admin/product-requests?status=cancelled&q=$encodedRunTag" `
            -WebSession $adminSession

    $cancelledIds =
        @(
            $cancelledList.data |
                ForEach-Object {
                    $_.id
                }
        )

    Assert-True `
        ($cancelledIds -contains $requestBId) `
        "cancelled filter contains cancellation request"

    # ========================================================
    # 37. Customer list
    # ========================================================

    Write-Step "37. Verify customer list contains both requests"

    $customerFinalList =
        Invoke-RestMethod `
            -Method GET `
            -Uri "$BaseUrl/api/v1/product-requests?limit=100&offset=0" `
            -Headers $customerHeaders

    $customerRequestIds =
        @(
            $customerFinalList.data |
                ForEach-Object {
                    $_.id
                }
        )

    Assert-True `
        ($customerRequestIds -contains $requestAId) `
        "Customer list contains agreed request"

    Assert-True `
        ($customerRequestIds -contains $requestBId) `
        "Customer list contains cancelled request"

    # ========================================================
    # 38. PostgreSQL verification
    # ========================================================

        # ========================================================
    # 37A. Create real sourcing order
    # ========================================================

    Write-Step "37A. Place sourcing COD order"

    $sourcingOrderBody = @{
        customer_name =
            "Product Request E2E Customer"

        customer_phone =
            $customerPhone

        customer_email =
            ""

        shipping_address_line1 =
            "House 10, Road 5"

        shipping_address_line2 =
            "Sourcing E2E $runTag"

        shipping_city =
            "Dhaka"

        shipping_area =
            "Gulshan"

        shipping_postal_code =
            "1212"

        payment_method =
            "cod"
    } | ConvertTo-Json -Depth 10

    # --------------------------------------------------------
    # First call creates the sourcing order.
    # --------------------------------------------------------

    $firstOrderHttp =
        Invoke-WebRequest `
            -UseBasicParsing `
            -Method POST `
            -Uri "$BaseUrl/api/v1/product-requests/$requestAId/order" `
            -Headers $customerHeaders `
            -ContentType "application/json" `
            -Body $sourcingOrderBody

    Assert-Equal `
        ([int]$firstOrderHttp.StatusCode) `
        201 `
        "First sourcing order placement returns HTTP 201"

    $firstOrderPayload =
        $firstOrderHttp.Content |
            ConvertFrom-Json

    Assert-True `
        ([bool]$firstOrderPayload.data.created) `
        "First sourcing order placement reports created=true"

    $sourcingOrder =
        $firstOrderPayload.data.order

    $sourcingOrderId =
        [string]$sourcingOrder.id

    Assert-True `
        (-not [string]::IsNullOrWhiteSpace(
            $sourcingOrderId
        )) `
        "Sourcing order ID was returned"

    Assert-Equal `
        $sourcingOrder.order_type `
        "sourcing" `
        "Created order type is sourcing"

    Assert-Equal `
        $sourcingOrder.status `
        "awaiting_procurement" `
        "COD sourcing order starts awaiting_procurement"

    Assert-Equal `
        $sourcingOrder.payment_status `
        "cod_pending" `
        "COD sourcing order starts cod_pending"

    Assert-Equal `
        $sourcingOrder.payment_method `
        "cod" `
        "Sourcing order uses COD"

    # --------------------------------------------------------
    # Commercial truth must come from confirmation.
    # --------------------------------------------------------

    $expectedSubtotal =
        [int64]$confirmation.unit_price_snapshot *
        [int64]$confirmation.quantity

    Assert-Equal `
        ([int64]$sourcingOrder.subtotal_amount) `
        $expectedSubtotal `
        "Order subtotal comes from immutable confirmation"

    Assert-Equal `
        ([int64]$sourcingOrder.shipping_amount) `
        ([int64]$confirmation.shipping_price_snapshot) `
        "Order shipping comes from immutable confirmation"

    Assert-Equal `
        ([int64]$sourcingOrder.total_amount) `
        ([int64]$confirmation.total_amount) `
        "Order total comes from immutable confirmation"

    Assert-Equal `
        $sourcingOrder.currency `
        $confirmation.currency `
        "Order currency comes from immutable confirmation"

    # --------------------------------------------------------
    # Confirmation now owns product/variant/order.
    # --------------------------------------------------------

    $confirmationAfterOrder =
        Invoke-RestMethod `
            -Method GET `
            -Uri "$BaseUrl/api/v1/product-requests/$requestAId/confirmation" `
            -Headers $customerHeaders

    Assert-Equal `
        $confirmationAfterOrder.data.status `
        "order_created" `
        "Confirmation moves to order_created"

    Assert-Equal `
        $confirmationAfterOrder.data.created_order_id `
        $sourcingOrderId `
        "Confirmation links to sourcing order"

    Assert-True `
        (-not [string]::IsNullOrWhiteSpace(
            [string]$confirmationAfterOrder.data.created_product_id
        )) `
        "Confirmation links to internal product"

    Assert-True `
        (-not [string]::IsNullOrWhiteSpace(
            [string]$confirmationAfterOrder.data.created_variant_id
        )) `
        "Confirmation links to internal variant"

    # --------------------------------------------------------
    # Product Request becomes converted_to_order.
    # --------------------------------------------------------

    $requestAfterOrder =
        Invoke-RestMethod `
            -Method GET `
            -Uri "$BaseUrl/api/v1/product-requests/$requestAId" `
            -Headers $customerHeaders

    Assert-Equal `
        $requestAfterOrder.data.status `
        "converted_to_order" `
        "Request moves to converted_to_order"

    Assert-True `
        (-not [bool]$requestAfterOrder.data.can_message) `
        "Converted request conversation is closed"

    # --------------------------------------------------------
    # Normal order API must read cartless sourcing order.
    # --------------------------------------------------------

    $directOrder =
        Invoke-RestMethod `
            -Method GET `
            -Uri "$BaseUrl/api/v1/orders/$sourcingOrderId" `
            -Headers $customerHeaders

    Assert-Equal `
        $directOrder.data.id `
        $sourcingOrderId `
        "Customer can load sourcing order"

    Assert-Equal `
        $directOrder.data.order_type `
        "sourcing" `
        "Normal order API preserves sourcing type"

    # --------------------------------------------------------
    # Retry must be idempotent.
    # --------------------------------------------------------

    $retryOrderHttp =
        Invoke-WebRequest `
            -UseBasicParsing `
            -Method POST `
            -Uri "$BaseUrl/api/v1/product-requests/$requestAId/order" `
            -Headers $customerHeaders `
            -ContentType "application/json" `
            -Body $sourcingOrderBody

    Assert-Equal `
        ([int]$retryOrderHttp.StatusCode) `
        200 `
        "Repeated sourcing order placement returns HTTP 200"

    $retryOrderPayload =
        $retryOrderHttp.Content |
            ConvertFrom-Json

    Assert-True `
        (-not [bool]$retryOrderPayload.data.created) `
        "Repeated sourcing order placement reports created=false"

    Assert-Equal `
        $retryOrderPayload.data.order.id `
        $sourcingOrderId `
        "Repeated placement returns same order"

    Write-Info "Sourcing order ID: $sourcingOrderId"


    # ========================================================
    # 37B. Receive real stock and complete procurement
    # ========================================================

    Write-Step "37B. Complete sourcing procurement"

    $sourcingVariantId =
        [string]$confirmationAfterOrder.data.created_variant_id

    $sourcingQuantity =
        [int]$confirmation.quantity

    Assert-True `
        (-not [string]::IsNullOrWhiteSpace(
            $sourcingVariantId
        )) `
        "Sourcing variant is available for procurement"

    Assert-True `
        ($sourcingQuantity -gt 0) `
        "Sourcing procurement quantity is positive"


    # --------------------------------------------------------
    # Procurement may NOT complete before stock is received.
    # --------------------------------------------------------

    $procurementBody = @{
        status = "confirmed"
    } | ConvertTo-Json

    $preReceiptFailed =
        $false

    $preReceiptStatus =
        0

    try {
        Invoke-WebRequest `
            -UseBasicParsing `
            -Method PATCH `
            -Uri "$BaseUrl/api/v1/admin/orders/$sourcingOrderId/fulfillment" `
            -WebSession $adminSession `
            -Headers $adminHeaders `
            -ContentType "application/json" `
            -Body $procurementBody |
            Out-Null
    }
    catch {
        $preReceiptFailed =
            $true

        if ($null -ne
            $_.Exception.Response) {

            try {
                $preReceiptStatus =
                    [int]$_.Exception.Response.StatusCode
            }
            catch {
                $preReceiptStatus =
                    0
            }
        }
    }

    Assert-True `
        $preReceiptFailed `
        "Procurement confirmation fails before physical stock receipt"

    Assert-Equal `
        $preReceiptStatus `
        409 `
        "Missing sourced stock returns HTTP 409"

    $beforeReceiptOrder =
        Invoke-RestMethod `
            -Method GET `
            -Uri "$BaseUrl/api/v1/orders/$sourcingOrderId" `
            -Headers $customerHeaders

    Assert-Equal `
        $beforeReceiptOrder.data.status `
        "awaiting_procurement" `
        "Failed procurement attempt leaves order awaiting_procurement"


    # --------------------------------------------------------
    # Receive the sourced quantity through the existing
    # inventory adjustment workflow.
    # --------------------------------------------------------

    $receiptBody = @{
        quantity_delta =
            $sourcingQuantity

        reason =
            "sourcing_procurement_received"

        note =
            "Product Request E2E procurement receipt $runTag"
    } | ConvertTo-Json -Depth 10

    $receiptResponse =
        Invoke-RestMethod `
            -Method POST `
            -Uri "$BaseUrl/api/v1/admin/inventory/$sourcingVariantId/adjust" `
            -WebSession $adminSession `
            -Headers $adminHeaders `
            -ContentType "application/json" `
            -Body $receiptBody

    Assert-Equal `
        $receiptResponse.data.variant_id `
        $sourcingVariantId `
        "Inventory receipt updated sourcing variant"

    Assert-True `
        (
            [int]$receiptResponse.data.quantity_on_hand -ge
            $sourcingQuantity
        ) `
        "Received inventory covers immutable sourcing quantity"


    # --------------------------------------------------------
    # Now procurement completion must reserve + commit the
    # received stock and move awaiting_procurement -> confirmed.
    # --------------------------------------------------------

    $procurementResponse =
        Invoke-RestMethod `
            -Method PATCH `
            -Uri "$BaseUrl/api/v1/admin/orders/$sourcingOrderId/fulfillment" `
            -WebSession $adminSession `
            -Headers $adminHeaders `
            -ContentType "application/json" `
            -Body $procurementBody

    $confirmedSourcingOrder =
        $procurementResponse.data

    Assert-Equal `
        $confirmedSourcingOrder.id `
        $sourcingOrderId `
        "Procurement completion returns sourcing order"

    Assert-Equal `
        $confirmedSourcingOrder.order_type `
        "sourcing" `
        "Procurement completion preserves sourcing order type"

    Assert-Equal `
        $confirmedSourcingOrder.status `
        "confirmed" `
        "Procurement completion moves order to confirmed"

    Assert-Equal `
        $confirmedSourcingOrder.payment_status `
        "cod_pending" `
        "COD payment remains pending after procurement"

    Assert-True `
        (-not [string]::IsNullOrWhiteSpace(
            [string]$confirmedSourcingOrder.confirmed_at
        )) `
        "Procurement completion sets confirmed_at"

    $customerConfirmedOrder =
        Invoke-RestMethod `
            -Method GET `
            -Uri "$BaseUrl/api/v1/orders/$sourcingOrderId" `
            -Headers $customerHeaders

    Assert-Equal `
        $customerConfirmedOrder.data.status `
        "confirmed" `
        "Customer sees sourced order confirmed after procurement"


    # --------------------------------------------------------
    # After procurement, the sourced order joins the ordinary
    # fulfillment lifecycle.
    # --------------------------------------------------------

    $processingBody = @{
        status = "processing"
    } | ConvertTo-Json

    $processingResponse =
        Invoke-RestMethod `
            -Method PATCH `
            -Uri "$BaseUrl/api/v1/admin/orders/$sourcingOrderId/fulfillment" `
            -WebSession $adminSession `
            -Headers $adminHeaders `
            -ContentType "application/json" `
            -Body $processingBody

    Assert-Equal `
        $processingResponse.data.status `
        "processing" `
        "Confirmed sourcing order enters normal processing state"

    Assert-Equal `
        $processingResponse.data.payment_status `
        "cod_pending" `
        "COD remains pending while sourcing order processes"

    $customerProcessingOrder =
        Invoke-RestMethod `
            -Method GET `
            -Uri "$BaseUrl/api/v1/orders/$sourcingOrderId" `
            -Headers $customerHeaders

    Assert-Equal `
        $customerProcessingOrder.data.status `
        "processing" `
        "Customer sees sourcing order in normal processing lifecycle"
    if (-not $SkipDatabaseVerification) {
        Write-Step "38. Verify PostgreSQL sourcing state"

        $docker =
            Get-Command `
                docker `
                -ErrorAction SilentlyContinue

        if ($null -eq $docker) {
            throw "Docker CLI is required for Step 38 database verification"
        }

        $projectRoot =
            Split-Path `
                -Parent `
                $PSScriptRoot
        # ====================================================
        # Final PostgreSQL state AFTER:
        #
        # finalization
        # -> sourcing order creation
        # -> physical stock receipt
        # -> procurement confirmation
        # -> normal processing transition
        #
        # Return one JSON object instead of pipe-separated
        # values so product names/text cannot break parsing.
        # ====================================================

        $verificationSql = @"
WITH request_rows AS (
    SELECT
        psr.id::text AS id,
        psr.status,

        COALESCE(
            psr.status_reason,
            ''
        ) AS status_reason,

        c.case_type,
        c.status AS crm_status,

        COALESCE(
            c.requested_quantity,
            0
        ) AS requested_quantity,

        COALESCE(
            (
                SELECT q.code

                FROM support_case_assignments sca

                JOIN support_queues q
                    ON q.id = sca.queue_id

                WHERE
                    sca.case_id = c.id
                    AND sca.released_at IS NULL

                ORDER BY
                    sca.assigned_at DESC

                LIMIT 1
            ),
            ''
        ) AS queue_code,

        (
            SELECT COUNT(*)::int

            FROM crm_messages m

            WHERE m.case_id = c.id
        ) AS message_count

    FROM product_sourcing_requests psr

    JOIN crm_cases c
        ON c.id = psr.case_id

    WHERE psr.id IN (
        '$requestAId'::uuid,
        '$requestBId'::uuid
    )
),

offer_rows AS (
    SELECT
        id::text AS id,
        status,
        product_name,
        unit_price,
        shipping_price,
        currency,

        COALESCE(
            quoted_quantity,
            0
        ) AS quantity,

        COALESCE(
            minimum_order_quantity,
            0
        ) AS moq,

        created_by::text AS created_by,

        sent_at IS NOT NULL
            AS has_sent_at,

        customer_responded_at IS NOT NULL
            AS has_responded_at,

        finalized_at IS NOT NULL
            AS has_finalized_at

    FROM product_sourcing_offers

    WHERE request_id =
        '$requestAId'::uuid
),

confirmation_row AS (
    SELECT
        id::text AS id,
        request_id::text AS request_id,
        offer_id::text AS offer_id,
        quantity,
        minimum_order_quantity AS moq,
        accepted_product_name,
        unit_price_snapshot,
        shipping_price_snapshot,
        currency,
        total_amount,
        status,
        finalized_by::text AS finalized_by,

        COALESCE(
            created_product_id::text,
            ''
        ) AS created_product_id,

        COALESCE(
            created_variant_id::text,
            ''
        ) AS created_variant_id,

        COALESCE(
            created_order_id::text,
            ''
        ) AS created_order_id

    FROM product_sourcing_confirmations

    WHERE request_id =
        '$requestAId'::uuid
),

sourcing_row AS (
    SELECT
        o.id::text AS order_id,
        o.order_type,
        o.status AS order_status,
        o.payment_status,
        o.payment_method,

        COALESCE(
            o.checkout_id::text,
            ''
        ) AS checkout_id,

        COALESCE(
            o.cart_id::text,
            ''
        ) AS cart_id,

        COALESCE(
            o.customer_id::text,
            ''
        ) AS customer_id,

        o.currency,
        o.subtotal_amount,
        o.discount_amount,
        o.shipping_amount,
        o.total_amount,

        o.confirmed_at IS NOT NULL
            AS confirmed_at_set,

        o.processing_at IS NOT NULL
            AS processing_at_set,

        (
            SELECT COUNT(*)::int

            FROM order_items oi_count

            WHERE
                oi_count.order_id = o.id
        ) AS item_count,

        COALESCE(
            oi.variant_id::text,
            ''
        ) AS item_variant_id,

        COALESCE(
            oi.product_name,
            ''
        ) AS item_product_name,

        COALESCE(
            oi.quantity,
            0
        ) AS item_quantity,

        COALESCE(
            oi.minimum_order_quantity,
            0
        ) AS item_moq,

        COALESCE(
            oi.unit_price_amount,
            0
        ) AS item_unit_price,

        COALESCE(
            oi.line_total_amount,
            0
        ) AS item_line_total,

        COALESCE(
            oi.currency,
            ''
        ) AS item_currency,

        p.id::text AS product_id,
        p.name AS product_name,
        p.status AS product_status,

        p.published_at IS NULL
            AS product_is_unpublished,

        cat.name AS category_name,
        cat.is_active AS category_is_active,

        v.id::text AS variant_id,
        v.product_id::text AS variant_product_id,
        v.minimum_order_quantity AS variant_moq,
        v.price_amount AS variant_price,
        v.currency AS variant_currency,
        v.is_active AS variant_is_active,

        inv.variant_id IS NOT NULL
            AS inventory_exists,

        COALESCE(
            inv.quantity_on_hand,
            0
        ) AS quantity_on_hand,

        COALESCE(
            inv.quantity_reserved,
            0
        ) AS quantity_reserved,

        COALESCE(
            ir.quantity,
            0
        ) AS reservation_quantity,

        COALESCE(
            ir.status,
            ''
        ) AS reservation_status,

        EXISTS (
            SELECT 1

            FROM inventory_movements im

            WHERE
                im.variant_id = v.id
                AND im.movement_type = 'adjustment'
                AND im.quantity_on_hand_delta > 0
        ) AS has_positive_receipt_adjustment,

        (
            ir.created_at IS NOT NULL
            AND EXISTS (
                SELECT 1

                FROM inventory_movements im

                WHERE
                    im.variant_id = v.id
                    AND im.movement_type = 'adjustment'
                    AND im.quantity_on_hand_delta > 0
                    AND im.created_at <= ir.created_at
            )
        ) AS receipt_precedes_reservation,

        (
            SELECT COUNT(*)::int

            FROM order_events oe

            WHERE
                oe.order_id = o.id
                AND oe.event_type =
                    'order_placed'
        ) AS order_placed_event_count,

        (
            SELECT COUNT(*)::int

            FROM order_events oe

            WHERE
                oe.order_id = o.id
                AND oe.event_type =
                    'sourcing_procurement_completed'
        ) AS procurement_event_count,

        (
            SELECT COUNT(*)::int

            FROM order_events oe

            WHERE
                oe.order_id = o.id
                AND oe.event_type =
                    'order_processing'
        ) AS processing_event_count,

        (
            SELECT COUNT(*)::int

            FROM crm_case_events ce

            JOIN product_sourcing_requests psr_event
                ON psr_event.case_id =
                    ce.case_id

            WHERE
                psr_event.id =
                    '$requestAId'::uuid
                AND ce.event_type =
                    'product_sourcing_order_created'
        ) AS sourcing_order_crm_event_count

    FROM orders o

    JOIN product_sourcing_confirmations cf
        ON cf.created_order_id = o.id
        AND cf.id =
            '$confirmationId'::uuid

    JOIN products p
        ON p.id = cf.created_product_id

    JOIN categories cat
        ON cat.id = p.category_id

    JOIN product_variants v
        ON v.id = cf.created_variant_id

    LEFT JOIN LATERAL (
        SELECT oi_one.*

        FROM order_items oi_one

        WHERE
            oi_one.order_id = o.id

        ORDER BY
            oi_one.created_at,
            oi_one.id

        LIMIT 1
    ) oi
        ON true

    LEFT JOIN inventory inv
        ON inv.variant_id = v.id

    LEFT JOIN inventory_reservations ir
        ON ir.reference_type = 'order'
        AND ir.reference_id = o.id::text
        AND ir.variant_id = v.id

    WHERE o.id =
        '$sourcingOrderId'::uuid
)

SELECT
    jsonb_build_object(
        'primary_request',
        (
            SELECT to_jsonb(r)

            FROM request_rows r

            WHERE r.id =
                '$requestAId'
        ),

        'cancel_request',
        (
            SELECT to_jsonb(r)

            FROM request_rows r

            WHERE r.id =
                '$requestBId'
        ),

        'offers',
        COALESCE(
            (
                SELECT
                    jsonb_agg(
                        to_jsonb(o)
                        ORDER BY o.id
                    )

                FROM offer_rows o
            ),
            '[]'::jsonb
        ),

        'confirmation',
        (
            SELECT to_jsonb(c)

            FROM confirmation_row c
        ),

        'sourcing',
        (
            SELECT to_jsonb(s)

            FROM sourcing_row s
        )
    )::text;
"@

        $previousLocation =
            Get-Location

        try {
            Set-Location `
                -LiteralPath $projectRoot

            $rawDbJson =
                & $docker.Source `
                    compose `
                    exec `
                    -T `
                    postgres `
                    psql `
                    -X `
                    -U postgres `
                    -d commerce `
                    -t `
                    -A `
                    -v "ON_ERROR_STOP=1" `
                    -c $verificationSql

            if ($LASTEXITCODE -ne 0) {
                throw "Docker PostgreSQL Step 38 verification query failed"
            }
        }
        finally {
            Set-Location `
                -LiteralPath $previousLocation
        }
        $dbJsonText =
            (
                @($rawDbJson) -join ""
            ).Trim()

        if (
            [string]::IsNullOrWhiteSpace(
                $dbJsonText
            )
        ) {
            throw "PostgreSQL Step 38 returned no verification data"
        }

        try {
            $db =
                $dbJsonText |
                    ConvertFrom-Json
        }
        catch {
            throw "Unable to parse PostgreSQL Step 38 JSON result: $($_.Exception.Message)"
        }

        # ====================================================
        # Request / CRM truth
        # ====================================================

        Assert-True `
            ($null -ne $db.primary_request) `
            "Primary request exists in PostgreSQL"

        Assert-True `
            ($null -ne $db.cancel_request) `
            "Cancellation request exists in PostgreSQL"

        Assert-Equal `
            ([string]$db.primary_request.status) `
            "converted_to_order" `
            "Primary request DB status is converted_to_order"

        Assert-True `
            ([string]::IsNullOrWhiteSpace(
                [string]$db.primary_request.status_reason
            )) `
            "Converted request has no stale status reason"

        Assert-Equal `
            ([string]$db.primary_request.case_type) `
            "product_request" `
            "Primary request uses CRM case_type product_request"

        Assert-Equal `
            ([string]$db.primary_request.crm_status) `
            "closed" `
            "Sourcing order creation closes primary CRM case"

        Assert-Equal `
            ([int]$db.primary_request.requested_quantity) `
            2 `
            "Primary requested quantity remains stored in CRM case"

        Assert-Equal `
            ([string]$db.primary_request.queue_code) `
            "bulk_sales" `
            "Primary request remains linked to bulk_sales assignment"

        Assert-Equal `
            ([int]$db.primary_request.message_count) `
            4 `
            "Order conversion does not create fake CRM chat messages"

        # ====================================================
        # Cancellation request
        # ====================================================

        Assert-Equal `
            ([string]$db.cancel_request.status) `
            "cancelled" `
            "Cancellation request DB status is cancelled"

        Assert-Equal `
            ([string]$db.cancel_request.status_reason) `
            $cancelReason `
            "Cancellation reason matches database"

        Assert-Equal `
            ([string]$db.cancel_request.case_type) `
            "product_request" `
            "Cancellation request uses CRM product_request case"

        Assert-Equal `
            ([string]$db.cancel_request.crm_status) `
            "closed" `
            "Cancelled sourcing request closes CRM case"

        Assert-Equal `
            ([string]$db.cancel_request.queue_code) `
            "bulk_sales" `
            "Cancellation request remains linked to bulk_sales"

        Assert-Equal `
            ([int]$db.cancel_request.message_count) `
            1 `
            "Cancellation test case contains only initial customer message"

        # ====================================================
        # Immutable offer history
        # ====================================================

        $dbOffers =
            @(
                $db.offers
            )

        Assert-Equal `
            $dbOffers.Count `
            2 `
            "PostgreSQL contains exactly two commercial offer revisions"

        $dbRejectedOfferMatches =
            @(
                $dbOffers |
                    Where-Object {
                        [string]$_.id -eq
                            $offerAId
                    }
            )

        $dbFinalOfferMatches =
            @(
                $dbOffers |
                    Where-Object {
                        [string]$_.id -eq
                            $offerBId
                    }
            )

        Assert-Equal `
            $dbRejectedOfferMatches.Count `
            1 `
            "First rejected offer exists exactly once in PostgreSQL"

        Assert-Equal `
            $dbFinalOfferMatches.Count `
            1 `
            "Finalized revised offer exists exactly once in PostgreSQL"

        $dbRejectedOffer =
            $dbRejectedOfferMatches[0]

        $dbFinalOffer =
            $dbFinalOfferMatches[0]

        Assert-Equal `
            ([string]$dbRejectedOffer.status) `
            "customer_rejected" `
            "First offer DB status is customer_rejected"

        Assert-True `
            ([bool]$dbRejectedOffer.has_sent_at) `
            "First offer has sent_at"

        Assert-True `
            ([bool]$dbRejectedOffer.has_responded_at) `
            "First offer has customer response timestamp"

        Assert-False `
            ([bool]$dbRejectedOffer.has_finalized_at) `
            "Rejected first offer has no finalized_at"

        Assert-Equal `
            ([string]$dbFinalOffer.status) `
            "finalized" `
            "Revised offer DB status is finalized"

        Assert-Equal `
            ([int64]$dbFinalOffer.unit_price) `
            ([int64]$revisedUnitPrice) `
            "Finalized offer DB unit price matches revision"

        Assert-Equal `
            ([int64]$dbFinalOffer.shipping_price) `
            ([int64]$revisedShippingPrice) `
            "Finalized offer DB shipping price matches revision"

        Assert-Equal `
            ([string]$dbFinalOffer.currency) `
            "BDT" `
            "Finalized offer DB currency is BDT"

        Assert-Equal `
            ([int]$dbFinalOffer.quantity) `
            $revisedQuantity `
            "Finalized offer DB quantity matches agreement"

        Assert-Equal `
            ([int]$dbFinalOffer.moq) `
            $revisedMOQ `
            "Finalized offer DB MOQ matches agreement"

        Assert-True `
            (-not [string]::IsNullOrWhiteSpace(
                [string]$dbFinalOffer.created_by
            )) `
            "Finalized offer references support actor"

        Assert-True `
            ([bool]$dbFinalOffer.has_sent_at) `
            "Finalized offer retains sent_at"

        Assert-True `
            ([bool]$dbFinalOffer.has_responded_at) `
            "Finalized offer retains customer response timestamp"

        Assert-True `
            ([bool]$dbFinalOffer.has_finalized_at) `
            "Finalized offer has finalized_at"

        # ====================================================
        # Immutable confirmation -> order bridge
        # ====================================================

        Assert-True `
            ($null -ne $db.confirmation) `
            "Exactly one sourcing confirmation snapshot is available"

        $dbConfirmation =
            $db.confirmation

        Assert-Equal `
            ([string]$dbConfirmation.id) `
            $confirmationId `
            "Confirmation API ID matches database"

        Assert-Equal `
            ([string]$dbConfirmation.request_id) `
            $requestAId `
            "Confirmation DB request_id matches primary request"

        Assert-Equal `
            ([string]$dbConfirmation.offer_id) `
            $offerBId `
            "Confirmation DB offer_id matches finalized offer"

        Assert-Equal `
            ([int]$dbConfirmation.quantity) `
            $revisedQuantity `
            "Confirmation DB quantity remains immutable"

        Assert-Equal `
            ([int]$dbConfirmation.moq) `
            $revisedMOQ `
            "Confirmation DB MOQ remains immutable"

        Assert-Equal `
            ([string]$dbConfirmation.accepted_product_name) `
            ([string]$confirmation.accepted_product_name) `
            "Confirmation DB product name remains immutable"

        Assert-Equal `
            ([int64]$dbConfirmation.unit_price_snapshot) `
            ([int64]$revisedUnitPrice) `
            "Confirmation DB unit-price snapshot remains immutable"

        Assert-Equal `
            ([int64]$dbConfirmation.shipping_price_snapshot) `
            ([int64]$revisedShippingPrice) `
            "Confirmation DB shipping snapshot remains immutable"

        Assert-Equal `
            ([string]$dbConfirmation.currency) `
            "BDT" `
            "Confirmation DB currency remains immutable"

        Assert-Equal `
            ([int64]$dbConfirmation.total_amount) `
            ([int64]$expectedTotal) `
            "Confirmation DB total remains mathematically consistent"

        Assert-Equal `
            ([string]$dbConfirmation.status) `
            "order_created" `
            "Confirmation DB status is order_created"

        Assert-True `
            (-not [string]::IsNullOrWhiteSpace(
                [string]$dbConfirmation.finalized_by
            )) `
            "Confirmation DB retains human finalizer"

        Assert-True `
            (-not [string]::IsNullOrWhiteSpace(
                [string]$dbConfirmation.created_product_id
            )) `
            "Confirmation links to internal catalog product"

        Assert-True `
            (-not [string]::IsNullOrWhiteSpace(
                [string]$dbConfirmation.created_variant_id
            )) `
            "Confirmation links to internal catalog variant"

        Assert-Equal `
            ([string]$dbConfirmation.created_order_id) `
            $sourcingOrderId `
            "Confirmation links to exactly the sourcing order created by API"

        # ====================================================
        # Real cartless sourcing order
        # ====================================================

        Assert-True `
            ($null -ne $db.sourcing) `
            "Sourcing order exists in PostgreSQL"

        $dbSourcing =
            $db.sourcing

        Assert-Equal `
            ([string]$dbSourcing.order_id) `
            $sourcingOrderId `
            "Sourcing order DB ID matches API order"

        Assert-Equal `
            ([string]$dbSourcing.order_type) `
            "sourcing" `
            "Order DB type is sourcing"

        Assert-Equal `
            ([string]$dbSourcing.order_status) `
            "processing" `
            "Sourcing order DB status is processing"

        Assert-Equal `
            ([string]$dbSourcing.payment_status) `
            "cod_pending" `
            "Sourcing COD remains cod_pending while processing"

        Assert-Equal `
            ([string]$dbSourcing.payment_method) `
            "cod" `
            "Sourcing order DB payment method is COD"

        Assert-True `
            ([string]::IsNullOrWhiteSpace(
                [string]$dbSourcing.checkout_id
            )) `
            "Sourcing order has no checkout_id"

        Assert-True `
            ([string]::IsNullOrWhiteSpace(
                [string]$dbSourcing.cart_id
            )) `
            "Sourcing order has no cart_id"

        Assert-Equal `
            ([string]$dbSourcing.customer_id) `
            $customerId `
            "Cartless sourcing order belongs to authenticated customer"

        Assert-Equal `
            ([string]$dbSourcing.currency) `
            "BDT" `
            "Sourcing order currency matches confirmation"

        Assert-Equal `
            ([int64]$dbSourcing.subtotal_amount) `
            ([int64]$expectedSubtotal) `
            "Sourcing order DB subtotal comes from immutable confirmation"

        Assert-Equal `
            ([int64]$dbSourcing.discount_amount) `
            0 `
            "Sourcing order has no storefront promotion discount"

        Assert-Equal `
            ([int64]$dbSourcing.shipping_amount) `
            ([int64]$revisedShippingPrice) `
            "Sourcing order DB shipping comes from immutable confirmation"

        Assert-Equal `
            ([int64]$dbSourcing.total_amount) `
            ([int64]$expectedTotal) `
            "Sourcing order DB total comes from immutable confirmation"

        Assert-True `
            ([bool]$dbSourcing.confirmed_at_set) `
            "Procurement completion persisted confirmed_at"

        Assert-True `
            ([bool]$dbSourcing.processing_at_set) `
            "Normal processing transition persisted processing_at"

        # ====================================================
        # Order item -> real internal variant
        # ====================================================

        Assert-Equal `
            ([int]$dbSourcing.item_count) `
            1 `
            "Sourcing order contains exactly one immutable order item"

        Assert-Equal `
            ([string]$dbSourcing.item_variant_id) `
            ([string]$dbConfirmation.created_variant_id) `
            "Order item references confirmation-created variant"

        Assert-Equal `
            ([string]$dbSourcing.item_product_name) `
            ([string]$dbConfirmation.accepted_product_name) `
            "Order item product name matches immutable confirmation"

        Assert-Equal `
            ([int]$dbSourcing.item_quantity) `
            $revisedQuantity `
            "Order item quantity matches immutable confirmation"

        Assert-Equal `
            ([int]$dbSourcing.item_moq) `
            $revisedMOQ `
            "Order item MOQ matches immutable confirmation"

        Assert-Equal `
            ([int64]$dbSourcing.item_unit_price) `
            ([int64]$revisedUnitPrice) `
            "Order item unit price matches immutable confirmation"

        Assert-Equal `
            ([int64]$dbSourcing.item_line_total) `
            ([int64]$expectedSubtotal) `
            "Order item line total matches immutable confirmation"

        Assert-Equal `
            ([string]$dbSourcing.item_currency) `
            "BDT" `
            "Order item currency matches immutable confirmation"

        # ====================================================
        # Internal sourcing catalog must stay hidden
        # ====================================================

        Assert-Equal `
            ([string]$dbSourcing.product_id) `
            ([string]$dbConfirmation.created_product_id) `
            "Internal product matches confirmation linkage"

        Assert-Equal `
            ([string]$dbSourcing.product_name) `
            ([string]$dbConfirmation.accepted_product_name) `
            "Internal product name matches immutable confirmation"

        Assert-Equal `
            ([string]$dbSourcing.product_status) `
            "draft" `
            "Sourcing-only product remains draft"

        Assert-True `
            ([bool]$dbSourcing.product_is_unpublished) `
            "Sourcing-only product remains unpublished"

        Assert-Equal `
            ([string]$dbSourcing.category_name) `
            "Internal Sourcing Orders" `
            "Sourcing product uses internal catalog category"

        Assert-False `
            ([bool]$dbSourcing.category_is_active) `
            "Internal sourcing category is hidden from storefront"

        Assert-Equal `
            ([string]$dbSourcing.variant_id) `
            ([string]$dbConfirmation.created_variant_id) `
            "Internal variant matches confirmation linkage"

        Assert-Equal `
            ([string]$dbSourcing.variant_product_id) `
            ([string]$dbConfirmation.created_product_id) `
            "Internal variant belongs to sourced product"

        Assert-Equal `
            ([int]$dbSourcing.variant_moq) `
            $revisedMOQ `
            "Internal variant MOQ matches immutable confirmation"

        Assert-Equal `
            ([int64]$dbSourcing.variant_price) `
            ([int64]$revisedUnitPrice) `
            "Internal variant price matches immutable confirmation"

        Assert-Equal `
            ([string]$dbSourcing.variant_currency) `
            "BDT" `
            "Internal variant currency matches confirmation"

        Assert-False `
            ([bool]$dbSourcing.variant_is_active) `
            "Internal sourcing variant remains disabled for ordinary storefront ordering"

        # ====================================================
        # Physical receipt -> reserve -> commit
        # ====================================================

        Assert-True `
            ([bool]$dbSourcing.inventory_exists) `
            "Real inventory row exists for sourced variant"

        Assert-Equal `
            ([int]$dbSourcing.quantity_reserved) `
            0 `
            "Committed sourcing inventory leaves no active reserved quantity"

        Assert-Equal `
            ([int]$dbSourcing.reservation_quantity) `
            $revisedQuantity `
            "Procurement reservation uses exact immutable sourcing quantity"

        Assert-Equal `
            ([string]$dbSourcing.reservation_status) `
            "committed" `
            "Procurement reservation is committed"

        Assert-True `
            ([bool]$dbSourcing.has_positive_receipt_adjustment) `
            "Physical sourced stock receipt is recorded as positive adjustment"

        Assert-True `
            ([bool]$dbSourcing.receipt_precedes_reservation) `
            "Physical stock receipt precedes persisted sourcing reservation"

        # ====================================================
        # Idempotent domain events
        # ====================================================

        Assert-Equal `
            ([int]$dbSourcing.order_placed_event_count) `
            1 `
            "Sourcing order has exactly one order_placed event"

        Assert-Equal `
            ([int]$dbSourcing.procurement_event_count) `
            1 `
            "Sourcing procurement completed event exists exactly once"

        Assert-Equal `
            ([int]$dbSourcing.processing_event_count) `
            1 `
            "Normal processing transition event exists exactly once"

        Assert-Equal `
            ([int]$dbSourcing.sourcing_order_crm_event_count) `
            1 `
            "Product Request order-conversion CRM event exists exactly once"
    }
    else {
        Write-Step "38. PostgreSQL verification skipped"
    }

    # ========================================================
    # Success
    # ========================================================

    Write-Host ""
    Write-Host "============================================================" -ForegroundColor Green
    Write-Host " ALL PRODUCT REQUEST + SOURCING OFFER TESTS PASSED" -ForegroundColor Green
    Write-Host "============================================================" -ForegroundColor Green
    Write-Host ""

    Write-Host "Test run:             $runTag"
    Write-Host "Customer ID:          $customerId"
    Write-Host "Agreed request ID:    $requestAId"
    Write-Host "Cancelled request ID: $requestBId"
    Write-Host "Rejected offer ID:    $offerAId"
    Write-Host "Finalized offer ID:   $offerBId"
    Write-Host "Confirmation ID:      $confirmationId"
    Write-Host "Admin staff code:     $($adminStaff.staff_code)"
    Write-Host ""

    Write-Host "Validated:" -ForegroundColor White
    Write-Host "  customer authentication"
    Write-Host "  customer request create/list/detail"
    Write-Host "  customer CRM messaging"
    Write-Host "  Admin password + MFA session"
    Write-Host "  sourcing RBAC"
    Write-Host "  Admin review lifecycle"
    Write-Host "  internal-note visibility"
    Write-Host "  customer-visible support replies"
    Write-Host "  draft offer creation"
    Write-Host "  draft offer editing"
    Write-Host "  draft hidden from customer"
    Write-Host "  quantity / MOQ validation"
    Write-Host "  offer send -> negotiating"
    Write-Host "  customer rejection"
    Write-Host "  commercial offer revision"
    Write-Host "  customer acceptance"
    Write-Host "  accepted-offer immutability"
    Write-Host "  sourcing finalization"
    Write-Host "  agreed request state"
    Write-Host "  immutable confirmation snapshots"
    Write-Host "  confirmation total calculation"
    Write-Host "  customer confirmation privacy"
    Write-Host "  duplicate-finalization protection"
    Write-Host "  cancellation lifecycle"
    Write-Host "  closed-conversation enforcement"
    Write-Host "  Admin status filters"

    if (-not $SkipDatabaseVerification) {
        Write-Host "  PostgreSQL request/offer/confirmation verification"
        Write-Host "  no premature product/variant/order creation"
    }

    Write-Host ""
    Write-Host "Passwords, access tokens, refresh tokens, CSRF tokens, OTPs, and TOTPs were not printed." -ForegroundColor DarkGray
    Write-Host ""
}
catch {
    Write-Host ""
    Write-Host "============================================================" -ForegroundColor Red
    Write-Host " PRODUCT REQUEST + SOURCING OFFER TEST FAILED" -ForegroundColor Red
    Write-Host "============================================================" -ForegroundColor Red
    Write-Host ""

    Write-Host $_.Exception.Message -ForegroundColor Red
    Write-Host ""

    if (
        -not [string]::IsNullOrWhiteSpace(
            $requestAId
        )
    ) {
        Write-Host "Primary request created before failure: $requestAId"
    }

    if (
        -not [string]::IsNullOrWhiteSpace(
            $offerAId
        )
    ) {
        Write-Host "First offer created before failure:      $offerAId"
    }

    if (
        -not [string]::IsNullOrWhiteSpace(
            $offerBId
        )
    ) {
        Write-Host "Revised offer created before failure:    $offerBId"
    }

    if (
        -not [string]::IsNullOrWhiteSpace(
            $confirmationId
        )
    ) {
        Write-Host "Confirmation created before failure:     $confirmationId"
    }

    if (
        -not [string]::IsNullOrWhiteSpace(
            $requestBId
        )
    ) {
        Write-Host "Cancellation request before failure:     $requestBId"
    }

    Write-Host ""
    Write-Host "Check Docker API logs for the matching server-side error." -ForegroundColor Yellow

    exit 1
}
finally {
    $customerToken = $null
    $csrf = $null
    $totp = $null
    $customerOtp = $null
    $customerPassword = $null
    $adminPassword = $null
}