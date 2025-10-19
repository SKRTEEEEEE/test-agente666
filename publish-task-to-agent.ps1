# Script to publish task events directly to NATS AGENT stream
# This simulates what the orchestrator would do

param(
    [Parameter(Mandatory=$false)]
    [string]$TaskId = "task-" + (New-Guid).ToString(),
    
    [Parameter(Mandatory=$false)]
    [string]$IssueId = "16",
    
    [Parameter(Mandatory=$false)]
    [string]$Repository = "/SKRTEEEEEE/test-agente666",
    
    [Parameter(Mandatory=$false)]
    [string]$TaskFilePath = "/SKRTEEEEEE/test-agente666/docs/task/16-task.md",
    
    [Parameter(Mandatory=$false)]
    [int]$SizeBytes = 2048,
    
    [Parameter(Mandatory=$false)]
    [string]$NatsUrl = "nats://localhost:4222"
)

Write-Host "Publishing task to NATS AGENT stream..." -ForegroundColor Cyan
Write-Host "Task ID: $TaskId" -ForegroundColor Yellow
Write-Host "Issue ID: $IssueId" -ForegroundColor Yellow
Write-Host "Repository: $Repository" -ForegroundColor Yellow

$timestamp = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ss.fffZ")

$taskEvent = @{
    task_id = $TaskId
    issue_id = $IssueId
    repository = $Repository
    task_file_path = $TaskFilePath
    size_bytes = $SizeBytes
    created_at = $timestamp
} | ConvertTo-Json -Compress

Write-Host "`nEvent payload:" -ForegroundColor Green
Write-Host $taskEvent

# Use docker exec to publish via NATS CLI (if available in container)
# Alternative: use nats-cli if installed locally

Write-Host "`nAttempting to publish..." -ForegroundColor Cyan

try {
    # Method 1: Try using nats CLI in container (may not exist)
    docker exec agent666-nats sh -c "echo '$taskEvent' | nats pub agent.task.new --server=$NatsUrl" 2>$null
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host "✅ Successfully published to NATS!" -ForegroundColor Green
    } else {
        Write-Host "⚠️  NATS CLI not available in container." -ForegroundColor Yellow
        Write-Host "📝 Please use one of the following methods:" -ForegroundColor Cyan
        Write-Host ""
        Write-Host "METHOD 1: Install nats-cli locally" -ForegroundColor White
        Write-Host "  1. Download from: https://github.com/nats-io/natscli/releases" -ForegroundColor Gray
        Write-Host "  2. Run: nats pub agent.task.new '$taskEvent' --server=$NatsUrl" -ForegroundColor Gray
        Write-Host ""
        Write-Host "METHOD 2: Use the MongoDB direct insertion (see below)" -ForegroundColor White
        Write-Host ""
        Write-Host "METHOD 3: Use MongoDB directly to insert task" -ForegroundColor White
        Write-Host "  docker exec -it agent666-mongodb mongosh agent_intel --eval '" -NoNewline -ForegroundColor Gray
        
        $mongoDoc = @"
db.pending_tasks.insertOne({
  task_id: "$TaskId",
  issue_id: "$IssueId",
  repository: "$Repository",
  task_file_path: "$TaskFilePath",
  created_at: new Date("$timestamp"),
  last_success_at: null,
  avg_runtime_ms: 0,
  pending_tasks_count: 1,
  size_bytes: $SizeBytes,
  status: "pending",
  assigned_at: null
})
"@
        Write-Host $mongoDoc -ForegroundColor Gray
        Write-Host "'" -ForegroundColor Gray
    }
} catch {
    Write-Host "❌ Error: $_" -ForegroundColor Red
}

Write-Host "`n📊 To verify the task was created, run:" -ForegroundColor Cyan
Write-Host "  docker exec agent666-mongodb mongosh agent_intel --eval 'db.pending_tasks.find().pretty()'" -ForegroundColor Gray
