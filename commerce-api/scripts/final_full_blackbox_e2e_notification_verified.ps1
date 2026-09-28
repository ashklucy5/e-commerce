param(
    [string]$BaseUrl = "http://127.0.0.1:8081/api/v1",
    [string]$AdminIdentifier = "",
    [string]$SupportIdentifier = "EMP-000001"
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

# ============================================================
# Final full cross-module Docker black-box E2E
#
# Preconditions:
#   - docker_backend_validation.ps1 already PASSED
#   - Docker Compose stack is still running
#   - admin MFA is enrolled
#   - a support staff account exists (EMP-000001 by default)
#
# This test intentionally creates uniquely-stamped audit/test data and
# does not destructively delete immutable commerce/finance history.
# ============================================================

function Write-Step {
    param([string]$Name)
    Write-Host ""
    Write-Host "============================================================"
    Write-Host "==> $Name"
    Write-Host "============================================================"
}

function Assert-True {
    param([bool]$Condition, [string]$Message)
    if (-not $Condition) { throw $Message }
}

function Assert-Equal {
    param($Actual, $Expected, [string]$Message)
    if ($Actual -ne $Expected) {
        throw "$Message | expected=[$Expected] actual=[$Actual]"
    }
}

function Get-OptionalPropertyValue {
    param(
        $Object,
        [string]$Name
    )

    if ($null -eq $Object) { return $null }

    $property = $Object.PSObject.Properties[$Name]
    if ($null -eq $property) { return $null }

    return $property.Value
}

function Read-PlainSecret {
    param([string]$Prompt)

    $secure = Read-Host $Prompt -AsSecureString
    $pointer = [Runtime.InteropServices.Marshal]::SecureStringToBSTR($secure)

    try {
        return [Runtime.InteropServices.Marshal]::PtrToStringBSTR($pointer)
    }
    finally {
        [Runtime.InteropServices.Marshal]::ZeroFreeBSTR($pointer)
    }
}

function Get-EnvOrPromptText {
    param(
        [string]$EnvironmentName,
        [string]$Prompt,
        [string]$Default = ""
    )

    $value = [Environment]::GetEnvironmentVariable($EnvironmentName, "Process")
    if (-not [string]::IsNullOrWhiteSpace($value)) { return $value }

    $inputValue = Read-Host $Prompt
    if ([string]::IsNullOrWhiteSpace($inputValue)) { return $Default }
    return $inputValue.Trim()
}

function Get-EnvOrPromptSecret {
    param(
        [string]$EnvironmentName,
        [string]$Prompt
    )

    $value = [Environment]::GetEnvironmentVariable($EnvironmentName, "Process")
    if (-not [string]::IsNullOrWhiteSpace($value)) { return $value }
    return Read-PlainSecret $Prompt
}

function Invoke-DbLines {
    param([string]$Sql)

    $output = @(
        & docker compose exec -T postgres psql `
            -X `
            -U postgres `
            -d commerce `
            -t `
            -A `
            -F "|" `
            -v "ON_ERROR_STOP=1" `
            -c $Sql 2>&1
    )

    if ($LASTEXITCODE -ne 0) {
        throw "PostgreSQL query failed:`n$($output -join "`n")"
    }

    return @(
        $output |
        Where-Object { -not [string]::IsNullOrWhiteSpace("$_") } |
        ForEach-Object { "$($_)".Trim() }
    )
}

function Invoke-DbScalar {
    param([string]$Sql)

    $lines = @(Invoke-DbLines $Sql)
    if ($lines.Count -eq 0) { return "" }
    return "$($lines[-1])".Trim()
}

function Invoke-Admin {
    param(
        [string]$Method,
        [string]$Path,
        $Body = $null,
        [hashtable]$ExtraHeaders = $null
    )

    $headers = @{}
    foreach ($key in $script:AdminHeaders.Keys) {
        $headers[$key] = $script:AdminHeaders[$key]
    }

    if ($null -ne $ExtraHeaders) {
        foreach ($key in $ExtraHeaders.Keys) {
            $headers[$key] = $ExtraHeaders[$key]
        }
    }

    $parameters = @{
        Method = $Method
        Uri = "$BaseUrl$Path"
        WebSession = $script:AdminSession
        Headers = $headers
    }

    if ($null -ne $Body) {
        $parameters.ContentType = "application/json"
        $parameters.Body = $Body | ConvertTo-Json -Depth 30
    }

    return Invoke-RestMethod @parameters
}

function Get-ProfitLoss {
    param([string]$Date)

    return Invoke-RestMethod `
        -Method Get `
        -Uri "$BaseUrl/admin/finance/profit-loss?from=$Date&to=$Date&granularity=day&currency=BDT" `
        -WebSession $script:AdminSession
}

function Get-CustomerTracking {
    param([string]$OrderID, [hashtable]$Headers)

    return Invoke-RestMethod `
        -Method Get `
        -Uri "$BaseUrl/orders/$OrderID/tracking" `
        -Headers $Headers
}

function Assert-TrackingStage {
    param($Tracking, [string]$Expected, [string]$Message)
    Assert-Equal $Tracking.data.current_stage $Expected $Message
}

function Wait-OutboxTerminal {
    param([string]$OrderID, [string]$CaseID)

    $remaining = -1
    for ($attempt = 0; $attempt -lt 30; $attempt++) {
        $remaining = [int](Invoke-DbScalar @"
SELECT count(*)
FROM notification_outbox
WHERE
    (order_id = '$OrderID'::uuid OR case_id = '$CaseID'::uuid)
    AND status IN ('pending', 'processing');
"@)

        if ($remaining -eq 0) { return }
        Start-Sleep -Seconds 2
    }

    throw "Notification outbox did not reach terminal state; remaining=$remaining"
}

function Assert-OutboxEventCount {
    param(
        [string]$Where,
        [int]$Expected,
        [string]$Message
    )

    $actual = [int](Invoke-DbScalar "SELECT count(*) FROM notification_outbox WHERE $Where;")
    Assert-Equal $actual $Expected $Message
}

function Get-HttpFailureStatus {
    param([scriptblock]$Action)

    try {
        & $Action | Out-Null
        return 0
    }
    catch {
        if ($null -ne $_.Exception.Response) {
            return [int]$_.Exception.Response.StatusCode
        }
        throw
    }
}

$root = Resolve-Path (Join-Path $PSScriptRoot "..")
Push-Location $root

try {
    Write-Host ""
    Write-Host "============================================================"
    Write-Host " FINAL FULL CROSS-MODULE BLACK-BOX E2E"
    Write-Host "============================================================"
    Write-Host "Repository: $root"
    Write-Host "API:        $BaseUrl"
    Write-Host ""

    # ========================================================
    # 1. Docker/runtime preflight
    # ========================================================

    Write-Step "Docker/runtime preflight"

    $runningServices = @(& docker compose ps --status running --services)
    if ($LASTEXITCODE -ne 0) { throw "docker compose ps failed" }

    foreach ($service in @("postgres", "redis", "api", "scheduler", "worker")) {
        Assert-True ($runningServices -contains $service) "Docker service is not running: $service"
        Write-Host "PASS: $service running"
    }

    $ready = Invoke-RestMethod -Method Get -Uri ($BaseUrl.Replace("/api/v1", "") + "/health/ready")
    Assert-Equal $ready.status "ready" "API readiness status mismatch"
    Assert-Equal $ready.dependencies.postgres "ok" "API PostgreSQL readiness mismatch"
    Assert-Equal $ready.dependencies.redis "ok" "API Redis readiness mismatch"

    $dbPing = Invoke-DbScalar "SELECT 'ok';"
    Assert-Equal $dbPing "ok" "PostgreSQL unavailable"

    $redisPing = @(& docker compose exec -T redis redis-cli PING)
    if ($LASTEXITCODE -ne 0) { throw "Redis PING failed" }
    Assert-Equal "$($redisPing[-1])" "PONG" "Redis PING mismatch"

    Write-Host "PASS: API/PostgreSQL/Redis ready"

    # ========================================================
    # 2. Admin MFA login
    # ========================================================

    Write-Step "Admin MFA login"

    if ([string]::IsNullOrWhiteSpace($AdminIdentifier)) {
        $AdminIdentifier = Get-EnvOrPromptText "E2E_ADMIN_IDENTIFIER" "Admin email / identifier"
    }

    $adminPassword = Get-EnvOrPromptSecret "E2E_ADMIN_PASSWORD" "Admin password"

    $script:AdminSession = New-Object Microsoft.PowerShell.Commands.WebRequestSession

    $adminLogin = Invoke-RestMethod `
        -Method Post `
        -Uri "$BaseUrl/admin/auth/login" `
        -WebSession $script:AdminSession `
        -ContentType "application/json" `
        -Body (@{
            identifier = $AdminIdentifier
            password = $adminPassword
        } | ConvertTo-Json)

    $adminPassword = $null

    Assert-True (-not [string]::IsNullOrWhiteSpace($adminLogin.data.challenge_token)) "Admin MFA challenge missing"
    Assert-True ($adminLogin.data.mfa_enrollment_required -ne $true) "Admin MFA enrollment is still required"

    $totp = [Environment]::GetEnvironmentVariable("E2E_ADMIN_TOTP", "Process")
    if ([string]::IsNullOrWhiteSpace($totp)) {
        $totp = Read-Host "Current admin TOTP code"
    }

    $mfa = Invoke-RestMethod `
        -Method Post `
        -Uri "$BaseUrl/admin/auth/mfa/verify" `
        -WebSession $script:AdminSession `
        -ContentType "application/json" `
        -Body (@{
            challenge_token = $adminLogin.data.challenge_token
            method = "totp"
            code = $totp
        } | ConvertTo-Json)

    $totp = $null

    Assert-True (-not [string]::IsNullOrWhiteSpace($mfa.data.csrf_token)) "Admin CSRF token missing"
    $script:AdminHeaders = @{ "X-CSRF-Token" = $mfa.data.csrf_token }

    $adminMe = Invoke-RestMethod -Method Get -Uri "$BaseUrl/admin/auth/me" -WebSession $script:AdminSession
    $adminStaffID = $adminMe.data.principal.staff.id
    Assert-True (-not [string]::IsNullOrWhiteSpace($adminStaffID)) "Admin staff ID missing"

    Write-Host "PASS: admin authenticated"

    # ========================================================
    # 3. Finance baseline
    # ========================================================

    Write-Step "Finance baseline"

    $dhakaDate = [DateTimeOffset]::UtcNow.ToOffset([TimeSpan]::FromHours(6)).ToString("yyyy-MM-dd")
    $baselineResponse = Get-ProfitLoss $dhakaDate
    $baseline = $baselineResponse.data

    Write-Host "Date: $dhakaDate"
    Write-Host "Baseline collected orders: $($baseline.collected_orders)"
    Write-Host "Baseline expenses:         $($baseline.recorded_expenses_amount)"
    Write-Host "PASS: baseline captured"

    # ========================================================
    # 4. Create isolated active catalog product with known COGS
    # ========================================================

    Write-Step "Catalog product creation + public discovery"

    $stamp = [DateTimeOffset]::UtcNow.ToUnixTimeMilliseconds()
    $category = $null

    for ($attempt = 0; $attempt -lt 10 -and $null -eq $category; $attempt++) {
        $suffix = "{0:X3}" -f (Get-Random -Minimum 0 -Maximum 4096)
        $prefix = "E2E-$suffix"
        $categorySlug = "e2e-final-$stamp-$attempt"

        try {
            $category = Invoke-Admin `
                -Method Post `
                -Path "/admin/categories" `
                -Body @{
                    name = "E2E Final $stamp $attempt"
                    parent_id = $null
                    slug = $categorySlug
                    description = "Final full cross-module E2E category"
                    sort_order = 9999
                    is_active = $true
                    product_code_prefix = $prefix
                }
        }
        catch {
            if ($attempt -eq 9) { throw }
        }
    }

    Assert-True ($null -ne $category) "Unable to create isolated E2E category"
    $categoryID = $category.data.id
    Assert-True (-not [string]::IsNullOrWhiteSpace($categoryID)) "Category ID missing"

    $moq = 2
    $initialStock = 25
    $unitPrice = [int64]5000
    $unitCost = [int64]3000
    $expectedLineCost = $unitCost * $moq

    $product = Invoke-Admin `
        -Method Post `
        -Path "/admin/products" `
        -Body @{
            product_code = ""
            name = "Final E2E Product $stamp"
            category_id = $categoryID
            category_path = ""
            slug = ""
            brand = "E2E"
            short_description = "Final backend E2E product"
            description = "Isolated product used by final cross-module backend validation."
            status = "active"
            is_featured = $false
            variants = @(
                @{
                    sku = ""
                    color_name = "Black"
                    color_hex = "#111111"
                    size = "STD"
                    minimum_order_quantity = $moq
                    order_increment = 1
                    price_amount = $unitPrice
                    compare_at_price_amount = $null
                    cost_amount = $unitCost
                    currency = "BDT"
                    barcode = ""
                    weight_grams = 500
                    is_active = $true
                    stock = $initialStock
                    reorder_level = 3
                    price_tiers = @()
                }
            )
        }

    $productID = $product.data.id
    $productSlug = $product.data.slug
    $variantID = @($product.data.variants)[0].id

    Assert-True (-not [string]::IsNullOrWhiteSpace($productID)) "Product ID missing"
    Assert-True (-not [string]::IsNullOrWhiteSpace($productSlug)) "Product slug missing"
    Assert-True (-not [string]::IsNullOrWhiteSpace($variantID)) "Variant ID missing"

    $publicProduct = Invoke-RestMethod -Method Get -Uri "$BaseUrl/products/$productSlug"
    Assert-Equal $publicProduct.data.id $productID "Public product ID mismatch"

    $publicVariant = @($publicProduct.data.variants | Where-Object { $_.id -eq $variantID })[0]
    Assert-True ($null -ne $publicVariant) "Created variant not visible on public storefront"
    Assert-True ($publicVariant.in_stock -eq $true) "Created variant is not in stock"
    Assert-True ([int64]$publicVariant.available_quantity -ge $moq) "Created variant available quantity is below MOQ"

    Write-Host "Product: $productID"
    Write-Host "Variant: $variantID"
    Write-Host "Slug:    $productSlug"
    Write-Host "PASS: catalog write -> public storefront"

    # ========================================================
    # 5. Customer register + login + preferences
    # ========================================================

    Write-Step "Customer registration/login"

    $phoneTail = Get-Random -Minimum 10000000 -Maximum 99999999
    $customerPhone = "+88017$phoneTail"
    $customerEmail = "final.e2e.$stamp@example.test"
    $customerPassword = "Final-E2E-2026!"

    $registration = Invoke-RestMethod `
        -Method Post `
        -Uri "$BaseUrl/auth/register" `
        -ContentType "application/json" `
        -Body (@{
            phone = $customerPhone
            email = $customerEmail
            password = $customerPassword
            full_name = "Final E2E Customer"
        } | ConvertTo-Json)

    $customerID = $registration.data.customer.id
    Assert-True (-not [string]::IsNullOrWhiteSpace($customerID)) "Customer ID missing"

    $login = Invoke-RestMethod `
        -Method Post `
        -Uri "$BaseUrl/auth/login" `
        -ContentType "application/json" `
        -Body (@{
            phone = $customerPhone
            password = $customerPassword
        } | ConvertTo-Json)

    $customerPassword = $null
    $customerToken = $login.data.tokens.access_token
    Assert-True (-not [string]::IsNullOrWhiteSpace($customerToken)) "Customer access token missing"

    $customerHeaders = @{ Authorization = "Bearer $customerToken" }

    $me = Invoke-RestMethod -Method Get -Uri "$BaseUrl/customers/me" -Headers $customerHeaders
    Assert-Equal $me.data.id $customerID "Authenticated customer mismatch"

    $prefs = Invoke-RestMethod `
        -Method Patch `
        -Uri "$BaseUrl/customers/me/notification-preferences" `
        -Headers $customerHeaders `
        -ContentType "application/json" `
        -Body (@{
            order_updates_sms = $true
            order_updates_email = $false
            delivery_updates_sms = $true
            delivery_updates_email = $false
            support_updates_sms = $true
            support_updates_email = $false
        } | ConvertTo-Json)

    Assert-True ($prefs.data.order_updates_sms -eq $true) "Order SMS preference not enabled"
    Assert-True ($prefs.data.order_updates_email -eq $false) "Order email preference not disabled"

    Write-Host "Customer: $customerID"
    Write-Host "PASS: register/login/preferences"

    # ========================================================
    # 6. Cart -> checkout -> COD order
    # ========================================================

    Write-Step "Cart -> checkout -> COD order"

    $cart = Invoke-RestMethod -Method Post -Uri "$BaseUrl/carts" -Headers $customerHeaders
    $cartKey = $cart.data.cart_key
    Assert-True (-not [string]::IsNullOrWhiteSpace($cartKey)) "Cart key missing"

    Invoke-RestMethod `
        -Method Post `
        -Uri "$BaseUrl/carts/$cartKey/items" `
        -Headers $customerHeaders `
        -ContentType "application/json" `
        -Body (@{
            variant_id = $variantID
            quantity = $moq
        } | ConvertTo-Json) | Out-Null

    $checkout = Invoke-RestMethod `
        -Method Post `
        -Uri "$BaseUrl/checkouts" `
        -Headers $customerHeaders `
        -ContentType "application/json" `
        -Body (@{ cart_key = $cartKey } | ConvertTo-Json)

    $checkoutKey = $checkout.data.checkout_key
    Assert-True (-not [string]::IsNullOrWhiteSpace($checkoutKey)) "Checkout key missing"

    $deliveryMethods = Invoke-RestMethod `
        -Method Get `
        -Uri "$BaseUrl/checkouts/$checkoutKey/delivery-methods" `
        -Headers $customerHeaders

    $deliveryCode = @($deliveryMethods.data)[0].code
    Assert-True (-not [string]::IsNullOrWhiteSpace($deliveryCode)) "No delivery method available"

    $paymentMethods = Invoke-RestMethod `
        -Method Get `
        -Uri "$BaseUrl/checkout-options/payment-methods?checkout_key=$checkoutKey" `
        -Headers $customerHeaders

    $paymentCodes = @($paymentMethods.data | ForEach-Object { $_.code })
    Assert-True ($paymentCodes -contains "cod") "COD payment option is unavailable"

    Invoke-RestMethod `
        -Method Patch `
        -Uri "$BaseUrl/checkouts/$checkoutKey" `
        -Headers $customerHeaders `
        -ContentType "application/json" `
        -Body (@{
            customer_name = "Final E2E Customer"
            customer_phone = $customerPhone
            customer_email = $customerEmail
            shipping_address_line1 = "Final E2E Address"
            shipping_address_line2 = ""
            shipping_city = "Dhaka"
            shipping_area = "Dhanmondi"
            shipping_postal_code = "1209"
            delivery_method = $deliveryCode
            payment_method = "cod"
            promotion_code = ""
        } | ConvertTo-Json) | Out-Null

    $orderEnvelope = Invoke-RestMethod `
        -Method Post `
        -Uri "$BaseUrl/checkouts/$checkoutKey/place-order" `
        -Headers $customerHeaders

    $orderID = $orderEnvelope.data.id
    Assert-True (-not [string]::IsNullOrWhiteSpace($orderID)) "Order ID missing"

    $orderEnvelope = Invoke-RestMethod -Method Get -Uri "$BaseUrl/orders/$orderID" -Headers $customerHeaders
    $order = $orderEnvelope.data
    $orderItem = @($order.items)[0]
    $orderItemID = $orderItem.id
    $orderQuantity = [int]$orderItem.quantity

    Assert-Equal $order.payment_method "cod" "Order payment method mismatch"
    Assert-Equal $orderQuantity $moq "Order quantity mismatch"
    Assert-Equal $orderItem.variant_id $variantID "Order variant mismatch"

    $paymentBefore = Invoke-RestMethod -Method Get -Uri "$BaseUrl/orders/$orderID/payment" -Headers $customerHeaders
    Assert-Equal $paymentBefore.data.payment_method "cod" "Payment endpoint method mismatch"
    Assert-True ((-not ($paymentBefore.data.PSObject.Properties.Name -contains "latest_attempt")) -or ($null -eq $paymentBefore.data.PSObject.Properties["latest_attempt"].Value)) "COD should not have provider payment attempt"

    Write-Host "Order: $orderID"
    Write-Host "PASS: cart/checkout/order/payment-status"

    # ========================================================
    # 7. Reservation + frozen COGS + persistent invoice
    # ========================================================

    Write-Step "Inventory reservation + frozen COGS + persistent invoice"

    $reservationState = Invoke-DbScalar @"
SELECT status
FROM inventory_reservations
WHERE
    reference_type = 'order'
    AND reference_id = '$orderID'
    AND variant_id = '$variantID'::uuid;
"@
    Assert-Equal $reservationState "committed" "COD inventory reservation was not committed"

    $inventoryAfterOrder = Invoke-Admin -Method Get -Path "/admin/inventory/$variantID"
    Assert-Equal ([int]$inventoryAfterOrder.data.quantity_on_hand) ($initialStock - $moq) "COD inventory on-hand quantity mismatch"
    Assert-Equal ([int]$inventoryAfterOrder.data.quantity_reserved) 0 "Committed COD order left reserved inventory behind"
    Assert-Equal ([int]$inventoryAfterOrder.data.available_quantity) ($initialStock - $moq) "COD inventory available quantity mismatch"

    $costSnapshot = @(Invoke-DbLines @"
SELECT
    unit_cost_amount,
    line_cost_amount,
    cost_currency
FROM order_items
WHERE id = '$orderItemID'::uuid;
"@)

    Assert-Equal $costSnapshot.Count 1 "Order-item COGS snapshot missing"
    $costParts = @("$($costSnapshot[0])" -split '\|')
    Assert-Equal ([int64]$costParts[0]) $unitCost "Frozen unit COGS mismatch"
    Assert-Equal ([int64]$costParts[1]) $expectedLineCost "Frozen line COGS mismatch"
    Assert-Equal $costParts[2] "BDT" "Frozen COGS currency mismatch"

    $invoice1 = Invoke-RestMethod -Method Get -Uri "$BaseUrl/orders/$orderID/invoice" -Headers $customerHeaders
    $invoice2 = Invoke-RestMethod -Method Get -Uri "$BaseUrl/orders/$orderID/invoice" -Headers $customerHeaders

    Assert-True (-not [string]::IsNullOrWhiteSpace($invoice1.data.id)) "Persistent invoice ID missing"
    Assert-Equal $invoice1.data.id $invoice2.data.id "Invoice replay returned a different invoice"
    Assert-Equal $invoice1.data.invoice_number $invoice2.data.invoice_number "Invoice number changed on replay"
    Assert-Equal $invoice1.data.order_id $orderID "Invoice order mismatch"

    $invoiceCount = [int](Invoke-DbScalar "SELECT count(*) FROM invoices WHERE order_id = '$orderID'::uuid;")
    Assert-Equal $invoiceCount 1 "Expected exactly one persistent invoice"

    Write-Host "Invoice: $($invoice1.data.invoice_number)"
    Write-Host "PASS: reservation committed / COGS frozen / invoice immutable"

    # ========================================================
    # 8. Warehouse allocation -> pick -> pack -> handoff
    # ========================================================

    Write-Step "Warehouse allocation -> pick -> pack -> handoff"

    $warehouse = Invoke-Admin `
        -Method Post `
        -Path "/admin/warehouse/locations" `
        -Body @{
            code = "E2E-$stamp"
            name = "Final E2E Dhaka Warehouse"
            country_code = "BD"
            city = "Dhaka"
            address_line1 = "Final E2E Warehouse"
            status = "active"
            is_default = $false
            allows_self_pickup = $false
        }

    $warehouseID = $warehouse.data.id
    Assert-True (-not [string]::IsNullOrWhiteSpace($warehouseID)) "Warehouse ID missing"

    Invoke-Admin `
        -Method Patch `
        -Path "/admin/warehouse/locations/$warehouseID/map-location" `
        -Body @{ latitude = 23.8103; longitude = 90.4125 } | Out-Null

    $fulfillment = Invoke-Admin `
        -Method Post `
        -Path "/admin/warehouse/fulfillments" `
        -Body @{
            order_item_id = $orderItemID
            warehouse_id = $warehouseID
            inbound_shipment_id = ""
            source = "bangladesh_stock"
            quantity = $orderQuantity
        }

    $fulfillmentID = $fulfillment.data.id
    Assert-Equal $fulfillment.data.status "allocated" "Bangladesh-stock fulfillment was not allocated"

    Invoke-Admin `
        -Method Patch `
        -Path "/admin/orders/$orderID/fulfillment" `
        -Body @{ status = "processing" } | Out-Null

    $picking = Invoke-Admin -Method Post -Path "/admin/warehouse/fulfillments/$fulfillmentID/start-picking"
    Assert-Equal $picking.data.status "picking" "Fulfillment did not enter picking"

    $packed = Invoke-Admin -Method Post -Path "/admin/warehouse/fulfillments/$fulfillmentID/mark-packed"
    Assert-Equal $packed.data.status "packed" "Fulfillment did not become packed"

    $readyForHandoff = Invoke-Admin -Method Post -Path "/admin/warehouse/fulfillments/$fulfillmentID/ready-for-handoff"
    Assert-Equal $readyForHandoff.data.status "ready_for_handoff" "Fulfillment did not become ready for handoff"

    Write-Host "Warehouse:   $warehouseID"
    Write-Host "Fulfillment: $fulfillmentID"
    Write-Host "PASS: allocation/pick/pack/ready"

    # ========================================================
    # 9. Shipment -> dispatch -> provider delivery -> receipt
    # ========================================================

    Write-Step "Shipment/tracking -> delivered -> COD collected"

    $shipment = Invoke-Admin `
        -Method Post `
        -Path "/admin/delivery/orders/$orderID/shipments" `
        -Body @{
            delivery_mode = "courier"
            provider_code = ""
            courier_name = "Final E2E Manual Courier"
            courier_reference = "E2E-C-$stamp"
            tracking_number = "E2E-BD-$stamp"
            tracking_url = ""
        }

    $shipmentID = $shipment.data.shipment.id
    Assert-True (-not [string]::IsNullOrWhiteSpace($shipmentID)) "Shipment ID missing"

    Invoke-Admin `
        -Method Post `
        -Path "/admin/warehouse/handoffs" `
        -Body @{
            shipment_id = $shipmentID
            handoff_type = "courier"
            reference = "E2E-HANDOFF-$stamp"
            items = @(
                @{
                    fulfillment_id = $fulfillmentID
                    quantity = $orderQuantity
                }
            )
        } | Out-Null

    $fulfillmentAfterHandoff = Invoke-Admin -Method Get -Path "/admin/warehouse/fulfillments/$fulfillmentID"
    Assert-Equal $fulfillmentAfterHandoff.data.status "handed_off" "Fulfillment was not marked handed off"

    Invoke-Admin `
        -Method Post `
        -Path "/admin/delivery/shipments/$shipmentID/dispatch" `
        -Body @{ message = "Final E2E courier dispatched" } | Out-Null

    $tracking = Get-CustomerTracking $orderID $customerHeaders
    Assert-TrackingStage $tracking "delivering" "Tracking did not enter delivering"

    Invoke-Admin `
        -Method Post `
        -Path "/admin/delivery/shipments/$shipmentID/tracking-events" `
        -Body @{
            source = "admin"
            event_code = "e2e_location_update"
            status = "shipped"
            message = "INTERNAL-E2E-$stamp"
            public_message = "Your E2E order is out for delivery"
            country_code = "BD"
            city = "Dhaka"
            location_name = "Dhanmondi E2E Delivery Area"
            latitude = 23.7465
            longitude = 90.3760
            customer_visible = $true
            external_event_id = "E2E-LOCATION-$stamp"
            metadata = @{ e2e = "private" }
        } | Out-Null

    Invoke-Admin `
        -Method Post `
        -Path "/admin/delivery/shipments/$shipmentID/provider-delivered" `
        -Body @{
            provider_status = "delivered"
            message = "INTERNAL-E2E-PROVIDER-DELIVERED-$stamp"
            external_event_id = "E2E-DELIVERED-$stamp"
        } | Out-Null

    $trackingBeforeReceipt = Get-CustomerTracking $orderID $customerHeaders
    Assert-TrackingStage $trackingBeforeReceipt "delivering" "Provider delivery incorrectly finalized customer receipt"

    Invoke-RestMethod `
        -Method Post `
        -Uri "$BaseUrl/orders/$orderID/confirm-receipt" `
        -Headers $customerHeaders `
        -ContentType "application/json" `
        -Body (@{ note = "Final E2E customer receipt confirmation" } | ConvertTo-Json) | Out-Null

    $trackingAfterReceipt = Get-CustomerTracking $orderID $customerHeaders
    Assert-TrackingStage $trackingAfterReceipt "delivered" "Customer receipt did not finalize tracking"

    $finalOrderEnvelope = Invoke-RestMethod -Method Get -Uri "$BaseUrl/orders/$orderID" -Headers $customerHeaders
    $finalOrder = $finalOrderEnvelope.data
    Assert-Equal $finalOrder.status "delivered" "Final order status mismatch"
    Assert-Equal $finalOrder.payment_status "cod_collected" "COD was not collected on receipt confirmation"

    $paymentAfter = Invoke-RestMethod -Method Get -Uri "$BaseUrl/orders/$orderID/payment" -Headers $customerHeaders
    Assert-Equal $paymentAfter.data.payment_status "cod_collected" "Payment endpoint did not report cod_collected"

    Write-Host "Shipment: $shipmentID"
    Write-Host "PASS: handoff/dispatch/tracking/receipt/COD collection"

    # ========================================================
    # 10. Verified-purchase review
    # ========================================================

    Write-Step "Verified-purchase review"

    $review = Invoke-RestMethod `
        -Method Post `
        -Uri "$BaseUrl/reviews" `
        -Headers $customerHeaders `
        -ContentType "application/json" `
        -Body (@{
            order_item_id = $orderItemID
            rating = 5
            title = "Final E2E verified review"
            body = "Created by the final cross-module backend validation."
        } | ConvertTo-Json)

    $reviewID = $review.data.id
    Assert-True (-not [string]::IsNullOrWhiteSpace($reviewID)) "Review ID missing"
    Assert-True ($review.data.verified_purchase -eq $true) "Review is not verified purchase"
    Assert-Equal $review.data.product_id $productID "Review product mismatch"

    $summary = Invoke-RestMethod -Method Get -Uri "$BaseUrl/reviews/products/$productID/summary"
    Assert-Equal ([int64]$summary.data.total_reviews) ([int64]1) "Unique product review count mismatch"
    Assert-Equal ([int64]$summary.data.verified_purchase_count) ([int64]1) "Verified review count mismatch"
    Assert-Equal ([double]$summary.data.average_rating) ([double]5) "Average rating mismatch"

    Write-Host "Review: $reviewID"
    Write-Host "PASS: delivered order -> verified public review"

    # ========================================================
    # 11. CRM -> support claim/reply/resolve
    # ========================================================

    Write-Step "CRM/support lifecycle"

    $case = Invoke-RestMethod `
        -Method Post `
        -Uri "$BaseUrl/crm/cases" `
        -Headers $customerHeaders `
        -ContentType "application/json" `
        -Body (@{
            type = "general_question"
            subject = "Final E2E support case"
            message = "Please validate the support lifecycle for this order."
            product_id = $productID
            variant_id = $variantID
            order_id = $orderID
            requested_quantity = 0
        } | ConvertTo-Json)

    $caseID = $case.data.id
    Assert-True (-not [string]::IsNullOrWhiteSpace($caseID)) "CRM case ID missing"
    Assert-Equal $case.data.status "waiting_support" "New CRM case status mismatch"

    Assert-OutboxEventCount "case_id = '$caseID'::uuid AND event_type = 'support.reply'" 0 "Customer-created case unexpectedly emitted support.reply"

    if ([string]::IsNullOrWhiteSpace($SupportIdentifier)) {
        $SupportIdentifier = "EMP-000001"
    }

    $supportIdentifierResolved = [Environment]::GetEnvironmentVariable("E2E_SUPPORT_IDENTIFIER", "Process")
    if (-not [string]::IsNullOrWhiteSpace($supportIdentifierResolved)) {
        $SupportIdentifier = $supportIdentifierResolved
    }

    if ([string]::IsNullOrWhiteSpace($SupportIdentifier)) {
        $SupportIdentifier = Get-EnvOrPromptText "E2E_SUPPORT_IDENTIFIER" "Support staff email / identifier" "EMP-000001"
    }

    $supportPassword = Get-EnvOrPromptSecret "E2E_SUPPORT_PASSWORD" "Support staff password"

    $supportLogin = Invoke-RestMethod `
        -Method Post `
        -Uri "$BaseUrl/support/auth/login" `
        -ContentType "application/json" `
        -Body (@{
            identifier = $SupportIdentifier
            password = $supportPassword
        } | ConvertTo-Json)

    $supportPassword = $null
    $supportToken = $supportLogin.data.tokens.access_token
    Assert-True (-not [string]::IsNullOrWhiteSpace($supportToken)) "Support access token missing"
    $supportHeaders = @{ Authorization = "Bearer $supportToken" }

    Invoke-RestMethod `
        -Method Patch `
        -Uri "$BaseUrl/support/presence" `
        -Headers $supportHeaders `
        -ContentType "application/json" `
        -Body (@{ presence = "available" } | ConvertTo-Json) | Out-Null

    Invoke-RestMethod -Method Post -Uri "$BaseUrl/support/cases/$caseID/claim" -Headers $supportHeaders | Out-Null

    Invoke-RestMethod `
        -Method Post `
        -Uri "$BaseUrl/support/cases/$caseID/messages" `
        -Headers $supportHeaders `
        -ContentType "application/json" `
        -Body (@{
            message = "Internal final-E2E note; customer must not be notified."
            visibility = "internal"
        } | ConvertTo-Json) | Out-Null

    Assert-OutboxEventCount "case_id = '$caseID'::uuid AND event_type = 'support.reply'" 0 "Internal support note created customer notification"

    Invoke-RestMethod `
        -Method Post `
        -Uri "$BaseUrl/support/cases/$caseID/messages" `
        -Headers $supportHeaders `
        -ContentType "application/json" `
        -Body (@{
            message = "Final E2E customer-visible support reply."
            visibility = "customer"
        } | ConvertTo-Json) | Out-Null

    Assert-OutboxEventCount "case_id = '$caseID'::uuid AND event_type = 'support.reply'" 2 "Customer-visible support reply did not create SMS + email rows"

    $customerFollowUp = Invoke-RestMethod `
        -Method Post `
        -Uri "$BaseUrl/crm/cases/$caseID/messages" `
        -Headers $customerHeaders `
        -ContentType "application/json" `
        -Body (@{
            message = "Final E2E customer follow-up after support reply."
        } | ConvertTo-Json)

    Assert-Equal `
        $customerFollowUp.data.author_type `
        "customer" `
        "CRM follow-up was not attributed to customer"

    Assert-Equal `
        $customerFollowUp.data.visibility `
        "customer" `
        "CRM customer follow-up visibility mismatch"

    Invoke-RestMethod -Method Post -Uri "$BaseUrl/support/cases/$caseID/resolve" -Headers $supportHeaders | Out-Null

    $customerCase = Invoke-RestMethod -Method Get -Uri "$BaseUrl/crm/cases/$caseID" -Headers $customerHeaders
    Assert-Equal $customerCase.data.status "resolved" "Resolved support case not visible as resolved to customer"

    Write-Host "Case: $caseID"
    Write-Host "PASS: create/claim/internal-note/reply/customer-follow-up/resolve"

    Write-Step "Inventory low-stock notification crossing"

    $inventoryBefore = Invoke-RestMethod `
        -Method Get `
        -Uri "$BaseUrl/admin/inventory/$variantID" `
        -WebSession $adminSession

    $stock = $inventoryBefore.data

    Assert-True `
        ($null -ne $stock) `
        "Inventory record missing before low-stock test"

    if (
        [int]$stock.available_quantity -le
        [int]$stock.reorder_level
    ) {
        $raiseBy =
            ([int]$stock.reorder_level + 5) -
            [int]$stock.available_quantity

        $raisedStock = Invoke-RestMethod `
            -Method Post `
            -Uri "$BaseUrl/admin/inventory/$variantID/adjust" `
            -WebSession $adminSession `
            -Headers $adminHeaders `
            -ContentType "application/json" `
            -Body (@{
                quantity_delta = $raiseBy
                reason = "final_e2e_low_stock_prep"
                note = "Raise stock above reorder threshold before low-stock crossing test."
            } | ConvertTo-Json)

        $stock = $raisedStock.data

        Assert-True `
            ([int]$stock.available_quantity -gt [int]$stock.reorder_level) `
            "Inventory preparation did not move stock above reorder threshold"
    }

    $lowStockDelta =
        [int]$stock.reorder_level -
        [int]$stock.available_quantity

    Assert-True `
        ($lowStockDelta -lt 0) `
        "Expected negative inventory delta for low-stock crossing"

    $lowStockResult = Invoke-RestMethod `
        -Method Post `
        -Uri "$BaseUrl/admin/inventory/$variantID/adjust" `
        -WebSession $adminSession `
        -Headers $adminHeaders `
        -ContentType "application/json" `
        -Body (@{
            quantity_delta = $lowStockDelta
            reason = "final_e2e_low_stock_crossing"
            note = "Cross available stock onto reorder threshold for notification verification."
        } | ConvertTo-Json)

    Assert-Equal `
        ([int]$lowStockResult.data.available_quantity) `
        ([int]$lowStockResult.data.reorder_level) `
        "Low-stock adjustment landed on reorder threshold"

    Assert-True `
        ([bool]$lowStockResult.data.low_stock) `
        "Inventory is marked low-stock after threshold crossing"

    Write-Host "PASS: inventory high-to-low threshold crossing"

    Write-Step "Customer notification inbox HTTP behavior"

    $customerInbox = Invoke-RestMethod `
        -Method Get `
        -Uri "$BaseUrl/customers/me/notifications?limit=100&offset=0" `
        -Headers $customerHeaders

    $customerNotifications = @($customerInbox.data)

    Assert-Equal ([int]$customerInbox.meta.limit) 100 "Customer notification list limit mismatch"
    Assert-Equal ([int]$customerInbox.meta.offset) 0 "Customer notification list offset mismatch"
    Assert-True ($customerNotifications.Count -gt 0) "Customer notification inbox is empty"

    $customerOrderNotifications = @(
        $customerNotifications |
            Where-Object {
                [string](Get-OptionalPropertyValue $_ "order_id") -eq $orderID
            }
    )

    Assert-True `
        ($customerOrderNotifications.Count -gt 0) `
        "Customer inbox does not contain current order notifications"

    $customerSupportNotifications = @(
        $customerNotifications |
            Where-Object {
                [string](Get-OptionalPropertyValue $_ "case_id") -eq $caseID -and
                [string](Get-OptionalPropertyValue $_ "event_type") -eq "support.reply"
            }
    )

    Assert-Equal `
        $customerSupportNotifications.Count `
        1 `
        "Customer inbox does not contain exactly one support.reply for current case"

    $customerSummaryBefore = Invoke-RestMethod `
        -Method Get `
        -Uri "$BaseUrl/customers/me/notifications/summary" `
        -Headers $customerHeaders

    $customerUnreadBefore =
        [int]$customerSummaryBefore.data.unread_count

    $customerUnreadDBBefore =
        [int](Invoke-DbScalar @"
SELECT count(*)
FROM customer_notifications
WHERE
    customer_id = '$customerID'::uuid
    AND read_at IS NULL;
"@)

    Assert-Equal `
        $customerUnreadBefore `
        $customerUnreadDBBefore `
        "Customer inbox summary unread count disagrees with PostgreSQL"

    Assert-True `
        ($customerUnreadBefore -gt 0) `
        "Customer inbox has no unread notification to test read-one"

    $customerUnreadNotification = @(
        $customerNotifications |
            Where-Object {
                $readAt = Get-OptionalPropertyValue $_ "read_at"
                $null -eq $readAt -or
                [string]::IsNullOrWhiteSpace(
                    [string]$readAt
                )
            }
    )[0]

    $customerNotificationID =
        [string]$customerUnreadNotification.id

    Assert-True `
        (-not [string]::IsNullOrWhiteSpace($customerNotificationID)) `
        "Customer unread notification ID missing"

    $customerReadOne = Invoke-RestMethod `
        -Method Post `
        -Uri "$BaseUrl/customers/me/notifications/$customerNotificationID/read" `
        -Headers $customerHeaders

    Assert-True `
        ([bool]$customerReadOne.data.read) `
        "Customer read-one endpoint did not report read=true"

    $customerSummaryAfterOne = Invoke-RestMethod `
        -Method Get `
        -Uri "$BaseUrl/customers/me/notifications/summary" `
        -Headers $customerHeaders

    Assert-Equal `
        ([int]$customerSummaryAfterOne.data.unread_count) `
        ($customerUnreadBefore - 1) `
        "Customer read-one did not decrement unread count by one"

    $customerReadOneAgain = Invoke-RestMethod `
        -Method Post `
        -Uri "$BaseUrl/customers/me/notifications/$customerNotificationID/read" `
        -Headers $customerHeaders

    Assert-True `
        ([bool]$customerReadOneAgain.data.read) `
        "Customer repeated read-one did not remain idempotently successful"

    $customerSummaryAfterReplay = Invoke-RestMethod `
        -Method Get `
        -Uri "$BaseUrl/customers/me/notifications/summary" `
        -Headers $customerHeaders

    Assert-Equal `
        ([int]$customerSummaryAfterReplay.data.unread_count) `
        ([int]$customerSummaryAfterOne.data.unread_count) `
        "Customer repeated read-one changed unread count"

    $customerUnreadBeforeAll =
        [int]$customerSummaryAfterReplay.data.unread_count

    $customerReadAll = Invoke-RestMethod `
        -Method Post `
        -Uri "$BaseUrl/customers/me/notifications/read-all" `
        -Headers $customerHeaders

    Assert-Equal `
        ([int]$customerReadAll.data.updated) `
        $customerUnreadBeforeAll `
        "Customer read-all updated count mismatch"

    $customerSummaryAfterAll = Invoke-RestMethod `
        -Method Get `
        -Uri "$BaseUrl/customers/me/notifications/summary" `
        -Headers $customerHeaders

    Assert-Equal `
        ([int]$customerSummaryAfterAll.data.unread_count) `
        0 `
        "Customer read-all did not clear unread count"

    $customerUnreadDBAfterAll =
        [int](Invoke-DbScalar @"
SELECT count(*)
FROM customer_notifications
WHERE
    customer_id = '$customerID'::uuid
    AND read_at IS NULL;
"@)

    Assert-Equal `
        $customerUnreadDBAfterAll `
        0 `
        "Customer read-all did not persist read state"

    $customerReadAllAgain = Invoke-RestMethod `
        -Method Post `
        -Uri "$BaseUrl/customers/me/notifications/read-all" `
        -Headers $customerHeaders

    Assert-Equal `
        ([int]$customerReadAllAgain.data.updated) `
        0 `
        "Customer repeated read-all was not idempotent"

    Write-Host "PASS: customer notification list/summary/read-one/read-all"

    Write-Step "Admin notification inbox HTTP behavior"

    $adminInbox = Invoke-Admin `
        -Method Get `
        -Path "/admin/notifications?limit=100&offset=0"

    $adminNotifications = @($adminInbox.data)

    Assert-Equal ([int]$adminInbox.meta.limit) 100 "Admin notification list limit mismatch"
    Assert-Equal ([int]$adminInbox.meta.offset) 0 "Admin notification list offset mismatch"
    Assert-True ($adminNotifications.Count -gt 0) "Admin notification inbox is empty"

    $inventoryNotificationMatches = @(
        $adminNotifications |
            Where-Object {
                [string](Get-OptionalPropertyValue $_ "event_type") -eq "inventory.low_stock" -and
                [string](Get-OptionalPropertyValue $_ "entity_type") -eq "variant" -and
                [string](Get-OptionalPropertyValue $_ "entity_id") -eq $variantID
            }
    )

    Assert-Equal `
        $inventoryNotificationMatches.Count `
        1 `
        "Admin inbox does not contain exactly one low-stock notification for current variant"

    $adminNotificationID =
        [string]$inventoryNotificationMatches[0].id

    Assert-True `
        (-not [string]::IsNullOrWhiteSpace($adminNotificationID)) `
        "Admin low-stock notification ID missing"

    $adminRecipientCount =
        [int](Invoke-DbScalar @"
SELECT count(*)
FROM staff_notification_recipients
WHERE notification_id = '$adminNotificationID'::uuid;
"@)

    Assert-Equal `
        $adminRecipientCount `
        2 `
        "Low-stock staff notification recipient count mismatch"

    $adminCurrentRecipientCount =
        [int](Invoke-DbScalar @"
SELECT count(*)
FROM staff_notification_recipients
WHERE
    notification_id = '$adminNotificationID'::uuid
    AND staff_account_id = '$adminStaffID'::uuid;
"@)

    Assert-Equal `
        $adminCurrentRecipientCount `
        1 `
        "Authenticated Admin is not a recipient of the low-stock notification"

    $adminOriginalUnreadIDs = @(
        Invoke-DbLines @"
SELECT notification_id::text
FROM staff_notification_recipients
WHERE
    staff_account_id = '$adminStaffID'::uuid
    AND read_at IS NULL
ORDER BY notification_id;
"@
    )

    $adminSummaryBefore = Invoke-Admin `
        -Method Get `
        -Path "/admin/notifications/summary"

    $adminUnreadBefore =
        [int]$adminSummaryBefore.data.unread_count

    $adminVisibleUnreadDBBefore =
        [int](Invoke-DbScalar @"
SELECT count(*)
FROM staff_notification_recipients
WHERE
    staff_account_id = '$adminStaffID'::uuid
    AND read_at IS NULL
    AND dismissed_at IS NULL;
"@)

    Assert-Equal `
        $adminUnreadBefore `
        $adminVisibleUnreadDBBefore `
        "Admin inbox summary unread count disagrees with PostgreSQL"

    $adminTargetUnreadBefore =
        [int](Invoke-DbScalar @"
SELECT count(*)
FROM staff_notification_recipients
WHERE
    notification_id = '$adminNotificationID'::uuid
    AND staff_account_id = '$adminStaffID'::uuid
    AND read_at IS NULL;
"@)

    Assert-Equal `
        $adminTargetUnreadBefore `
        1 `
        "Fresh low-stock notification is not unread for authenticated Admin"

    $otherRecipientUnreadBefore =
        [int](Invoke-DbScalar @"
SELECT count(*)
FROM staff_notification_recipients
WHERE
    notification_id = '$adminNotificationID'::uuid
    AND staff_account_id <> '$adminStaffID'::uuid
    AND read_at IS NULL;
"@)

    $adminReadOne = Invoke-Admin `
        -Method Post `
        -Path "/admin/notifications/$adminNotificationID/read"

    Assert-True `
        ([bool]$adminReadOne.data.read) `
        "Admin read-one endpoint did not report read=true"

    $adminTargetUnreadAfter =
        [int](Invoke-DbScalar @"
SELECT count(*)
FROM staff_notification_recipients
WHERE
    notification_id = '$adminNotificationID'::uuid
    AND staff_account_id = '$adminStaffID'::uuid
    AND read_at IS NULL;
"@)

    Assert-Equal `
        $adminTargetUnreadAfter `
        0 `
        "Admin read-one did not persist read state"

    $otherRecipientUnreadAfter =
        [int](Invoke-DbScalar @"
SELECT count(*)
FROM staff_notification_recipients
WHERE
    notification_id = '$adminNotificationID'::uuid
    AND staff_account_id <> '$adminStaffID'::uuid
    AND read_at IS NULL;
"@)

    Assert-Equal `
        $otherRecipientUnreadAfter `
        $otherRecipientUnreadBefore `
        "Admin read-one changed another staff recipient's read state"

    $adminSummaryAfterOne = Invoke-Admin `
        -Method Get `
        -Path "/admin/notifications/summary"

    Assert-Equal `
        ([int]$adminSummaryAfterOne.data.unread_count) `
        ($adminUnreadBefore - 1) `
        "Admin read-one did not decrement unread count by one"

    $adminReadOneAgain = Invoke-Admin `
        -Method Post `
        -Path "/admin/notifications/$adminNotificationID/read"

    Assert-True `
        ([bool]$adminReadOneAgain.data.read) `
        "Admin repeated read-one did not remain idempotently successful"

    $adminSummaryAfterReplay = Invoke-Admin `
        -Method Get `
        -Path "/admin/notifications/summary"

    Assert-Equal `
        ([int]$adminSummaryAfterReplay.data.unread_count) `
        ([int]$adminSummaryAfterOne.data.unread_count) `
        "Admin repeated read-one changed unread count"

    $adminUnreadBeforeAll =
        [int]$adminSummaryAfterReplay.data.unread_count

    Assert-True `
        ($adminUnreadBeforeAll -gt 0) `
        "Admin inbox has no remaining unread rows to exercise read-all"

    $adminReadAll = Invoke-Admin `
        -Method Post `
        -Path "/admin/notifications/read-all"

    Assert-True `
        ([int]$adminReadAll.data.updated -ge $adminUnreadBeforeAll) `
        "Admin read-all updated fewer rows than visible unread count"

    $adminSummaryAfterAll = Invoke-Admin `
        -Method Get `
        -Path "/admin/notifications/summary"

    Assert-Equal `
        ([int]$adminSummaryAfterAll.data.unread_count) `
        0 `
        "Admin read-all did not clear visible unread count"

    $adminVisibleUnreadDBAfterAll =
        [int](Invoke-DbScalar @"
SELECT count(*)
FROM staff_notification_recipients
WHERE
    staff_account_id = '$adminStaffID'::uuid
    AND read_at IS NULL
    AND dismissed_at IS NULL;
"@)

    Assert-Equal `
        $adminVisibleUnreadDBAfterAll `
        0 `
        "Admin read-all did not persist visible read state"

    $adminReadAllAgain = Invoke-Admin `
        -Method Post `
        -Path "/admin/notifications/read-all"

    Assert-Equal `
        ([int]$adminReadAllAgain.data.updated) `
        0 `
        "Admin repeated read-all was not idempotent"

    if ($adminOriginalUnreadIDs.Count -gt 0) {
        $adminOriginalUnreadUUIDs =
            @(
                $adminOriginalUnreadIDs |
                    ForEach-Object {
                        "'$($_)'::uuid"
                    }
            ) -join ","

        $adminRestoredUnread =
            [int](Invoke-DbScalar @"
WITH restored AS (
    UPDATE staff_notification_recipients
    SET read_at = NULL
    WHERE
        staff_account_id = '$adminStaffID'::uuid
        AND notification_id IN ($adminOriginalUnreadUUIDs)
    RETURNING 1
)
SELECT count(*) FROM restored;
"@)

        Assert-Equal `
            $adminRestoredUnread `
            $adminOriginalUnreadIDs.Count `
            "Admin unread-state restoration count mismatch"
    }

    $adminSummaryRestored = Invoke-Admin `
        -Method Get `
        -Path "/admin/notifications/summary"

    Assert-Equal `
        ([int]$adminSummaryRestored.data.unread_count) `
        $adminUnreadBefore `
        "Admin unread state was not restored after read-all verification"

    Write-Host "PASS: admin notification list/summary/read-one/read-all + per-staff isolation"

    Write-Step "Notification inbox dedupe integrity"

    $duplicateCustomerInboxKeys =
        [int](Invoke-DbScalar @"
SELECT count(*)
FROM (
    SELECT dedupe_key
    FROM customer_notifications
    GROUP BY dedupe_key
    HAVING count(*) > 1
) duplicate_keys;
"@)

    Assert-Equal `
        $duplicateCustomerInboxKeys `
        0 `
        "Duplicate customer notification dedupe keys detected"

    $duplicateStaffInboxKeys =
        [int](Invoke-DbScalar @"
SELECT count(*)
FROM (
    SELECT dedupe_key
    FROM staff_notifications
    GROUP BY dedupe_key
    HAVING count(*) > 1
) duplicate_keys;
"@)

    Assert-Equal `
        $duplicateStaffInboxKeys `
        0 `
        "Duplicate staff notification dedupe keys detected"

    $duplicateStaffRecipients =
        [int](Invoke-DbScalar @"
SELECT count(*)
FROM (
    SELECT
        notification_id,
        staff_account_id
    FROM staff_notification_recipients
    GROUP BY
        notification_id,
        staff_account_id
    HAVING count(*) > 1
) duplicate_recipients;
"@)

    Assert-Equal `
        $duplicateStaffRecipients `
        0 `
        "Duplicate staff notification recipients detected"

    $currentLowStockNotificationCount =
        [int](Invoke-DbScalar @"
SELECT count(*)
FROM staff_notifications
WHERE
    event_type = 'inventory.low_stock'
    AND entity_type = 'variant'
    AND entity_id = '$variantID';
"@)

    Assert-Equal `
        $currentLowStockNotificationCount `
        1 `
        "Current E2E variant has duplicate or missing low-stock staff notification"

    $currentSupportCreatedCount =
        [int](Invoke-DbScalar @"
SELECT count(*)
FROM staff_notifications
WHERE
    event_type = 'support.case.created'
    AND entity_type = 'crm_case'
    AND entity_id = '$caseID';
"@)

    Assert-Equal `
        $currentSupportCreatedCount `
        1 `
        "Current E2E case has duplicate or missing support.case.created staff notification"

    $currentSupportReplyCount =
        [int](Invoke-DbScalar @"
SELECT count(*)
FROM staff_notifications
WHERE
    event_type = 'support.customer_reply'
    AND entity_type = 'crm_case'
    AND entity_id = '$caseID';
"@)

    Assert-Equal `
        $currentSupportReplyCount `
        1 `
        "Current E2E case has duplicate or missing support.customer_reply staff notification"

    Write-Host "PASS: notification inbox dedupe + current producer event uniqueness"



    # ========================================================
    # 12. Notification outbox -> scheduler -> Redis -> worker
    # ========================================================

    Write-Step "Notification outbox -> scheduler -> Redis -> worker"

    Assert-OutboxEventCount "order_id = '$orderID'::uuid AND event_type = 'order.placed'" 2 "order.placed outbox rows missing"
    Assert-OutboxEventCount "order_id = '$orderID'::uuid AND event_type = 'invoice.issued'" 2 "invoice.issued outbox rows missing"
    Assert-OutboxEventCount "order_id = '$orderID'::uuid AND event_type = 'order.processing'" 2 "order.processing outbox rows missing"
    Assert-OutboxEventCount "order_id = '$orderID'::uuid AND event_type = 'delivery.dispatched'" 2 "delivery.dispatched outbox rows missing"
    Assert-OutboxEventCount "order_id = '$orderID'::uuid AND event_type = 'delivery.delivered'" 2 "delivery.delivered outbox rows missing"
    Assert-OutboxEventCount "case_id = '$caseID'::uuid AND event_type = 'support.reply'" 2 "support.reply outbox rows missing"

    Wait-OutboxTerminal $orderID $caseID

    $failedOutbox = [int](Invoke-DbScalar @"
SELECT count(*)
FROM notification_outbox
WHERE
    (order_id = '$orderID'::uuid OR case_id = '$caseID'::uuid)
    AND status = 'failed';
"@)
    Assert-Equal $failedOutbox 0 "Final E2E outbox contains failed rows"

    $providerUnavailable = [int](Invoke-DbScalar @"
SELECT count(*)
FROM notification_outbox
WHERE
    (order_id = '$orderID'::uuid OR case_id = '$caseID'::uuid)
    AND status = 'skipped'
    AND skip_reason = 'provider_unavailable';
"@)
    Assert-True ($providerUnavailable -gt 0) "No provider_unavailable notification was observed"

    $preferenceDisabled = [int](Invoke-DbScalar @"
SELECT count(*)
FROM notification_outbox
WHERE
    (order_id = '$orderID'::uuid OR case_id = '$caseID'::uuid)
    AND status = 'skipped'
    AND skip_reason = 'preference_disabled';
"@)
    Assert-True ($preferenceDisabled -gt 0) "No preference_disabled notification was observed"

    $duplicateDedupe = [int](Invoke-DbScalar @"
SELECT count(*)
FROM (
    SELECT dedupe_key
    FROM notification_outbox
    WHERE order_id = '$orderID'::uuid OR case_id = '$caseID'::uuid
    GROUP BY dedupe_key
    HAVING count(*) > 1
) duplicate_keys;
"@)
    Assert-Equal $duplicateDedupe 0 "Duplicate notification dedupe keys detected"

    & docker compose exec -T api /app/bin/queue-inspect
    if ($LASTEXITCODE -ne 0) {
        throw "queue-inspect failed (exit code $LASTEXITCODE)"
    }

    Write-Host "provider_unavailable rows: $providerUnavailable"
    Write-Host "preference_disabled rows: $preferenceDisabled"
    Write-Host "PASS: outbox -> scheduler -> Redis -> worker terminal processing"

    # ========================================================
    # 13. Finance expense ledger + idempotency
    # ========================================================

    Write-Step "Finance expense ledger + idempotency"

    $expenseAmount = [int64]700
    $expenseKey = "final-e2e-expense-$stamp"
    $expenseBody = @{
        category = "delivery"
        amount = $expenseAmount
        currency = "BDT"
        occurred_at = [DateTimeOffset]::UtcNow.ToString("o")
        description = "Final E2E order-linked delivery expense"
        order_id = $orderID
        reference = "FINAL-E2E-$stamp"
    }

    $expenseHeaders = @{ "Idempotency-Key" = $expenseKey }

    $expenseCreate = Invoke-Admin `
        -Method Post `
        -Path "/admin/finance/expenses" `
        -Body $expenseBody `
        -ExtraHeaders $expenseHeaders

    Assert-True ($expenseCreate.data.inserted -eq $true) "First finance expense request was not inserted"
    $expenseID = $expenseCreate.data.expense.id
    Assert-True (-not [string]::IsNullOrWhiteSpace($expenseID)) "Expense ID missing"
    Assert-Equal $expenseCreate.data.expense.order_id $orderID "Expense order link mismatch"

    $expenseReplay = Invoke-Admin `
        -Method Post `
        -Path "/admin/finance/expenses" `
        -Body $expenseBody `
        -ExtraHeaders $expenseHeaders

    Assert-True ($expenseReplay.data.inserted -eq $false) "Finance expense replay was not deduplicated"
    Assert-Equal $expenseReplay.data.expense.id $expenseID "Expense replay returned different ID"

    $expenseCount = [int](Invoke-DbScalar "SELECT count(*) FROM finance_expenses WHERE id = '$expenseID'::uuid;")
    Assert-Equal $expenseCount 1 "Finance idempotency replay created duplicate row"

    Write-Host "Expense: $expenseID"
    Write-Host "PASS: order-linked expense + idempotent replay"

    # ========================================================
    # 14. P&L delta reconciliation + unique product profitability
    # ========================================================

    Write-Step "P&L reconciliation + complete COGS profitability"

    $afterResponse = Get-ProfitLoss $dhakaDate
    $after = $afterResponse.data

    Assert-Equal `
        ([int64]$after.collected_orders - [int64]$baseline.collected_orders) `
        ([int64]1) `
        "Collected order delta mismatch"

    Assert-Equal `
        ([int64]$after.units_sold - [int64]$baseline.units_sold) `
        ([int64]$moq) `
        "Units-sold delta mismatch"

    Assert-Equal `
        ([int64]$after.gross_merchandise_revenue_amount - [int64]$baseline.gross_merchandise_revenue_amount) `
        ([int64]$finalOrder.subtotal_amount) `
        "Gross merchandise delta mismatch"

    Assert-Equal `
        ([int64]$after.discount_amount - [int64]$baseline.discount_amount) `
        ([int64]$finalOrder.discount_amount) `
        "Discount delta mismatch"

    Assert-Equal `
        ([int64]$after.shipping_revenue_amount - [int64]$baseline.shipping_revenue_amount) `
        ([int64]$finalOrder.shipping_amount) `
        "Shipping revenue delta mismatch"

    Assert-Equal `
        ([int64]$after.gross_collected_revenue_amount - [int64]$baseline.gross_collected_revenue_amount) `
        ([int64]$finalOrder.total_amount) `
        "Gross collected revenue delta mismatch"

    Assert-Equal `
        ([int64]$after.costed_units - [int64]$baseline.costed_units) `
        ([int64]$moq) `
        "Costed-unit delta mismatch"

    Assert-Equal `
        ([int64]$after.missing_cost_units - [int64]$baseline.missing_cost_units) `
        ([int64]0) `
        "New order introduced missing COGS units"

    Assert-Equal `
        ([int64]$after.gross_cogs_amount - [int64]$baseline.gross_cogs_amount) `
        ([int64]$expectedLineCost) `
        "Gross COGS delta mismatch"

    Assert-Equal `
        ([int64]$after.recorded_expense_entries - [int64]$baseline.recorded_expense_entries) `
        ([int64]1) `
        "Recorded expense entry delta mismatch"

    Assert-Equal `
        ([int64]$after.recorded_expenses_amount - [int64]$baseline.recorded_expenses_amount) `
        $expenseAmount `
        "Recorded expense amount delta mismatch"

    $profitability = Invoke-RestMethod `
        -Method Get `
        -Uri "$BaseUrl/admin/finance/products/profitability?from=$dhakaDate&to=$dhakaDate&currency=BDT&limit=50" `
        -WebSession $script:AdminSession

    $variantProfitability = @($profitability.data | Where-Object { $_.variant_id -eq $variantID })
    Assert-Equal $variantProfitability.Count 1 "Unique E2E variant missing from product profitability"

    $vp = $variantProfitability[0]
    Assert-Equal ([int64]$vp.collected_orders) ([int64]1) "Variant profitability collected-order count mismatch"
    Assert-Equal ([int64]$vp.units_sold) ([int64]$moq) "Variant profitability units mismatch"
    Assert-Equal ([int64]$vp.missing_cost_units) ([int64]0) "Variant profitability has missing COGS"
    Assert-Equal ([int64]$vp.cogs_coverage_bps) ([int64]10000) "Variant COGS coverage is not 100%"
    Assert-True ($vp.profit_complete -eq $true) "Unique E2E variant profit is not complete"
    Assert-True ($null -ne $vp.gross_profit_before_refunds_amount) "Unique E2E variant gross profit is null"

    $expectedVariantProfit = [int64]$vp.net_merchandise_revenue_amount - [int64]$vp.known_cogs_amount
    Assert-Equal ([int64]$vp.gross_profit_before_refunds_amount) $expectedVariantProfit "Variant gross profit mismatch"

    Write-Host "Variant gross profit: $($vp.gross_profit_before_refunds_amount) BDT"
    Write-Host "PASS: P&L deltas reconcile and new-order COGS/profit is complete"

    # ========================================================
    # 15. Final integrity / ownership / service sanity
    # ========================================================

    Write-Step "Final integrity checks"

    $reviewRows = [int](Invoke-DbScalar "SELECT count(*) FROM reviews WHERE id = '$reviewID'::uuid AND deleted_at IS NULL;")
    Assert-Equal $reviewRows 1 "Review persistence mismatch"

    $caseRows = [int](Invoke-DbScalar "SELECT count(*) FROM crm_cases WHERE id = '$caseID'::uuid;")
    Assert-Equal $caseRows 1 "CRM case persistence mismatch"

    $fulfillmentRows = [int](Invoke-DbScalar "SELECT count(*) FROM warehouse_fulfillments WHERE id = '$fulfillmentID'::uuid AND status = 'handed_off';")
    Assert-Equal $fulfillmentRows 1 "Warehouse fulfillment final state mismatch"

    $shipmentRows = [int](Invoke-DbScalar "SELECT count(*) FROM shipments WHERE id = '$shipmentID'::uuid;")
    Assert-Equal $shipmentRows 1 "Delivery shipment persistence mismatch"

    $orderRows = [int](Invoke-DbScalar "SELECT count(*) FROM orders WHERE id = '$orderID'::uuid AND status = 'delivered' AND payment_status = 'cod_collected';")
    Assert-Equal $orderRows 1 "Delivered/COD-collected order persistence mismatch"

    $runningServices = @(& docker compose ps --status running --services)
    if ($LASTEXITCODE -ne 0) { throw "docker compose ps failed after E2E" }
    foreach ($service in @("postgres", "redis", "api", "scheduler", "worker")) {
        Assert-True ($runningServices -contains $service) "Service stopped during final E2E: $service"
    }

    Write-Host "PASS: persistent state and Docker services healthy"

    Write-Host ""
    Write-Host "============================================================"
    Write-Host " FINAL FULL CROSS-MODULE BLACK-BOX E2E: PASS"
    Write-Host "============================================================"
    Write-Host ""
    Write-Host "Validated:"
    Write-Host "  [PASS] Docker API/PostgreSQL/Redis runtime"
    Write-Host "  [PASS] admin MFA"
    Write-Host "  [PASS] catalog create -> public product discovery"
    Write-Host "  [PASS] customer register/login"
    Write-Host "  [PASS] cart -> checkout -> COD order"
    Write-Host "  [PASS] inventory reservation commit"
    Write-Host "  [PASS] frozen order-item COGS"
    Write-Host "  [PASS] immutable persistent invoice"
    Write-Host "  [PASS] warehouse allocation/pick/pack/handoff"
    Write-Host "  [PASS] shipment/tracking/customer receipt"
    Write-Host "  [PASS] COD collection"
    Write-Host "  [PASS] verified-purchase review"
    Write-Host "  [PASS] CRM/support claim/reply/customer-follow-up/resolve"
    Write-Host "  [PASS] inventory low-stock staff notification crossing"
    Write-Host "  [PASS] customer notification inbox list/summary/read-one/read-all"
    Write-Host "  [PASS] Admin notification inbox list/summary/read-one/read-all"
    Write-Host "  [PASS] per-staff notification read-state isolation"
    Write-Host "  [PASS] notification inbox dedupe integrity"
    Write-Host "  [PASS] notification outbox -> scheduler -> Redis -> worker"
    Write-Host "  [PASS] provider_unavailable / preference_disabled truthfulness"
    Write-Host "  [PASS] finance expense idempotency"
    Write-Host "  [PASS] P&L delta reconciliation"
    Write-Host "  [PASS] 100% COGS coverage + non-null profit for isolated new variant"
    Write-Host "  [PASS] final persistent-state integrity"
    Write-Host ""
    Write-Host "Artifacts retained intentionally for auditability:"
    Write-Host "  Product ID:     $productID"
    Write-Host "  Variant ID:     $variantID"
    Write-Host "  Customer ID:    $customerID"
    Write-Host "  Order ID:       $orderID"
    Write-Host "  Invoice ID:     $($invoice1.data.id)"
    Write-Host "  Warehouse ID:   $warehouseID"
    Write-Host "  Fulfillment ID: $fulfillmentID"
    Write-Host "  Shipment ID:    $shipmentID"
    Write-Host "  Review ID:      $reviewID"
    Write-Host "  Case ID:        $caseID"
    Write-Host "  Expense ID:     $expenseID"
    Write-Host ""
    Write-Host "Backend freeze gate: PASS"
    Write-Host ""
}
catch {
    Write-Host ""
    Write-Host "============================================================"
    Write-Host " FINAL FULL CROSS-MODULE BLACK-BOX E2E: FAIL"
    Write-Host "============================================================"
    Write-Host ""
    Write-Host $_.Exception.Message
    Write-Host ""

    try {
        Write-Host "Recent Docker logs:"
        & docker compose logs --tail 80 api scheduler worker postgres redis
    }
    catch {
        Write-Host "Could not collect Docker logs."
    }

    exit 1
}
finally {
    Pop-Location
}



