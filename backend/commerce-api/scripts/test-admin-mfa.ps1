
param(
    [string]$BaseUrl = "http://127.0.0.1:8081"
)

$ErrorActionPreference = "Stop"

function Read-SecretText {
    param(
        [string]$Prompt
    )

    $secure = Read-Host $Prompt -AsSecureString

    $ptr =
        [Runtime.InteropServices.Marshal]::SecureStringToBSTR(
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

function Get-ApiError {
    param(
        [System.Management.Automation.ErrorRecord]$ErrorRecord
    )

    $status = $null
    $body = $null

    if ($null -ne $ErrorRecord.Exception.Response) {
        try {
            $status =
                [int]$ErrorRecord.Exception.Response.StatusCode
        }
        catch {
        }
    }

    if (
        $null -ne $ErrorRecord.ErrorDetails -and
        -not [string]::IsNullOrWhiteSpace(
            $ErrorRecord.ErrorDetails.Message
        )
    ) {
        $body =
            $ErrorRecord.ErrorDetails.Message
    }

    return [PSCustomObject]@{
        Status = $status
        Body   = $body
    }
}

Write-Host ""
Write-Host "==============================================="
Write-Host " Commerce Admin MFA Test"
Write-Host "==============================================="
Write-Host ""
Write-Host "API: $BaseUrl"
Write-Host ""

$credential =
    Get-Credential `
        -Message "Commerce LOCAL Admin Login"

if ($null -eq $credential) {
    throw "Admin credential entry cancelled"
}

$identifier =
    $credential.UserName

$password =
    $credential.
        GetNetworkCredential().
        Password

$session =
    New-Object `
        Microsoft.PowerShell.Commands.WebRequestSession

try {
    Write-Host "Logging in with Admin password..."

    $login =
        Invoke-RestMethod `
            -Method POST `
            -Uri "$BaseUrl/api/v1/admin/auth/login" `
            -WebSession $session `
            -ContentType "application/json" `
            -Body (@{
                identifier = $identifier
                password   = $password
            } | ConvertTo-Json)

    $password = $null
    $credential = $null

    Write-Host ""
    Write-Host "[PASS] Admin password accepted"
    Write-Host "Challenge expires: $($login.data.challenge_expires_at)"
    Write-Host "MFA enrollment required: $($login.data.mfa_enrollment_required)"
    Write-Host ""

    if ($login.data.mfa_enrollment_required -eq $true) {
        throw "This LOCAL Admin account requires MFA enrollment."
    }

    $verified = $false

    for ($attempt = 1; $attempt -le 3; $attempt++) {
        Write-Host "MFA attempt $attempt of 3"
        Write-Host "Use the authenticator entry belonging to this LOCAL Admin account."
        Write-Host ""

        $code =
            Read-SecretText `
                "Enter current 6-digit Admin TOTP"

        if ($code -notmatch '^\d{6}$') {
            $code = $null

            Write-Host ""
            Write-Host "[FAIL] TOTP must contain exactly 6 digits."
            Write-Host ""

            continue
        }

        try {
            $mfa =
                Invoke-RestMethod `
                    -Method POST `
                    -Uri "$BaseUrl/api/v1/admin/auth/mfa/verify" `
                    -WebSession $session `
                    -ContentType "application/json" `
                    -Body (@{
                        challenge_token =
                            $login.data.challenge_token

                        method =
                            "totp"

                        code =
                            $code
                    } | ConvertTo-Json)

            $code = $null
            $verified = $true

            Write-Host ""
            Write-Host "[PASS] Admin MFA accepted"
            Write-Host ""

            break
        }
        catch {
            $code = $null

            $apiError =
                Get-ApiError `
                    $_

            Write-Host ""
            Write-Host "[FAIL] MFA verification failed"
            Write-Host "HTTP status: $($apiError.Status)"

            if (
                -not [string]::IsNullOrWhiteSpace(
                    $apiError.Body
                )
            ) {
                Write-Host "API response:"
                Write-Host $apiError.Body
            }

            Write-Host ""

            if (
                $apiError.Body -match
                'INVALID_ADMIN_CHALLENGE'
            ) {
                throw "Admin login challenge is no longer valid. Run the script again."
            }

            if (
                $apiError.Body -match
                'ADMIN_MFA_BLOCKED|ADMIN_CHALLENGE_LOCKED'
            ) {
                throw "Admin MFA/challenge is temporarily locked."
            }

            if ($attempt -lt 3) {
                Write-Host "Wait for a fresh authenticator code if necessary, then try again."
                Write-Host ""
            }
        }
    }

    if (-not $verified) {
        throw @"
MFA was rejected three times.

The most likely cause is that the authenticator entry does not match the
TOTP credential stored in the LOCAL PostgreSQL database.

Do not reset anything yet. Inspect the LOCAL MFA credential metadata next.
"@
    }

    $csrf =
        [string]$mfa.data.csrf_token

    if ([string]::IsNullOrWhiteSpace($csrf)) {
        throw "MFA succeeded but no CSRF token was returned."
    }

    $me =
        Invoke-RestMethod `
            -Method GET `
            -Uri "$BaseUrl/api/v1/admin/auth/me" `
            -WebSession $session

    Write-Host "[PASS] Admin session works"
    Write-Host ""
    Write-Host "Staff ID:   $($me.data.principal.staff.id)"
    Write-Host "Staff code: $($me.data.principal.staff.staff_code)"
    Write-Host "Name:       $($me.data.principal.staff.full_name)"
    Write-Host "Email:      $($me.data.principal.staff.email)"
    Write-Host ""
    Write-Host "Roles:"
    $me.data.principal.staff.roles |
        ForEach-Object {
            Write-Host "  $_"
        }

    Write-Host ""
    Write-Host "==============================================="
    Write-Host " ADMIN MFA TEST PASSED"
    Write-Host "==============================================="
    Write-Host ""
}
finally {
    $password = $null
    $code = $null
    $csrf = $null
}