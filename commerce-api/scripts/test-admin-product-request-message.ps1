param(
    [string]$BaseUrl = "http://127.0.0.1:8081",
    [string]$RequestID = "d418928d-909f-43f3-9918-b00d5cedd3a6"
)

$ErrorActionPreference = "Stop"

function Read-SecretText {
    param([string]$Prompt)

    $secure = Read-Host $Prompt -AsSecureString

    $ptr = [Runtime.InteropServices.Marshal]::SecureStringToBSTR(
        $secure
    )

    try {
        return [Runtime.InteropServices.Marshal]::PtrToStringBSTR(
            $ptr
        )
    }
    finally {
        [Runtime.InteropServices.Marshal]::ZeroFreeBSTR(
            $ptr
        )
    }
}

Write-Host ""
Write-Host "Admin Product Request Message Test"
Write-Host "Request: $RequestID"
Write-Host ""

$credential = Get-Credential `
    -Message "Commerce Admin Login"

if ($null -eq $credential) {
    throw "Admin login cancelled."
}

$identifier = $credential.UserName
$password = $credential.GetNetworkCredential().Password

$session = New-Object `
    Microsoft.PowerShell.Commands.WebRequestSession

try {
    Write-Host "1. Admin password login..."

    $login = Invoke-RestMethod `
        -Method POST `
        -Uri "$BaseUrl/api/v1/admin/auth/login" `
        -WebSession $session `
        -ContentType "application/json" `
        -Body (@{
            identifier = $identifier
            password   = $password
        } | ConvertTo-Json)

    Write-Host "[PASS] Password accepted"

    $totp = Read-SecretText `
        "Enter current 6-digit Admin TOTP"

    if ($totp -notmatch '^\d{6}$') {
        throw "TOTP must contain exactly 6 digits."
    }

    Write-Host "2. Verify MFA..."

    $mfa = Invoke-RestMethod `
        -Method POST `
        -Uri "$BaseUrl/api/v1/admin/auth/mfa/verify" `
        -WebSession $session `
        -ContentType "application/json" `
        -Body (@{
            challenge_token = $login.data.challenge_token
            method          = "totp"
            code            = $totp
        } | ConvertTo-Json)

    $totp = $null

    Write-Host "[PASS] MFA accepted"

    $csrf = [string]$mfa.data.csrf_token

    if ([string]::IsNullOrWhiteSpace($csrf)) {
        throw "No CSRF token returned."
    }

    Write-Host "3. Send internal note..."

    $body = @{
        message     = "Targeted internal note diagnostic"
        visibility  = "internal"
        attachments = @()
    } | ConvertTo-Json -Depth 10

    $response = Invoke-RestMethod `
        -Method POST `
        -Uri "$BaseUrl/api/v1/admin/product-requests/$RequestID/messages" `
        -WebSession $session `
        -Headers @{
            "X-CSRF-Token" = $csrf
        } `
        -ContentType "application/json" `
        -Body $body

    Write-Host ""
    Write-Host "[PASS] Internal note succeeded" -ForegroundColor Green
    Write-Host ""

    $response | ConvertTo-Json -Depth 10
}
catch {
    Write-Host ""
    Write-Host "[FAIL] Request failed" -ForegroundColor Red
    Write-Host $_.Exception.Message -ForegroundColor Red

    if (
        $null -ne $_.ErrorDetails -and
        -not [string]::IsNullOrWhiteSpace($_.ErrorDetails.Message)
    ) {
        Write-Host ""
        Write-Host "API body:"
        Write-Host $_.ErrorDetails.Message
    }

    Write-Host ""
    Write-Host "NOW CHECK THE AIR TERMINAL." -ForegroundColor Yellow
    Write-Host "Look for:"
    Write-Host "productrequest admin internal error:"
    Write-Host ""

    exit 1
}
finally {
    $password = $null
    $totp = $null
    $csrf = $null
}