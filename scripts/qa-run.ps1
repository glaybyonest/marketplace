param(
  [string]$BaseUrl = "http://localhost:18080",
  [string]$ArtifactsDir = "qa-artifacts",
  [string]$DbContainer = "marketplace-postgres"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
$artifactsPath = Join-Path $root $ArtifactsDir
$rawPath = Join-Path $artifactsPath "raw"

New-Item -ItemType Directory -Force $artifactsPath | Out-Null
New-Item -ItemType Directory -Force $rawPath | Out-Null

Add-Type -AssemblyName System.Net.Http

$script:Client = [System.Net.Http.HttpClient]::new()
$script:Client.Timeout = [TimeSpan]::FromSeconds(60)
$script:Utf8 = [System.Text.Encoding]::UTF8

function New-State {
  $runId = [DateTimeOffset]::UtcNow.ToString("yyyyMMddHHmmss")
  return [ordered]@{
    run_id                  = $runId
    base_url                = $BaseUrl.TrimEnd("/")
    customer_email          = "qa.codex+$runId@example.com"
    customer_password       = "Secure123!"
    customer_new_password   = "NewSecure123!"
    customer_full_name      = "QA User 001"
    customer_tokens         = $null
    customer_tokens_2       = $null
    verify_token            = $null
    reset_token             = $null
    product                 = $null
    place_crud_id           = $null
    order_place_id          = $null
    order_id                = $null
    secondary_session_id    = $null
    seller_category_id      = $null
    seller_tokens           = $null
    seller_product_id       = $null
    results                 = @()
  }
}

function Save-TextArtifact {
  param([string]$Name, [string]$Content)
  $path = Join-Path $rawPath $Name
  [System.IO.File]::WriteAllText($path, $Content, $script:Utf8)
  return $path
}

function Save-JsonArtifact {
  param([string]$Name, $Data)
  return Save-TextArtifact -Name $Name -Content ($Data | ConvertTo-Json -Depth 100)
}

function Invoke-Api {
  param(
    [string]$Method,
    [string]$Path,
    $Body = $null,
    [hashtable]$Headers = @{},
    [string]$ArtifactName
  )

  $uri = if ($Path.StartsWith("http")) { $Path } else { "$($state.base_url)$Path" }
  $request = [System.Net.Http.HttpRequestMessage]::new([System.Net.Http.HttpMethod]::$Method, $uri)
  foreach ($header in $Headers.GetEnumerator()) {
    $null = $request.Headers.TryAddWithoutValidation([string]$header.Key, [string]$header.Value)
  }

  $requestBody = $null
  if ($null -ne $Body) {
    $requestBody = if ($Body -is [string]) { $Body } else { $Body | ConvertTo-Json -Depth 100 -Compress }
    $request.Content = [System.Net.Http.StringContent]::new($requestBody, $script:Utf8, "application/json")
  }

  $response = $script:Client.SendAsync($request).GetAwaiter().GetResult()
  $responseText = $response.Content.ReadAsStringAsync().GetAwaiter().GetResult()
  $responseJson = $null
  if ($responseText) {
    try {
      $responseJson = $responseText | ConvertFrom-Json -Depth 100
    }
    catch {
      $responseJson = $null
    }
  }

  $headersOut = [ordered]@{}
  foreach ($header in $response.Headers) {
    $headersOut[$header.Key] = ($header.Value -join ", ")
  }
  foreach ($header in $response.Content.Headers) {
    $headersOut[$header.Key] = ($header.Value -join ", ")
  }

  $artifact = [ordered]@{
    method        = $Method
    path          = $Path
    url           = $uri
    status_code   = [int]$response.StatusCode
    request_body  = $requestBody
    request_header = $Headers
    response_body = $responseText
    response_json = $responseJson
    response_head = $headersOut
  }

  if ($ArtifactName) {
    $artifact["artifact_path"] = Save-JsonArtifact -Name $ArtifactName -Data $artifact
  }

  return $artifact
}

function Invoke-DbScalar {
  param([string]$Sql)
  $safeSql = $Sql.Replace('"', '\"')
  $output = docker exec $DbContainer sh -lc "psql -U postgres -d marketplace -t -A -c ""$safeSql"""
  if ($LASTEXITCODE -ne 0) {
    throw "psql command failed: $Sql"
  }
  return ($output | Out-String).Trim()
}

function Invoke-DbRow {
  param([string]$Sql)
  $raw = Invoke-DbScalar -Sql "SELECT COALESCE(row_to_json(q)::text, '') FROM ($Sql) q;"
  if ([string]::IsNullOrWhiteSpace($raw)) {
    return $null
  }
  return $raw | ConvertFrom-Json -Depth 100
}

function Get-LatestEmailToken {
  param([string]$Email, [string]$SubjectLike)

  $safeEmail = $Email.Replace("'", "''")
  $safeSubject = $SubjectLike.Replace("'", "''")
  $row = Invoke-DbRow -Sql @"
SELECT
  id,
  recipient,
  subject,
  body_text,
  status,
  created_at,
  sent_at
FROM email_jobs
WHERE recipient = '$safeEmail'
  AND subject ILIKE '%$safeSubject%'
ORDER BY created_at DESC
LIMIT 1
"@
  if ($null -eq $row) {
    throw "Email job not found for $Email / $SubjectLike"
  }
  $match = [regex]::Match([string]$row.body_text, "token=([A-Za-z0-9_\-\.]+)")
  if (-not $match.Success) {
    throw "Token not found in email body"
  }
  return [ordered]@{
    token = $match.Groups[1].Value
    email = $row
  }
}

function Record-Case {
  param(
    [string]$Id,
    [string]$Priority,
    [string]$Name,
    [string]$Precondition,
    [string[]]$Steps,
    $TestData,
    [string]$Expected,
    [string]$Actual,
    [string]$Status,
    [string[]]$Evidence,
    [string]$Comment = ""
  )
  $state.results += [ordered]@{
    id               = $Id
    priority         = $Priority
    name             = $Name
    precondition     = $Precondition
    steps            = $Steps
    test_data        = $TestData
    expected_result  = $Expected
    actual_result    = $Actual
    status           = $Status
    evidence         = $Evidence
    comment          = $Comment
  }
}

$script:state = New-State

function Find-SeedProduct {
  $row = Invoke-DbRow -Sql @"
SELECT
  p.id,
  p.name,
  p.slug,
  p.price,
  p.stock_qty,
  p.category_id
FROM products p
WHERE p.is_active = TRUE
  AND p.stock_qty >= 10
ORDER BY p.stock_qty DESC, p.created_at ASC
LIMIT 1
"@
  if ($null -eq $row) {
    throw "No active product with sufficient stock found"
  }
  return $row
}

function Find-SeedCategory {
  $row = Invoke-DbRow -Sql @"
SELECT id, name, slug
FROM categories
ORDER BY created_at ASC
LIMIT 1
"@
  if ($null -eq $row) {
    throw "No category found"
  }
  return $row
}

function Get-AuthHeader {
  param([string]$AccessToken)
  return @{ Authorization = "Bearer $AccessToken" }
}

function Login-User {
  param([string]$Email, [string]$Password, [string]$ArtifactName)
  return Invoke-Api -Method Post -Path "/api/v1/auth/login" -ArtifactName $ArtifactName -Body @{
    email = $Email
    password = $Password
  }
}

function Write-Reports {
  $jsonPath = Join-Path $artifactsPath "test-results.json"
  $mdPath = Join-Path $artifactsPath "test-report.md"

  $summaryLines = foreach ($result in $state.results) {
    $actual = $result.actual_result.Replace("`r", " ").Replace("`n", " ")
    "| $($result.id) | $($result.name) | $actual | $($result.status) |"
  }

  $detailBlocks = foreach ($result in $state.results) {
@"
РўРµСЃС‚РѕРІС‹Р№ РїСЂРёРјРµСЂ #: $($result.id)
РџСЂРёРѕСЂРёС‚РµС‚: $($result.priority)
РќР°Р·РІР°РЅРёРµ: $($result.name)
РџСЂРµРґРІР°СЂРёС‚РµР»СЊРЅРѕРµ СѓСЃР»РѕРІРёРµ: $($result.precondition)
РЁР°РіРё РІС‹РїРѕР»РЅРµРЅРёСЏ: $([string]::Join("; ", $result.steps))
РўРµСЃС‚РѕРІС‹Рµ РґР°РЅРЅС‹Рµ: $(($result.test_data | ConvertTo-Json -Depth 20 -Compress))
РћР¶РёРґР°РµРјС‹Р№ СЂРµР·СѓР»СЊС‚Р°С‚: $($result.expected_result)
Р¤Р°РєС‚РёС‡РµСЃРєРёР№ СЂРµР·СѓР»СЊС‚Р°С‚: $($result.actual_result)
РЎС‚Р°С‚СѓСЃ: $($result.status)
Р”РѕРєР°Р·Р°С‚РµР»СЊСЃС‚РІР°: $([string]::Join("; ", $result.evidence))
РљРѕРјРјРµРЅС‚Р°СЂРёР№: $($result.comment)
"@
  }

  $md = @"
# QA Test Report

- РљР°Рє РїРѕРґРЅСЏР» РїСЂРѕРµРєС‚: `docker compose up -d postgres`, backend Р»РѕРєР°Р»СЊРЅРѕ С‡РµСЂРµР· `scripts/start-qa-api.ps1`.
- РљР°РєРёРµ СЃРµСЂРІРёСЃС‹ Р·Р°РїСѓСЃРєР°Р»: `postgres` РІ Docker, backend API РЅР° `18080`.
- РљР°РєРѕР№ base URL РёСЃРїРѕР»СЊР·РѕРІР°Р»: `$($state.base_url)`.
- РљР°РєРѕР№ auth mode РёСЃРїРѕР»СЊР·РѕРІР°Р»: bearer (`AUTH_COOKIE_MODE=false`).
- Р“РґРµ Р±СЂР°Р» email verify/reset С‚РѕРєРµРЅС‹: РёР· `email_jobs.body_text` С‡РµСЂРµР· `docker exec ... psql`.

| ID | РќР°Р·РІР°РЅРёРµ С‚РµСЃС‚Р° | Р¤Р°РєС‚РёС‡РµСЃРєРёР№ СЂРµР·СѓР»СЊС‚Р°С‚ | РЎС‚Р°С‚СѓСЃ |
| --- | --- | --- | --- |
$([string]::Join("`n", $summaryLines))

$([string]::Join("`n`n", $detailBlocks))
"@

  $machine = [ordered]@{
    generated_at = [DateTimeOffset]::UtcNow.ToString("o")
    base_url = $state.base_url
    auth_mode = "bearer"
    token_source = "email_jobs.body_text"
    results = $state.results
  }

  [System.IO.File]::WriteAllText($jsonPath, ($machine | ConvertTo-Json -Depth 100), $script:Utf8)
  [System.IO.File]::WriteAllText($mdPath, $md, $script:Utf8)

  return [ordered]@{
    json = $jsonPath
    md = $mdPath
  }
}

try {
  $ready = Invoke-Api -Method Get -Path "/readyz" -ArtifactName "00-readyz.json"
  if ($ready.status_code -ne 200) {
    throw "Backend is not ready: $($ready.status_code)"
  }

  $state.product = Find-SeedProduct
  $state.seller_category_id = (Find-SeedCategory).id

  $register = Invoke-Api -Method Post -Path "/api/v1/auth/register" -ArtifactName "01-auth-register.json" -Body @{
    email = $state.customer_email
    password = $state.customer_password
    full_name = $state.customer_full_name
  }
  $loginBeforeVerify = Login-User -Email $state.customer_email -Password $state.customer_password -ArtifactName "01-auth-login-before-verify.json"
  $auth1Ok = (
    $register.status_code -eq 201 -and
    $register.response_json.data.user.is_email_verified -eq $false -and
    $register.response_json.data.requires_email_verification -eq $true -and
    $loginBeforeVerify.status_code -eq 403 -and
    -not $loginBeforeVerify.response_json.data.tokens
  )
  Record-Case -Id "TC_API_AUTH_001" -Priority "Р’С‹СЃРѕРєРёР№" -Name "Р РµРіРёСЃС‚СЂР°С†РёСЏ РЅРѕРІРѕРіРѕ РїРѕР»СЊР·РѕРІР°С‚РµР»СЏ Рё Р·Р°РїСЂРµС‚ РІС…РѕРґР° РґРѕ РїРѕРґС‚РІРµСЂР¶РґРµРЅРёСЏ email" `
    -Precondition "Р—Р°РїСѓС‰РµРЅ backend РЅР° $($state.base_url); email $($state.customer_email) СЂР°РЅРµРµ РЅРµ РёСЃРїРѕР»СЊР·РѕРІР°Р»СЃСЏ." `
    -Steps @("POST /api/v1/auth/register", "POST /api/v1/auth/login РґРѕ РІРµСЂРёС„РёРєР°С†РёРё") `
    -TestData @{ email = $state.customer_email; password = $state.customer_password; full_name = $state.customer_full_name } `
    -Expected "201 РЅР° СЂРµРіРёСЃС‚СЂР°С†РёСЋ; requires_email_verification=true; is_email_verified=false; login РґРѕ РїРѕРґС‚РІРµСЂР¶РґРµРЅРёСЏ РґР°РµС‚ 403 Рё РЅРµ РІС‹РґР°РµС‚ С‚РѕРєРµРЅС‹." `
    -Actual "register -> $($register.status_code), is_email_verified=$($register.response_json.data.user.is_email_verified), requires_email_verification=$($register.response_json.data.requires_email_verification); login РґРѕ РІРµСЂРёС„РёРєР°С†РёРё -> $($loginBeforeVerify.status_code), tokens_present=$([bool]$loginBeforeVerify.response_json.data.tokens)." `
    -Status $(if ($auth1Ok) { "Р—Р°С‡РµС‚" } else { "РќРµР·Р°С‡РµС‚" }) `
    -Evidence @(
      "POST /api/v1/auth/register -> $($register.status_code), Р°СЂС‚РµС„Р°РєС‚: $($register.artifact_path)",
      "POST /api/v1/auth/login -> $($loginBeforeVerify.status_code), Р°СЂС‚РµС„Р°РєС‚: $($loginBeforeVerify.artifact_path)"
    )

  $verifyRequest = Invoke-Api -Method Post -Path "/api/v1/auth/verify-email/request" -ArtifactName "02-auth-verify-request.json" -Body @{
    email = $state.customer_email
  }
  $verifyInfo = Get-LatestEmailToken -Email $state.customer_email -SubjectLike "verify"
  $state.verify_token = $verifyInfo.token
  Save-JsonArtifact -Name "02-auth-verify-email-job.json" -Data $verifyInfo.email | Out-Null
  $verifyConfirm = Invoke-Api -Method Post -Path "/api/v1/auth/verify-email/confirm" -ArtifactName "02-auth-verify-confirm.json" -Body @{
    token = $state.verify_token
  }
  $loginAfterVerify = Login-User -Email $state.customer_email -Password $state.customer_password -ArtifactName "02-auth-login-after-verify.json"
  $state.customer_tokens = $loginAfterVerify.response_json.data.tokens
  $auth2Ok = (
    $verifyRequest.status_code -eq 200 -and
    $verifyConfirm.status_code -eq 200 -and
    $loginAfterVerify.status_code -eq 200 -and
    [bool]$state.customer_tokens.access_token -and
    [bool]$state.customer_tokens.refresh_token -and
    $loginAfterVerify.response_json.data.requires_email_verification -eq $false
  )
  Record-Case -Id "TC_API_AUTH_002" -Priority "Р’С‹СЃРѕРєРёР№" -Name "РџРѕРґС‚РІРµСЂР¶РґРµРЅРёРµ email С‚РѕРєРµРЅРѕРј Рё СѓСЃРїРµС€РЅР°СЏ Р°РІС‚РѕСЂРёР·Р°С†РёСЏ РїРѕР»СЊР·РѕРІР°С‚РµР»СЏ" `
    -Precondition "РџРѕР»СЊР·РѕРІР°С‚РµР»СЊ Р·Р°СЂРµРіРёСЃС‚СЂРёСЂРѕРІР°РЅ Рё РµС‰Рµ РЅРµ РІРµСЂРёС„РёС†РёСЂРѕРІР°РЅ." `
    -Steps @("POST /api/v1/auth/verify-email/request", "РџРѕР»СѓС‡РµРЅРёРµ verify_token РёР· email_jobs", "POST /api/v1/auth/verify-email/confirm", "POST /api/v1/auth/login") `
    -TestData @{ email = $state.customer_email; verify_token = $state.verify_token } `
    -Expected "Р—Р°РїСЂРѕСЃ СЃСЃС‹Р»РєРё 200; confirm 200; login 200; РµСЃС‚СЊ access_token Рё refresh_token; requires_email_verification=false." `
    -Actual "verify request -> $($verifyRequest.status_code); verify confirm -> $($verifyConfirm.status_code); login -> $($loginAfterVerify.status_code); access_token_present=$([bool]$state.customer_tokens.access_token); refresh_token_present=$([bool]$state.customer_tokens.refresh_token); requires_email_verification=$($loginAfterVerify.response_json.data.requires_email_verification)." `
    -Status $(if ($auth2Ok) { "Р—Р°С‡РµС‚" } else { "РќРµР·Р°С‡РµС‚" }) `
    -Evidence @(
      "POST /api/v1/auth/verify-email/request -> $($verifyRequest.status_code), Р°СЂС‚РµС„Р°РєС‚: $($verifyRequest.artifact_path)",
      "email_jobs verify token=$($state.verify_token), Р°СЂС‚РµС„Р°РєС‚: $(Join-Path $rawPath '02-auth-verify-email-job.json')",
      "POST /api/v1/auth/verify-email/confirm -> $($verifyConfirm.status_code), Р°СЂС‚РµС„Р°РєС‚: $($verifyConfirm.artifact_path)",
      "POST /api/v1/auth/login -> $($loginAfterVerify.status_code), Р°СЂС‚РµС„Р°РєС‚: $($loginAfterVerify.artifact_path)"
    )

  $resetRequest = Invoke-Api -Method Post -Path "/api/v1/auth/password-reset/request" -ArtifactName "03-auth-reset-request.json" -Body @{
    email = $state.customer_email
  }
  $resetInfo = Get-LatestEmailToken -Email $state.customer_email -SubjectLike "reset"
  $state.reset_token = $resetInfo.token
  Save-JsonArtifact -Name "03-auth-reset-email-job.json" -Data $resetInfo.email | Out-Null
  $resetConfirm = Invoke-Api -Method Post -Path "/api/v1/auth/password-reset/confirm" -ArtifactName "03-auth-reset-confirm.json" -Body @{
    token = $state.reset_token
    new_password = $state.customer_new_password
  }
  $loginOldPassword = Login-User -Email $state.customer_email -Password $state.customer_password -ArtifactName "03-auth-login-old-password.json"
  $loginNewPassword = Login-User -Email $state.customer_email -Password $state.customer_new_password -ArtifactName "03-auth-login-new-password.json"
  $state.customer_tokens = $loginNewPassword.response_json.data.tokens
  $resetReuse = Invoke-Api -Method Post -Path "/api/v1/auth/password-reset/confirm" -ArtifactName "03-auth-reset-reuse.json" -Body @{
    token = $state.reset_token
    new_password = "AnotherSecure123!"
  }
  $auth3Ok = (
    $resetRequest.status_code -eq 200 -and
    $resetConfirm.status_code -eq 200 -and
    $loginOldPassword.status_code -eq 401 -and
    $loginNewPassword.status_code -eq 200 -and
    (@(400, 404) -contains $resetReuse.status_code)
  )
  Record-Case -Id "TC_API_AUTH_003" -Priority "Р’С‹СЃРѕРєРёР№" -Name "Р’РѕСЃСЃС‚Р°РЅРѕРІР»РµРЅРёРµ РїР°СЂРѕР»СЏ С‡РµСЂРµР· Р·Р°РїСЂРѕСЃ Рё РїРѕРґС‚РІРµСЂР¶РґРµРЅРёРµ РѕРґРЅРѕСЂР°Р·РѕРІРѕРіРѕ С‚РѕРєРµРЅР°" `
    -Precondition "Email РїРѕР»СЊР·РѕРІР°С‚РµР»СЏ СѓР¶Рµ РїРѕРґС‚РІРµСЂР¶РґРµРЅ." `
    -Steps @("POST /api/v1/auth/password-reset/request", "РџРѕР»СѓС‡РµРЅРёРµ reset_token РёР· email_jobs", "POST /api/v1/auth/password-reset/confirm", "POST /api/v1/auth/login СЃС‚Р°СЂС‹Рј РїР°СЂРѕР»РµРј", "POST /api/v1/auth/login РЅРѕРІС‹Рј РїР°СЂРѕР»РµРј", "РџРѕРІС‚РѕСЂРЅС‹Р№ POST /api/v1/auth/password-reset/confirm С‚РµРј Р¶Рµ token") `
    -TestData @{ email = $state.customer_email; old_password = $state.customer_password; new_password = $state.customer_new_password; reset_token = $state.reset_token } `
    -Expected "request 200; confirm 200; СЃС‚Р°СЂС‹Р№ РїР°СЂРѕР»СЊ РґР°РµС‚ 401; РЅРѕРІС‹Р№ РїР°СЂРѕР»СЊ РґР°РµС‚ 200; reuse token РґР°РµС‚ 400/404." `
    -Actual "reset request -> $($resetRequest.status_code); reset confirm -> $($resetConfirm.status_code); login old -> $($loginOldPassword.status_code); login new -> $($loginNewPassword.status_code); reset reuse -> $($resetReuse.status_code)." `
    -Status $(if ($auth3Ok) { "Р—Р°С‡РµС‚" } else { "РќРµР·Р°С‡РµС‚" }) `
    -Evidence @(
      "POST /api/v1/auth/password-reset/request -> $($resetRequest.status_code), Р°СЂС‚РµС„Р°РєС‚: $($resetRequest.artifact_path)",
      "email_jobs reset token=$($state.reset_token), Р°СЂС‚РµС„Р°РєС‚: $(Join-Path $rawPath '03-auth-reset-email-job.json')",
      "POST /api/v1/auth/password-reset/confirm -> $($resetConfirm.status_code), Р°СЂС‚РµС„Р°РєС‚: $($resetConfirm.artifact_path)",
      "POST /api/v1/auth/login old -> $($loginOldPassword.status_code), Р°СЂС‚РµС„Р°РєС‚: $($loginOldPassword.artifact_path)",
      "POST /api/v1/auth/login new -> $($loginNewPassword.status_code), Р°СЂС‚РµС„Р°РєС‚: $($loginNewPassword.artifact_path)",
      "POST /api/v1/auth/password-reset/confirm reuse -> $($resetReuse.status_code), Р°СЂС‚РµС„Р°РєС‚: $($resetReuse.artifact_path)"
    )

  $catalogPage1 = Invoke-Api -Method Get -Path "/api/v1/products?q=headphones&min_price=100&max_price=1000&in_stock=true&sort=price_asc&page=1&limit=20" -ArtifactName "04-catalog-page1.json"
  $catalogPage2 = Invoke-Api -Method Get -Path "/api/v1/products?q=headphones&min_price=100&max_price=1000&in_stock=true&sort=price_asc&page=2&limit=20" -ArtifactName "04-catalog-page2.json"
  $catalogItems = @($catalogPage1.response_json.data.items)
  $previousPrice = $null
  $allFiltered = $true
  $sortedAsc = $true
  foreach ($item in $catalogItems) {
    if (-not ($item.price -ge 100 -and $item.price -le 1000 -and $item.stock_qty -gt 0 -and (($item.name + " " + $item.description) -match "(?i)headphones"))) {
      $allFiltered = $false
    }
    if ($null -ne $previousPrice -and [decimal]$item.price -lt [decimal]$previousPrice) {
      $sortedAsc = $false
    }
    $previousPrice = [decimal]$item.price
  }
  $catalogOk = (
    $catalogPage1.status_code -eq 200 -and
    $catalogPage1.response_json.data.page -eq 1 -and
    $catalogPage1.response_json.data.limit -eq 20 -and
    $catalogPage1.response_json.data.total -ge $catalogItems.Count -and
    $catalogPage2.status_code -eq 200 -and
    $allFiltered -and
    $sortedAsc
  )
  Record-Case -Id "TC_API_CAT_001" -Priority "Р’С‹СЃРѕРєРёР№" -Name "РџРѕР»СѓС‡РµРЅРёРµ РєР°С‚Р°Р»РѕРіР° С‚РѕРІР°СЂРѕРІ СЃ РїРѕРёСЃРєРѕРј, С†РµРЅРѕРІС‹РјРё С„РёР»СЊС‚СЂР°РјРё, СЃРѕСЂС‚РёСЂРѕРІРєРѕР№ Рё РїР°РіРёРЅР°С†РёРµР№" `
    -Precondition "РљР°С‚Р°Р»РѕРі Р·Р°РіСЂСѓР¶РµРЅ РёР· РјРёРіСЂР°С†РёР№ Рё РґРѕСЃС‚СѓРїРµРЅ С‡РµСЂРµР· GET /api/v1/products." `
    -Steps @("GET /api/v1/products ... page=1", "РџСЂРѕРІРµСЂРєР° С„РёР»СЊС‚СЂРѕРІ/СЃРѕСЂС‚РёСЂРѕРІРєРё", "GET /api/v1/products ... page=2") `
    -TestData @{ query = "q=headphones&min_price=100&max_price=1000&in_stock=true&sort=price_asc&page=1&limit=20" } `
    -Expected "200 OK; РІСЃРµ СЌР»РµРјРµРЅС‚С‹ СЃРѕРѕС‚РІРµС‚СЃС‚РІСѓСЋС‚ С„РёР»СЊС‚СЂР°Рј; price_asc СЃРѕР±Р»СЋРґРµРЅ; РјРµС‚Р°РґР°РЅРЅС‹Рµ page/limit/total РІР°Р»РёРґРЅС‹." `
    -Actual "page1 -> $($catalogPage1.status_code), items=$($catalogItems.Count), page=$($catalogPage1.response_json.data.page), limit=$($catalogPage1.response_json.data.limit), total=$($catalogPage1.response_json.data.total), filters_ok=$allFiltered, sorted_ok=$sortedAsc; page2 -> $($catalogPage2.status_code), page=$($catalogPage2.response_json.data.page), items=$(@($catalogPage2.response_json.data.items).Count)." `
    -Status $(if ($catalogOk) { "Р—Р°С‡РµС‚" } else { "РќРµР·Р°С‡РµС‚" }) `
    -Evidence @(
      "GET /api/v1/products page=1 -> $($catalogPage1.status_code), Р°СЂС‚РµС„Р°РєС‚: $($catalogPage1.artifact_path)",
      "GET /api/v1/products page=2 -> $($catalogPage2.status_code), Р°СЂС‚РµС„Р°РєС‚: $($catalogPage2.artifact_path)"
    )

  $authHeader = Get-AuthHeader -AccessToken $state.customer_tokens.access_token
  $favAdd = Invoke-Api -Method Post -Path "/api/v1/favorites/$($state.product.id)" -Headers $authHeader -ArtifactName "05-favorites-add.json"
  $favList = Invoke-Api -Method Get -Path "/api/v1/favorites?page=1&limit=20" -Headers $authHeader -ArtifactName "05-favorites-list.json"
  $favDelete = Invoke-Api -Method Delete -Path "/api/v1/favorites/$($state.product.id)" -Headers $authHeader -ArtifactName "05-favorites-delete.json"
  $favListAfterDelete = Invoke-Api -Method Get -Path "/api/v1/favorites?page=1&limit=20" -Headers $authHeader -ArtifactName "05-favorites-list-after-delete.json"
  $favoriteFound = @($favList.response_json.data.items.id) -contains $state.product.id
  $favoriteStillFound = @($favListAfterDelete.response_json.data.items.id) -contains $state.product.id
  $favOk = ($favAdd.status_code -eq 200 -and $favoriteFound -and $favDelete.status_code -eq 200 -and -not $favoriteStillFound)
  Record-Case -Id "TC_API_FAV_001" -Priority "РЎСЂРµРґРЅРёР№" -Name "Р”РѕР±Р°РІР»РµРЅРёРµ С‚РѕРІР°СЂР° РІ РёР·Р±СЂР°РЅРЅРѕРµ Рё РїРѕСЃР»РµРґСѓСЋС‰РµРµ СѓРґР°Р»РµРЅРёРµ РёР· СЃРїРёСЃРєР° favorites" `
    -Precondition "РџРѕРґС‚РІРµСЂР¶РґРµРЅРЅС‹Р№ РїРѕР»СЊР·РѕРІР°С‚РµР»СЊ Р°РІС‚РѕСЂРёР·РѕРІР°РЅ РїРѕ bearer token." `
    -Steps @("POST /api/v1/favorites/{product_id}", "GET /api/v1/favorites", "DELETE /api/v1/favorites/{product_id}", "GET /api/v1/favorites") `
    -TestData @{ product_id = $state.product.id } `
    -Expected "POST 200; С‚РѕРІР°СЂ РІРёРґРµРЅ РІ favorites; DELETE 200; РїРѕСЃР»Рµ СѓРґР°Р»РµРЅРёСЏ С‚РѕРІР°СЂР° РЅРµС‚ РІ СЃРїРёСЃРєРµ." `
    -Actual "favorites add -> $($favAdd.status_code); GET after add -> $($favList.status_code), present=$favoriteFound; DELETE -> $($favDelete.status_code); GET after delete -> $($favListAfterDelete.status_code), present=$favoriteStillFound." `
    -Status $(if ($favOk) { "Р—Р°С‡РµС‚" } else { "РќРµР·Р°С‡РµС‚" }) `
    -Evidence @(
      "POST /api/v1/favorites/$($state.product.id) -> $($favAdd.status_code), Р°СЂС‚РµС„Р°РєС‚: $($favAdd.artifact_path)",
      "GET /api/v1/favorites -> $($favList.status_code), Р°СЂС‚РµС„Р°РєС‚: $($favList.artifact_path)",
      "DELETE /api/v1/favorites/$($state.product.id) -> $($favDelete.status_code), Р°СЂС‚РµС„Р°РєС‚: $($favDelete.artifact_path)",
      "GET /api/v1/favorites after delete -> $($favListAfterDelete.status_code), Р°СЂС‚РµС„Р°РєС‚: $($favListAfterDelete.artifact_path)"
    )

  $placeCreate = Invoke-Api -Method Post -Path "/api/v1/places" -Headers $authHeader -ArtifactName "06-places-create.json" -Body @{
    title = "Home"
    address_text = "Moscow, Tverskaya 1"
    lat = 55.7558
    lon = 37.6173
  }
  $state.place_crud_id = $placeCreate.response_json.data.id
  $placeUpdate = Invoke-Api -Method Patch -Path "/api/v1/places/$($state.place_crud_id)" -Headers $authHeader -ArtifactName "06-places-update.json" -Body @{
    title = "QA Home"
    address_text = "Moscow, Tverskaya 1, apt 2"
  }
  $placeList = Invoke-Api -Method Get -Path "/api/v1/places" -Headers $authHeader -ArtifactName "06-places-list.json"
  $placeDelete = Invoke-Api -Method Delete -Path "/api/v1/places/$($state.place_crud_id)" -Headers $authHeader -ArtifactName "06-places-delete.json"
  $placeListAfterDelete = Invoke-Api -Method Get -Path "/api/v1/places" -Headers $authHeader -ArtifactName "06-places-list-after-delete.json"
  $updatedPlace = @($placeList.response_json.data | Where-Object { $_.id -eq $state.place_crud_id }) | Select-Object -First 1
  $placeStillPresent = @($placeListAfterDelete.response_json.data.id) -contains $state.place_crud_id
  $placeOk = ($placeCreate.status_code -eq 201 -and $placeUpdate.status_code -eq 200 -and $updatedPlace.title -eq "QA Home" -and $placeDelete.status_code -eq 200 -and -not $placeStillPresent)
  Record-Case -Id "TC_API_PLACE_001" -Priority "РЎСЂРµРґРЅРёР№" -Name "РЎРѕР·РґР°РЅРёРµ, СЂРµРґР°РєС‚РёСЂРѕРІР°РЅРёРµ Рё СѓРґР°Р»РµРЅРёРµ Р°РґСЂРµСЃР° РїРѕР»СЊР·РѕРІР°С‚РµР»СЏ (places)" `
    -Precondition "РџРѕРґС‚РІРµСЂР¶РґРµРЅРЅС‹Р№ РїРѕР»СЊР·РѕРІР°С‚РµР»СЊ Р°РІС‚РѕСЂРёР·РѕРІР°РЅ." `
    -Steps @("POST /api/v1/places", "PATCH /api/v1/places/{id}", "GET /api/v1/places", "DELETE /api/v1/places/{id}", "GET /api/v1/places") `
    -TestData @{ place_id = $state.place_crud_id } `
    -Expected "РЎРѕР·РґР°РЅРёРµ 201; update 200 Рё СЃРѕС…СЂР°РЅСЏРµС‚ РЅРѕРІС‹Рµ РїРѕР»СЏ; delete 200; РїРѕСЃР»Рµ СѓРґР°Р»РµРЅРёСЏ Р°РґСЂРµСЃР° РЅРµС‚ РІ СЃРїРёСЃРєРµ." `
    -Actual "create -> $($placeCreate.status_code), id=$($state.place_crud_id); update -> $($placeUpdate.status_code); list after update -> title='$($updatedPlace.title)'; delete -> $($placeDelete.status_code); list after delete -> present=$placeStillPresent." `
    -Status $(if ($placeOk) { "Р—Р°С‡РµС‚" } else { "РќРµР·Р°С‡РµС‚" }) `
    -Evidence @(
      "POST /api/v1/places -> $($placeCreate.status_code), Р°СЂС‚РµС„Р°РєС‚: $($placeCreate.artifact_path)",
      "PATCH /api/v1/places/$($state.place_crud_id) -> $($placeUpdate.status_code), Р°СЂС‚РµС„Р°РєС‚: $($placeUpdate.artifact_path)",
      "GET /api/v1/places -> $($placeList.status_code), Р°СЂС‚РµС„Р°РєС‚: $($placeList.artifact_path)",
      "DELETE /api/v1/places/$($state.place_crud_id) -> $($placeDelete.status_code), Р°СЂС‚РµС„Р°РєС‚: $($placeDelete.artifact_path)",
      "GET /api/v1/places after delete -> $($placeListAfterDelete.status_code), Р°СЂС‚РµС„Р°РєС‚: $($placeListAfterDelete.artifact_path)"
    )

  $cartClear = Invoke-Api -Method Delete -Path "/api/v1/cart" -Headers $authHeader -ArtifactName "07-cart-clear.json"
  $cartAdd = Invoke-Api -Method Post -Path "/api/v1/cart/items" -Headers $authHeader -ArtifactName "07-cart-add.json" -Body @{ product_id = $state.product.id; quantity = 2 }
  $cartGet1 = Invoke-Api -Method Get -Path "/api/v1/cart" -Headers $authHeader -ArtifactName "07-cart-get-1.json"
  $cartPatch = Invoke-Api -Method Patch -Path "/api/v1/cart/items/$($state.product.id)" -Headers $authHeader -ArtifactName "07-cart-patch.json" -Body @{ quantity = 4 }
  $cartGet2 = Invoke-Api -Method Get -Path "/api/v1/cart" -Headers $authHeader -ArtifactName "07-cart-get-2.json"
  $cartDelete = Invoke-Api -Method Delete -Path "/api/v1/cart/items/$($state.product.id)" -Headers $authHeader -ArtifactName "07-cart-delete-item.json"
  $cartGet3 = Invoke-Api -Method Get -Path "/api/v1/cart" -Headers $authHeader -ArtifactName "07-cart-get-3.json"
  $cartItem1 = @($cartGet1.response_json.data.items | Where-Object { $_.product_id -eq $state.product.id }) | Select-Object -First 1
  $cartItem2 = @($cartGet2.response_json.data.items | Where-Object { $_.product_id -eq $state.product.id }) | Select-Object -First 1
  $cartEmpty = (@($cartGet3.response_json.data.items).Count -eq 0)
  $cartOk = ($cartClear.status_code -eq 200 -and $cartAdd.status_code -eq 201 -and $cartItem1.quantity -eq 2 -and $cartPatch.status_code -eq 200 -and $cartItem2.quantity -eq 4 -and $cartDelete.status_code -eq 200 -and $cartEmpty)
  Record-Case -Id "TC_API_CART_001" -Priority "Р’С‹СЃРѕРєРёР№" -Name "Р”РѕР±Р°РІР»РµРЅРёРµ С‚РѕРІР°СЂР° РІ РєРѕСЂР·РёРЅСѓ, РёР·РјРµРЅРµРЅРёРµ РєРѕР»РёС‡РµСЃС‚РІР° Рё СѓРґР°Р»РµРЅРёРµ РїРѕР·РёС†РёРё" `
    -Precondition "Р•СЃС‚СЊ Р°РєС‚РёРІРЅС‹Р№ С‚РѕРІР°СЂ $($state.product.id) СЃ stock_qty=$($state.product.stock_qty)." `
    -Steps @("DELETE /api/v1/cart", "POST /api/v1/cart/items", "GET /api/v1/cart", "PATCH /api/v1/cart/items/{product_id}", "GET /api/v1/cart", "DELETE /api/v1/cart/items/{product_id}", "GET /api/v1/cart") `
    -TestData @{ product_id = $state.product.id; add_quantity = 2; update_quantity = 4 } `
    -Expected "РћС‡РёСЃС‚РєР° 200; РґРѕР±Р°РІР»РµРЅРёРµ 201; PATCH РјРµРЅСЏРµС‚ quantity Рё СЃСѓРјРјС‹; РїРѕСЃР»Рµ СѓРґР°Р»РµРЅРёСЏ РєРѕСЂР·РёРЅР° РїСѓСЃС‚Р°СЏ." `
    -Actual "clear -> $($cartClear.status_code); add -> $($cartAdd.status_code); get1 quantity=$($cartItem1.quantity), total_items=$($cartGet1.response_json.data.total_items), total_amount=$($cartGet1.response_json.data.total_amount); patch -> $($cartPatch.status_code); get2 quantity=$($cartItem2.quantity), total_items=$($cartGet2.response_json.data.total_items), total_amount=$($cartGet2.response_json.data.total_amount); delete -> $($cartDelete.status_code); get3 items=$(@($cartGet3.response_json.data.items).Count)." `
    -Status $(if ($cartOk) { "Р—Р°С‡РµС‚" } else { "РќРµР·Р°С‡РµС‚" }) `
    -Evidence @(
      "DELETE /api/v1/cart -> $($cartClear.status_code), Р°СЂС‚РµС„Р°РєС‚: $($cartClear.artifact_path)",
      "POST /api/v1/cart/items -> $($cartAdd.status_code), Р°СЂС‚РµС„Р°РєС‚: $($cartAdd.artifact_path)",
      "GET /api/v1/cart after add -> $($cartGet1.status_code), Р°СЂС‚РµС„Р°РєС‚: $($cartGet1.artifact_path)",
      "PATCH /api/v1/cart/items/$($state.product.id) -> $($cartPatch.status_code), Р°СЂС‚РµС„Р°РєС‚: $($cartPatch.artifact_path)",
      "GET /api/v1/cart after patch -> $($cartGet2.status_code), Р°СЂС‚РµС„Р°РєС‚: $($cartGet2.artifact_path)",
      "DELETE /api/v1/cart/items/$($state.product.id) -> $($cartDelete.status_code), Р°СЂС‚РµС„Р°РєС‚: $($cartDelete.artifact_path)",
      "GET /api/v1/cart after delete -> $($cartGet3.status_code), Р°СЂС‚РµС„Р°РєС‚: $($cartGet3.artifact_path)"
    )

  $orderPlaceCreate = Invoke-Api -Method Post -Path "/api/v1/places" -Headers $authHeader -ArtifactName "08-order-place-create.json" -Body @{
    title = "Order Place"
    address_text = "Moscow, Arbat 10"
    lat = 55.7522
    lon = 37.5924
  }
  $state.order_place_id = $orderPlaceCreate.response_json.data.id
  $cartAddOrder = Invoke-Api -Method Post -Path "/api/v1/cart/items" -Headers $authHeader -ArtifactName "08-order-cart-add.json" -Body @{ product_id = $state.product.id; quantity = 2 }
  $orderCreate = Invoke-Api -Method Post -Path "/api/v1/orders" -Headers $authHeader -ArtifactName "08-order-create.json" -Body @{ place_id = $state.order_place_id }
  $state.order_id = $orderCreate.response_json.data.id
  $ordersList = Invoke-Api -Method Get -Path "/api/v1/orders?page=1&limit=20" -Headers $authHeader -ArtifactName "08-order-list.json"
  $orderGet = Invoke-Api -Method Get -Path "/api/v1/orders/$($state.order_id)" -Headers $authHeader -ArtifactName "08-order-get.json"
  $orderInList = @($ordersList.response_json.data.items.id) -contains $state.order_id
  $orderOk = ($orderPlaceCreate.status_code -eq 201 -and $cartAddOrder.status_code -eq 201 -and $orderCreate.status_code -eq 201 -and $ordersList.status_code -eq 200 -and $orderInList -and $orderGet.status_code -eq 200 -and $orderGet.response_json.data.place_id -eq $state.order_place_id)
  Record-Case -Id "TC_API_ORDER_001" -Priority "Р’С‹СЃРѕРєРёР№" -Name "РћС„РѕСЂРјР»РµРЅРёРµ Р·Р°РєР°Р·Р° РёР· РЅРµРїСѓСЃС‚РѕР№ РєРѕСЂР·РёРЅС‹ Рё РїСЂРѕРІРµСЂРєР° РїРѕСЏРІР»РµРЅРёСЏ Р·Р°РєР°Р·Р° РІ РёСЃС‚РѕСЂРёРё" `
    -Precondition "РџРѕРґРіРѕС‚РѕРІР»РµРЅ РѕС‚РґРµР»СЊРЅС‹Р№ place_id Рё РєРѕСЂР·РёРЅР° СЃ С‚РѕРІР°СЂРѕРј." `
    -Steps @("POST /api/v1/places", "POST /api/v1/cart/items", "POST /api/v1/orders", "GET /api/v1/orders", "GET /api/v1/orders/{id}") `
    -TestData @{ place_id = $state.order_place_id; product_id = $state.product.id; quantity = 2 } `
    -Expected "Checkout 201; РІ РѕС‚РІРµС‚Рµ РµСЃС‚СЊ order_id; Р·Р°РєР°Р· РІРёРґРµРЅ РІ РёСЃС‚РѕСЂРёРё; GET /orders/{id} РІРѕР·РІСЂР°С‰Р°РµС‚ РєРѕСЂСЂРµРєС‚РЅС‹Рµ РґРµС‚Р°Р»Рё." `
    -Actual "place create -> $($orderPlaceCreate.status_code), place_id=$($state.order_place_id); cart add -> $($cartAddOrder.status_code); order create -> $($orderCreate.status_code), order_id=$($state.order_id); orders list -> $($ordersList.status_code), in_list=$orderInList; order get -> $($orderGet.status_code), items=$(@($orderGet.response_json.data.items).Count), total_amount=$($orderGet.response_json.data.total_amount)." `
    -Status $(if ($orderOk) { "Р—Р°С‡РµС‚" } else { "РќРµР·Р°С‡РµС‚" }) `
    -Evidence @(
      "POST /api/v1/places -> $($orderPlaceCreate.status_code), Р°СЂС‚РµС„Р°РєС‚: $($orderPlaceCreate.artifact_path)",
      "POST /api/v1/cart/items -> $($cartAddOrder.status_code), Р°СЂС‚РµС„Р°РєС‚: $($cartAddOrder.artifact_path)",
      "POST /api/v1/orders -> $($orderCreate.status_code), Р°СЂС‚РµС„Р°РєС‚: $($orderCreate.artifact_path)",
      "GET /api/v1/orders -> $($ordersList.status_code), Р°СЂС‚РµС„Р°РєС‚: $($ordersList.artifact_path)",
      "GET /api/v1/orders/$($state.order_id) -> $($orderGet.status_code), Р°СЂС‚РµС„Р°РєС‚: $($orderGet.artifact_path)"
    )

  $loginSecondSession = Login-User -Email $state.customer_email -Password $state.customer_new_password -ArtifactName "09-auth-login-second-session.json"
  $state.customer_tokens_2 = $loginSecondSession.response_json.data.tokens
  $sessionsListBefore = Invoke-Api -Method Get -Path "/api/v1/auth/sessions" -Headers $authHeader -ArtifactName "09-sessions-list-before.json"
  $sessions = @($sessionsListBefore.response_json.data)
  $secondary = @($sessions | Where-Object { -not $_.is_current }) | Select-Object -First 1
  if ($null -eq $secondary) {
    $secondary = $sessions | Select-Object -Last 1
  }
  $state.secondary_session_id = $secondary.id
  $sessionDelete = Invoke-Api -Method Delete -Path "/api/v1/auth/sessions/$($state.secondary_session_id)" -Headers $authHeader -ArtifactName "09-sessions-delete.json"
  $sessionsListAfter = Invoke-Api -Method Get -Path "/api/v1/auth/sessions" -Headers $authHeader -ArtifactName "09-sessions-list-after.json"
  $refreshRevoked = Invoke-Api -Method Post -Path "/api/v1/auth/refresh" -ArtifactName "09-sessions-refresh-revoked.json" -Body @{
    refresh_token = $state.customer_tokens_2.refresh_token
  }
  $sessionStillPresent = @($sessionsListAfter.response_json.data.id) -contains $state.secondary_session_id
  $sessionOk = ($loginSecondSession.status_code -eq 200 -and @($sessions).Count -ge 2 -and $sessionDelete.status_code -eq 200 -and -not $sessionStillPresent -and $refreshRevoked.status_code -eq 401)
  Record-Case -Id "TC_API_SESS_001" -Priority "Р’С‹СЃРѕРєРёР№" -Name "РџСЂРѕСЃРјРѕС‚СЂ Р°РєС‚РёРІРЅС‹С… СЃРµСЃСЃРёР№ Рё РѕС‚Р·С‹РІ РєРѕРЅРєСЂРµС‚РЅРѕР№ refresh-СЃРµСЃСЃРёРё РїРѕР»СЊР·РѕРІР°С‚РµР»СЏ" `
    -Precondition "РЎРѕР·РґР°РЅС‹ РґРІРµ Р°РєС‚РёРІРЅС‹Рµ bearer-СЃРµСЃСЃРёРё РѕРґРЅРѕРіРѕ РїРѕР»СЊР·РѕРІР°С‚РµР»СЏ." `
    -Steps @("POST /api/v1/auth/login РґР»СЏ РІС‚РѕСЂРѕР№ СЃРµСЃСЃРёРё", "GET /api/v1/auth/sessions", "DELETE /api/v1/auth/sessions/{id}", "GET /api/v1/auth/sessions", "POST /api/v1/auth/refresh c revoked refresh_token") `
    -TestData @{ session_count_before = @($sessions).Count; revoked_session_id = $state.secondary_session_id } `
    -Expected "РЎРїРёСЃРѕРє СЃРµСЃСЃРёР№ 200 Рё >=2 Р·Р°РїРёСЃРё; СѓРґР°Р»РµРЅРёРµ 200; revoked session РѕС‚СЃСѓС‚СЃС‚РІСѓРµС‚ РІ СЃРїРёСЃРєРµ; refresh РµСЋ Р·Р°РІРµСЂС€Р°РµС‚СЃСЏ 401." `
    -Actual "second login -> $($loginSecondSession.status_code); sessions before -> $($sessionsListBefore.status_code), count=$(@($sessions).Count); delete -> $($sessionDelete.status_code), revoked_session_id=$($state.secondary_session_id); sessions after -> $($sessionsListAfter.status_code), present=$sessionStillPresent; refresh revoked -> $($refreshRevoked.status_code)." `
    -Status $(if ($sessionOk) { "Р—Р°С‡РµС‚" } else { "РќРµР·Р°С‡РµС‚" }) `
    -Evidence @(
      "POST /api/v1/auth/login second session -> $($loginSecondSession.status_code), Р°СЂС‚РµС„Р°РєС‚: $($loginSecondSession.artifact_path)",
      "GET /api/v1/auth/sessions -> $($sessionsListBefore.status_code), Р°СЂС‚РµС„Р°РєС‚: $($sessionsListBefore.artifact_path)",
      "DELETE /api/v1/auth/sessions/$($state.secondary_session_id) -> $($sessionDelete.status_code), Р°СЂС‚РµС„Р°РєС‚: $($sessionDelete.artifact_path)",
      "GET /api/v1/auth/sessions after delete -> $($sessionsListAfter.status_code), Р°СЂС‚РµС„Р°РєС‚: $($sessionsListAfter.artifact_path)",
      "POST /api/v1/auth/refresh revoked -> $($refreshRevoked.status_code), Р°СЂС‚РµС„Р°РєС‚: $($refreshRevoked.artifact_path)"
    )

  $sellerLogin = Login-User -Email "merchant-urbanwave@seed.marketplace.local" -Password "Seller123" -ArtifactName "10-seller-login.json"
  $state.seller_tokens = $sellerLogin.response_json.data.tokens
  $sellerHeader = Get-AuthHeader -AccessToken $state.seller_tokens.access_token
  $sellerDashboard = Invoke-Api -Method Get -Path "/api/v1/seller/dashboard" -Headers $sellerHeader -ArtifactName "10-seller-dashboard.json"
  $sellerPayload = @{
    category_id = $state.seller_category_id
    name = "QA Seller Product $($state.run_id)"
    slug = "qa-seller-product-$($state.run_id)"
    description = "Temporary QA seller product"
    price = 1234
    currency = "RUB"
    sku = "QA-SKU-$($state.run_id)"
    image_url = "marketplace-media://product/qa-seller-product-$($state.run_id)/hero"
    images = @("marketplace-media://product/qa-seller-product-$($state.run_id)/hero")
    brand = "UrbanWave"
    unit = "piece"
    specs = @{ qa = "true"; run_id = $state.run_id }
    stock_qty = 7
    is_active = $true
  }
  $sellerCreate = Invoke-Api -Method Post -Path "/api/v1/seller/products" -Headers $sellerHeader -ArtifactName "10-seller-create-product.json" -Body $sellerPayload
  $state.seller_product_id = $sellerCreate.response_json.data.id
  $sellerPatch = Invoke-Api -Method Patch -Path "/api/v1/seller/products/$($state.seller_product_id)/stock" -Headers $sellerHeader -ArtifactName "10-seller-patch-stock.json" -Body @{ stock_qty = 11 }
  $sellerList = Invoke-Api -Method Get -Path "/api/v1/seller/products?page=1&limit=20&q=QA%20Seller%20Product" -Headers $sellerHeader -ArtifactName "10-seller-products-list.json"
  $sellerDelete = Invoke-Api -Method Delete -Path "/api/v1/seller/products/$($state.seller_product_id)" -Headers $sellerHeader -ArtifactName "10-seller-delete-product.json"
  $listedProduct = @($sellerList.response_json.data.items | Where-Object { $_.id -eq $state.seller_product_id }) | Select-Object -First 1
  $sellerOk = ($sellerLogin.status_code -eq 200 -and $sellerDashboard.status_code -eq 200 -and $sellerCreate.status_code -eq 201 -and $sellerPatch.status_code -eq 200 -and $listedProduct.stock_qty -eq 11 -and $sellerDelete.status_code -eq 200)
  Record-Case -Id "TC_API_SELL_001" -Priority "РЎСЂРµРґРЅРёР№" -Name "Р”РѕСЃС‚СѓРї РїСЂРѕРґР°РІС†Р° Рє seller dashboard Рё Р±Р°Р·РѕРІРѕРµ СѓРїСЂР°РІР»РµРЅРёРµ СЃРѕР±СЃС‚РІРµРЅРЅС‹Рј С‚РѕРІР°СЂРѕРј" `
    -Precondition "РСЃРїРѕР»СЊР·СѓРµС‚СЃСЏ seeded seller user СЃ Р°РєС‚РёРІРЅС‹Рј seller profile." `
    -Steps @("POST /api/v1/auth/login", "GET /api/v1/seller/dashboard", "POST /api/v1/seller/products", "PATCH /api/v1/seller/products/{id}/stock", "GET /api/v1/seller/products", "DELETE /api/v1/seller/products/{id}") `
    -TestData @{ seller_email = "merchant-urbanwave@seed.marketplace.local"; category_id = $state.seller_category_id; seller_product_id = $state.seller_product_id } `
    -Expected "Dashboard 200; СЃРѕР·РґР°РЅРёРµ С‚РѕРІР°СЂР° 201; update stock 200; С‚РѕРІР°СЂ РІРёРґРµРЅ РІ seller list СЃ РЅРѕРІС‹Рј stock_qty; Р°СЂС…РёРІР°С†РёСЏ 200." `
    -Actual "seller login -> $($sellerLogin.status_code); dashboard -> $($sellerDashboard.status_code); create -> $($sellerCreate.status_code), product_id=$($state.seller_product_id); stock patch -> $($sellerPatch.status_code), stock_qty=$($sellerPatch.response_json.data.stock_qty); seller list -> $($sellerList.status_code), listed_stock=$($listedProduct.stock_qty); delete -> $($sellerDelete.status_code)." `
    -Status $(if ($sellerOk) { "Р—Р°С‡РµС‚" } else { "РќРµР·Р°С‡РµС‚" }) `
    -Evidence @(
      "POST /api/v1/auth/login seller -> $($sellerLogin.status_code), Р°СЂС‚РµС„Р°РєС‚: $($sellerLogin.artifact_path)",
      "GET /api/v1/seller/dashboard -> $($sellerDashboard.status_code), Р°СЂС‚РµС„Р°РєС‚: $($sellerDashboard.artifact_path)",
      "POST /api/v1/seller/products -> $($sellerCreate.status_code), Р°СЂС‚РµС„Р°РєС‚: $($sellerCreate.artifact_path)",
      "PATCH /api/v1/seller/products/$($state.seller_product_id)/stock -> $($sellerPatch.status_code), Р°СЂС‚РµС„Р°РєС‚: $($sellerPatch.artifact_path)",
      "GET /api/v1/seller/products -> $($sellerList.status_code), Р°СЂС‚РµС„Р°РєС‚: $($sellerList.artifact_path)",
      "DELETE /api/v1/seller/products/$($state.seller_product_id) -> $($sellerDelete.status_code), Р°СЂС‚РµС„Р°РєС‚: $($sellerDelete.artifact_path)"
    )

  $null = Invoke-Api -Method Delete -Path "/api/v1/favorites/$($state.product.id)" -Headers $authHeader -ArtifactName "99-cleanup-favorites.json"
  $null = Invoke-Api -Method Delete -Path "/api/v1/cart" -Headers $authHeader -ArtifactName "99-cleanup-cart.json"
  if ($state.seller_product_id) {
    $null = Invoke-Api -Method Delete -Path "/api/v1/seller/products/$($state.seller_product_id)" -Headers $sellerHeader -ArtifactName "99-cleanup-seller-product.json"
  }
  if ($state.secondary_session_id) {
    $null = Invoke-Api -Method Delete -Path "/api/v1/auth/sessions/$($state.secondary_session_id)" -Headers $authHeader -ArtifactName "99-cleanup-session.json"
  }

  $reportFiles = Write-Reports
  [PSCustomObject]@{
    base_url = $state.base_url
    results_json = $reportFiles.json
    report_md = $reportFiles.md
  } | ConvertTo-Json -Depth 10
}
catch {
  $fatal = [ordered]@{
    message = $_.Exception.Message
    stack = $_.ScriptStackTrace
  }
  Save-JsonArtifact -Name "fatal-error.json" -Data $fatal | Out-Null
  throw
}
finally {
  if ($script:Client) {
    $script:Client.Dispose()
  }
}
