# Performance Benchmark Script for app-go (PowerShell)
# Tests cache performance and gzip compression

Write-Host "🚀 app-go Performance Benchmark" -ForegroundColor Cyan
Write-Host "================================" -ForegroundColor Cyan
Write-Host ""

# Check if server is running
Write-Host "Checking if server is running... " -NoNewline
try {
    $null = Invoke-WebRequest -Uri "http://localhost:8080/health" -UseBasicParsing -TimeoutSec 2
    Write-Host "OK" -ForegroundColor Green
} catch {
    Write-Host "FAILED" -ForegroundColor Red
    Write-Host "Please start the server first:" -ForegroundColor Yellow
    Write-Host "  docker run -d -p 8080:8080 --name hello-world-go hello-world-go:latest"
    exit 1
}
Write-Host ""

# Test 1: Cache Performance (Cold vs Warm)
Write-Host "Test 1: Cache Performance" -ForegroundColor Blue
Write-Host "------------------------"

$user = "SKRTEEEEEE"
$endpoint = "http://localhost:8080/issues/$user"

Write-Host "Cold request (no cache): " -NoNewline
$coldStart = Get-Date
$null = Invoke-WebRequest -Uri $endpoint -UseBasicParsing
$coldEnd = Get-Date
$coldTime = ($coldEnd - $coldStart).TotalSeconds
Write-Host "$([math]::Round($coldTime, 3))s" -ForegroundColor Yellow

Start-Sleep -Seconds 1

Write-Host "Warm request (cached):   " -NoNewline
$warmStart = Get-Date
$null = Invoke-WebRequest -Uri $endpoint -UseBasicParsing
$warmEnd = Get-Date
$warmTime = ($warmEnd - $warmStart).TotalSeconds
Write-Host "$([math]::Round($warmTime, 3))s" -ForegroundColor Green

$speedup = [math]::Round($coldTime / $warmTime, 2)
Write-Host "Speedup: " -NoNewline
Write-Host "${speedup}x faster" -ForegroundColor Green
Write-Host ""

# Test 2: Gzip Compression
Write-Host "Test 2: Gzip Compression" -ForegroundColor Blue
Write-Host "------------------------"

Write-Host "Without gzip: " -NoNewline
$responseUncompressed = Invoke-WebRequest -Uri $endpoint -UseBasicParsing
$sizeUncompressed = $responseUncompressed.Content.Length
Write-Host "$sizeUncompressed bytes" -ForegroundColor Yellow

Write-Host "With gzip:    " -NoNewline
$responseCompressed = Invoke-WebRequest -Uri $endpoint -UseBasicParsing -Headers @{"Accept-Encoding"="gzip"}
$sizeCompressed = $responseCompressed.Content.Length
Write-Host "$sizeCompressed bytes" -ForegroundColor Green

$savings = [math]::Round(100 - ($sizeCompressed * 100 / $sizeUncompressed), 1)
Write-Host "Savings: " -NoNewline
Write-Host "$savings% reduction" -ForegroundColor Green
Write-Host ""

# Test 3: Multiple Requests (Cache Hit Rate)
Write-Host "Test 3: Cache Hit Rate (10 requests)" -ForegroundColor Blue
Write-Host "-------------------------------------"

$times = @()
for ($i = 1; $i -le 10; $i++) {
    $start = Get-Date
    $null = Invoke-WebRequest -Uri $endpoint -UseBasicParsing
    $end = Get-Date
    $time = ($end - $start).TotalSeconds
    $times += $time
    Write-Host "Request $($i.ToString().PadLeft(2)): $([math]::Round($time, 4))s"
}

$avgTime = ($times | Measure-Object -Average).Average
Write-Host "Average: " -NoNewline
Write-Host "$([math]::Round($avgTime, 4))s" -ForegroundColor Green
Write-Host ""

# Test 4: Health Check Performance
Write-Host "Test 4: Health Check Performance" -ForegroundColor Blue
Write-Host "---------------------------------"

$healthTimes = @()
for ($i = 1; $i -le 5; $i++) {
    $start = Get-Date
    $null = Invoke-WebRequest -Uri "http://localhost:8080/health" -UseBasicParsing
    $end = Get-Date
    $healthTimes += ($end - $start).TotalSeconds
}

$healthAvg = ($healthTimes | Measure-Object -Average).Average
Write-Host "Average health check time: " -NoNewline
Write-Host "$([math]::Round($healthAvg, 4))s" -ForegroundColor Green
Write-Host ""

# Summary
Write-Host "================================" -ForegroundColor Cyan
Write-Host "✓ Benchmark Complete!" -ForegroundColor Green
Write-Host ""
Write-Host "Key Findings:"
Write-Host "  • Cache provides ${speedup}x speedup"
Write-Host "  • Gzip reduces response size by $savings%"
Write-Host "  • Average cached response time: $([math]::Round($avgTime, 4))s"
Write-Host "  • Health check latency: $([math]::Round($healthAvg, 4))s"
Write-Host ""
Write-Host "For detailed performance documentation, see PERFORMANCE.md"
