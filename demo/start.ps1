# PowerShell script to start multiple demo servers
# Usage: .\start_demo_servers.ps1

Write-Host "Building demo server..." -ForegroundColor Green
go build -o demo_server.exe demo_server.go

if ($LASTEXITCODE -ne 0) {
    Write-Host "Failed to build demo server" -ForegroundColor Red
    exit 1
}

Write-Host "Starting demo servers..." -ForegroundColor Green

# Start Server 1
Start-Process -FilePath ".\demo_server.exe" -ArgumentList "-id", "1", "-port", "8001" -NoNewWindow
# Start Server 2
Start-Process -FilePath ".\demo_server.exe" -ArgumentList "-id", "2", "-port", "8002" -NoNewWindow
# Start Server 3
Start-Process -FilePath ".\demo_server.exe" -ArgumentList "-id", "3", "-port", "8003" -NoNewWindow

Write-Host "`nDemo servers are running!" -ForegroundColor Green

# Keep script running
try {
    while ($true) {
        Start-Sleep -Seconds 1
    }
} finally {
    Write-Host "`nStopping demo servers..." -ForegroundColor Yellow
    Get-Process -Name "demo_server" -ErrorAction SilentlyContinue | Stop-Process -Force
    Write-Host "Demo servers stopped" -ForegroundColor Green
}