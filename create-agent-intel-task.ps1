# Helper script to create tasks directly in Agent Intel MongoDB
# This bypasses NATS and inserts directly into pending_tasks collection

param(
    [Parameter(Mandatory=$false)]
    [string]$IssueId = "16",
    
    [Parameter(Mandatory=$false)]
    [string]$Repository = "/SKRTEEEEEE/test-agente666",
    
    [Parameter(Mandatory=$false)]
    [string]$TaskFilePath,
    
    [Parameter(Mandatory=$false)]
    [int]$SizeBytes = 2048
)

# Generate task ID if not provided
$TaskId = "task-" + (New-Guid).ToString()

# Generate task file path if not provided
if ([string]::IsNullOrEmpty($TaskFilePath)) {
    $TaskFilePath = "$Repository/docs/task/$IssueId-task.md"
}

$timestamp = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ss.fffZ")

Write-Host "=" * 70 -ForegroundColor Cyan
Write-Host "Creating Task in Agent Intel (MongoDB)" -ForegroundColor Cyan
Write-Host "=" * 70 -ForegroundColor Cyan
Write-Host ""
Write-Host "Task ID:       " -NoNewline -ForegroundColor Yellow
Write-Host $TaskId -ForegroundColor White
Write-Host "Issue ID:      " -NoNewline -ForegroundColor Yellow
Write-Host $IssueId -ForegroundColor White
Write-Host "Repository:    " -NoNewline -ForegroundColor Yellow
Write-Host $Repository -ForegroundColor White
Write-Host "Task File:     " -NoNewline -ForegroundColor Yellow
Write-Host $TaskFilePath -ForegroundColor White
Write-Host "Size:          " -NoNewline -ForegroundColor Yellow
Write-Host "$SizeBytes bytes" -ForegroundColor White
Write-Host "Created At:    " -NoNewline -ForegroundColor Yellow
Write-Host $timestamp -ForegroundColor White
Write-Host ""

# MongoDB insertion command
$mongoCommand = @"
db.pending_tasks.insertOne({
  task_id: '$TaskId',
  issue_id: '$IssueId',
  repository: '$Repository',
  task_file_path: '$TaskFilePath',
  created_at: new Date('$timestamp'),
  last_success_at: null,
  avg_runtime_ms: 0,
  pending_tasks_count: 1,
  size_bytes: $SizeBytes,
  status: 'pending',
  assigned_at: null
})
"@

Write-Host "Executing MongoDB insertion..." -ForegroundColor Cyan

try {
    $result = docker exec agent666-mongodb mongosh agent_intel --quiet --eval $mongoCommand 2>&1
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host "✅ Task created successfully!" -ForegroundColor Green
        Write-Host ""
        Write-Host "MongoDB Result:" -ForegroundColor Gray
        Write-Host $result
        Write-Host ""
        Write-Host "🔍 Verify with:" -ForegroundColor Cyan
        Write-Host "  GET http://localhost:8082/api/v1/queue/status" -ForegroundColor White
        Write-Host "  GET http://localhost:8082/api/v1/queue/next" -ForegroundColor White
        Write-Host ""
        Write-Host "📋 Task ID for testing:" -ForegroundColor Yellow
        Write-Host "  $TaskId" -ForegroundColor White
    } else {
        Write-Host "❌ Failed to create task" -ForegroundColor Red
        Write-Host "Error: $result" -ForegroundColor Red
    }
} catch {
    Write-Host "❌ Error executing MongoDB command: $_" -ForegroundColor Red
}

Write-Host ""
Write-Host "=" * 70 -ForegroundColor Cyan
