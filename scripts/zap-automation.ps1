# OWASP ZAP Automation Script for Security Testing (PowerShell)
# This script automates the setup and execution of OWASP ZAP security scans

param(
    [string]$TargetUrl = "http://localhost:8080",
    [int]$ZapPort = 8080,
    [string]$ApiKey = "",
    [string]$ReportDir = "./reports/security/zap",
    [switch]$AutoInstall,
    [switch]$SpiderOnly,
    [switch]$ActiveOnly,
    [switch]$Quiet,
    [switch]$Help
)

# Configuration
$ZapVersion = "2.14.0"
$ConfigFile = "./configs/owasp-zap-config.yaml"

# Colors for output
$Colors = @{
    Red = "Red"
    Green = "Green"
    Yellow = "Yellow"
    Blue = "Blue"
    White = "White"
}

# Logging functions
function Write-Info {
    param([string]$Message)
    if (-not $Quiet) {
        Write-Host "[INFO] $Message" -ForegroundColor $Colors.Blue
    }
}

function Write-Success {
    param([string]$Message)
    if (-not $Quiet) {
        Write-Host "[SUCCESS] $Message" -ForegroundColor $Colors.Green
    }
}

function Write-Warning {
    param([string]$Message)
    Write-Host "[WARNING] $Message" -ForegroundColor $Colors.Yellow
}

function Write-Error {
    param([string]$Message)
    Write-Host "[ERROR] $Message" -ForegroundColor $Colors.Red
}

# Show usage information
function Show-Usage {
    Write-Host @"
OWASP ZAP Automation Script for Security Testing

USAGE:
    .\zap-automation.ps1 [OPTIONS]

OPTIONS:
    -TargetUrl URL          Target URL to scan (default: http://localhost:8080)
    -ZapPort PORT           ZAP daemon port (default: 8080)
    -ApiKey KEY             ZAP API key (default: auto-generated)
    -ReportDir DIR          Report output directory (default: ./reports/security/zap)
    -AutoInstall            Auto-install ZAP if not found
    -SpiderOnly             Run spider scan only
    -ActiveOnly             Run active scan only (requires previous spider)
    -Quiet                  Quiet mode
    -Help                   Show this help message

EXAMPLES:
    .\zap-automation.ps1                                    # Scan localhost:8080
    .\zap-automation.ps1 -TargetUrl "https://example.com"   # Scan specific URL
    .\zap-automation.ps1 -AutoInstall -TargetUrl "https://staging.example.com"

"@
}

# Check if ZAP is installed
function Test-ZapInstallation {
    Write-Info "Checking OWASP ZAP installation..."
    
    # Check common installation paths
    $ZapPaths = @(
        "${env:ProgramFiles}\OWASP\Zap\zap.bat",
        "${env:ProgramFiles(x86)}\OWASP\Zap\zap.bat",
        ".\zap\zap.bat",
        "C:\Program Files\OWASP\Zap\zap.bat"
    )
    
    foreach ($Path in $ZapPaths) {
        if (Test-Path $Path) {
            Write-Success "OWASP ZAP found at: $Path"
            $script:ZapCmd = $Path
            return $true
        }
    }
    
    # Check if zap is in PATH
    try {
        $null = Get-Command "zap.bat" -ErrorAction Stop
        Write-Success "OWASP ZAP found in PATH"
        $script:ZapCmd = "zap.bat"
        return $true
    }
    catch {
        Write-Error "OWASP ZAP not found. Please install ZAP or use -AutoInstall flag."
        return $false
    }
}

# Download and install ZAP
function Install-Zap {
    Write-Info "Installing OWASP ZAP..."
    
    $ZapDir = ".\zap"
    $ZapDownloadUrl = "https://github.com/zaproxy/zaproxy/releases/download/v$ZapVersion/ZAP_${ZapVersion}_windows.exe"
    $ZapInstaller = "$ZapDir\zap-installer.exe"
    
    # Create zap directory
    if (-not (Test-Path $ZapDir)) {
        New-Item -ItemType Directory -Path $ZapDir -Force | Out-Null
    }
    
    try {
        Write-Info "Downloading ZAP installer..."
        Invoke-WebRequest -Uri $ZapDownloadUrl -OutFile $ZapInstaller -UseBasicParsing
        
        Write-Info "Installing ZAP (this may take a few minutes)..."
        Start-Process -FilePath $ZapInstaller -ArgumentList "/S", "/D=$((Get-Location).Path)\zap\installation" -Wait
        
        $script:ZapCmd = ".\zap\installation\zap.bat"
        
        if (Test-Path $script:ZapCmd) {
            Write-Success "OWASP ZAP installed successfully"
            return $true
        }
        else {
            Write-Error "ZAP installation failed"
            return $false
        }
    }
    catch {
        Write-Error "Failed to download or install ZAP: $($_.Exception.Message)"
        return $false
    }
}

# Start ZAP daemon
function Start-ZapDaemon {
    Write-Info "Starting OWASP ZAP daemon..."
    
    # Generate API key if not provided
    if (-not $ApiKey) {
        $ApiKey = [System.Web.Security.Membership]::GeneratePassword(32, 0)
        $script:ApiKey = $ApiKey
    }
    
    # Check if ZAP is already running
    try {
        $Response = Invoke-RestMethod -Uri "http://localhost:$ZapPort/JSON/core/view/version/" -Method Get -TimeoutSec 5
        Write-Warning "ZAP daemon is already running on port $ZapPort"
        return $true
    }
    catch {
        # ZAP is not running, continue with startup
    }
    
    # Create report directory
    if (-not (Test-Path $ReportDir)) {
        New-Item -ItemType Directory -Path $ReportDir -Force | Out-Null
    }
    
    # Start ZAP in daemon mode
    $ZapArgs = @(
        "-daemon",
        "-port", $ZapPort,
        "-config", "api.key=$ApiKey",
        "-config", "api.addrs.addr.name=*",
        "-config", "api.addrs.addr.regex=true",
        "-config", "spider.maxDepth=5",
        "-config", "spider.maxChildren=10",
        "-config", "scanner.maxRuleDurationInMins=5",
        "-config", "scanner.maxScanDurationInMins=30"
    )
    
    $ZapProcess = Start-Process -FilePath $script:ZapCmd -ArgumentList $ZapArgs -PassThru -WindowStyle Hidden
    $ZapProcess.Id | Out-File -FilePath "$ReportDir\zap.pid" -Encoding ASCII
    
    # Wait for ZAP to start
    Write-Info "Waiting for ZAP daemon to start..."
    $MaxAttempts = 30
    $Attempt = 0
    
    do {
        Start-Sleep -Seconds 2
        $Attempt++
        
        try {
            $Response = Invoke-RestMethod -Uri "http://localhost:$ZapPort/JSON/core/view/version/?apikey=$ApiKey" -Method Get -TimeoutSec 5
            Write-Success "ZAP daemon started successfully on port $ZapPort"
            return $true
        }
        catch {
            # Continue waiting
        }
    } while ($Attempt -lt $MaxAttempts)
    
    Write-Error "Failed to start ZAP daemon"
    return $false
}

# Stop ZAP daemon
function Stop-ZapDaemon {
    Write-Info "Stopping OWASP ZAP daemon..."
    
    $PidFile = "$ReportDir\zap.pid"
    
    if (Test-Path $PidFile) {
        try {
            $ZapPid = Get-Content $PidFile
            $ZapProcess = Get-Process -Id $ZapPid -ErrorAction SilentlyContinue
            
            if ($ZapProcess) {
                $ZapProcess.Kill()
                Write-Success "ZAP daemon stopped"
            }
            else {
                Write-Warning "ZAP daemon was not running"
            }
            
            Remove-Item $PidFile -Force -ErrorAction SilentlyContinue
        }
        catch {
            Write-Warning "Could not stop ZAP daemon via PID"
        }
    }
    
    # Try to stop via API
    try {
        Invoke-RestMethod -Uri "http://localhost:$ZapPort/JSON/core/action/shutdown/?apikey=$ApiKey" -Method Get -TimeoutSec 5 | Out-Null
        Write-Info "Sent shutdown command to ZAP daemon"
    }
    catch {
        # Ignore errors
    }
}

# Create new ZAP session
function New-ZapSession {
    $SessionName = "security-scan-$(Get-Date -Format 'yyyyMMdd-HHmmss')"
    
    Write-Info "Creating ZAP session: $SessionName"
    
    try {
        $Response = Invoke-RestMethod -Uri "http://localhost:$ZapPort/JSON/core/action/newSession/?apikey=$ApiKey&name=$SessionName" -Method Get
        Write-Success "ZAP session created: $SessionName"
        return $SessionName
    }
    catch {
        Write-Error "Failed to create ZAP session: $($_.Exception.Message)"
        return $null
    }
}

# Configure ZAP context
function Set-ZapContext {
    $ContextName = "news-website-context"
    
    Write-Info "Configuring ZAP context: $ContextName"
    
    try {
        # Create context
        Invoke-RestMethod -Uri "http://localhost:$ZapPort/JSON/context/action/newContext/?apikey=$ApiKey&contextName=$ContextName" -Method Get | Out-Null
        
        # Include URLs in context
        $EncodedUrl = [System.Web.HttpUtility]::UrlEncode("$TargetUrl.*")
        Invoke-RestMethod -Uri "http://localhost:$ZapPort/JSON/context/action/includeInContext/?apikey=$ApiKey&contextName=$ContextName&regex=$EncodedUrl" -Method Get | Out-Null
        
        # Exclude static resources
        $StaticUrl = [System.Web.HttpUtility]::UrlEncode("$TargetUrl/static/.*")
        Invoke-RestMethod -Uri "http://localhost:$ZapPort/JSON/context/action/excludeFromContext/?apikey=$ApiKey&contextName=$ContextName&regex=$StaticUrl" -Method Get | Out-Null
        
        $FaviconUrl = [System.Web.HttpUtility]::UrlEncode("$TargetUrl/favicon.ico")
        Invoke-RestMethod -Uri "http://localhost:$ZapPort/JSON/context/action/excludeFromContext/?apikey=$ApiKey&contextName=$ContextName&regex=$FaviconUrl" -Method Get | Out-Null
        
        Write-Success "ZAP context configured"
        return $true
    }
    catch {
        Write-Error "Failed to configure ZAP context: $($_.Exception.Message)"
        return $false
    }
}

# Run ZAP spider
function Start-ZapSpider {
    Write-Info "Starting ZAP spider scan on $TargetUrl"
    
    try {
        $Response = Invoke-RestMethod -Uri "http://localhost:$ZapPort/JSON/spider/action/scan/?apikey=$ApiKey&url=$TargetUrl&maxChildren=10&recurse=true&contextName=news-website-context" -Method Get
        $SpiderId = $Response.scan
        
        if (-not $SpiderId) {
            Write-Error "Failed to start spider scan"
            return $false
        }
        
        Write-Info "Spider scan started with ID: $SpiderId"
        
        # Wait for spider to complete
        do {
            Start-Sleep -Seconds 5
            $StatusResponse = Invoke-RestMethod -Uri "http://localhost:$ZapPort/JSON/spider/view/status/?apikey=$ApiKey&scanId=$SpiderId" -Method Get
            $Status = [int]$StatusResponse.status
            
            Write-Info "Spider progress: $Status%"
        } while ($Status -lt 100)
        
        Write-Success "Spider scan completed"
        
        # Get spider results
        $UrlsResponse = Invoke-RestMethod -Uri "http://localhost:$ZapPort/JSON/core/view/urls/?apikey=$ApiKey" -Method Get
        $UrlsFound = $UrlsResponse.urls.Count
        Write-Success "Spider found $UrlsFound URLs"
        
        return $true
    }
    catch {
        Write-Error "Spider scan failed: $($_.Exception.Message)"
        return $false
    }
}

# Run ZAP active scan
function Start-ZapActiveScan {
    Write-Info "Starting ZAP active scan on $TargetUrl"
    
    try {
        $Response = Invoke-RestMethod -Uri "http://localhost:$ZapPort/JSON/ascan/action/scan/?apikey=$ApiKey&url=$TargetUrl&recurse=true&inScopeOnly=false&scanPolicyName=Default%20Policy&method=GET&postData=" -Method Get
        $ScanId = $Response.scan
        
        if (-not $ScanId) {
            Write-Error "Failed to start active scan"
            return $false
        }
        
        Write-Info "Active scan started with ID: $ScanId"
        
        # Wait for active scan to complete
        do {
            Start-Sleep -Seconds 10
            $StatusResponse = Invoke-RestMethod -Uri "http://localhost:$ZapPort/JSON/ascan/view/status/?apikey=$ApiKey&scanId=$ScanId" -Method Get
            $Status = [int]$StatusResponse.status
            
            Write-Info "Active scan progress: $Status%"
        } while ($Status -lt 100)
        
        Write-Success "Active scan completed"
        return $true
    }
    catch {
        Write-Error "Active scan failed: $($_.Exception.Message)"
        return $false
    }
}

# Generate ZAP reports
function New-ZapReports {
    $Timestamp = Get-Date -Format "yyyyMMdd-HHmmss"
    $ReportBase = "$ReportDir\zap-report-$Timestamp"
    
    Write-Info "Generating ZAP reports..."
    
    try {
        # Generate HTML report
        $HtmlReport = Invoke-RestMethod -Uri "http://localhost:$ZapPort/OTHER/core/other/htmlreport/?apikey=$ApiKey" -Method Get
        $HtmlReport | Out-File -FilePath "$ReportBase.html" -Encoding UTF8
        
        # Generate JSON report
        $JsonReport = Invoke-RestMethod -Uri "http://localhost:$ZapPort/JSON/core/view/alerts/?apikey=$ApiKey" -Method Get
        $JsonReport | ConvertTo-Json -Depth 10 | Out-File -FilePath "$ReportBase.json" -Encoding UTF8
        
        # Generate XML report
        $XmlReport = Invoke-RestMethod -Uri "http://localhost:$ZapPort/OTHER/core/other/xmlreport/?apikey=$ApiKey" -Method Get
        $XmlReport | Out-File -FilePath "$ReportBase.xml" -Encoding UTF8
        
        # Get alert summary
        $AlertsResponse = Invoke-RestMethod -Uri "http://localhost:$ZapPort/JSON/core/view/numberOfAlerts/?apikey=$ApiKey" -Method Get
        $TotalAlerts = $AlertsResponse.numberOfAlerts
        
        $SummaryResponse = Invoke-RestMethod -Uri "http://localhost:$ZapPort/JSON/core/view/alertsSummary/?apikey=$ApiKey" -Method Get
        $HighAlerts = if ($SummaryResponse.alertsSummary.High) { $SummaryResponse.alertsSummary.High } else { 0 }
        $MediumAlerts = if ($SummaryResponse.alertsSummary.Medium) { $SummaryResponse.alertsSummary.Medium } else { 0 }
        $LowAlerts = if ($SummaryResponse.alertsSummary.Low) { $SummaryResponse.alertsSummary.Low } else { 0 }
        
        # Generate summary
        $SummaryContent = @"
OWASP ZAP Security Scan Summary
==============================
Scan Date: $(Get-Date)
Target URL: $TargetUrl
Total Alerts: $TotalAlerts

Alert Breakdown:
- High Risk: $HighAlerts
- Medium Risk: $MediumAlerts
- Low Risk: $LowAlerts

Reports Generated:
- HTML Report: $ReportBase.html
- JSON Report: $ReportBase.json
- XML Report: $ReportBase.xml
"@
        
        $SummaryContent | Out-File -FilePath "$ReportBase-summary.txt" -Encoding UTF8
        
        Write-Success "Reports generated in $ReportDir"
        Write-Info "Total alerts found: $TotalAlerts (High: $HighAlerts, Medium: $MediumAlerts, Low: $LowAlerts)"
        
        # Return exit code based on findings
        if ($HighAlerts -gt 0) {
            Write-Error "High risk vulnerabilities found!"
            return 2
        }
        elseif ($MediumAlerts -gt 5) {
            Write-Warning "Multiple medium risk vulnerabilities found!"
            return 1
        }
        
        return 0
    }
    catch {
        Write-Error "Failed to generate reports: $($_.Exception.Message)"
        return 1
    }
}

# Main execution function
function Start-SecurityScan {
    Write-Info "Starting OWASP ZAP security scan..."
    
    # Check if target is accessible
    try {
        $Response = Invoke-WebRequest -Uri $TargetUrl -Method Head -TimeoutSec 10 -UseBasicParsing
        Write-Success "Target URL $TargetUrl is accessible"
    }
    catch {
        Write-Error "Target URL $TargetUrl is not accessible: $($_.Exception.Message)"
        return 1
    }
    
    # Check ZAP installation
    if (-not (Test-ZapInstallation)) {
        if ($AutoInstall) {
            if (-not (Install-Zap)) {
                return 1
            }
        }
        else {
            Write-Error "Please install OWASP ZAP or use -AutoInstall flag"
            return 1
        }
    }
    
    # Start ZAP daemon
    if (-not (Start-ZapDaemon)) {
        return 1
    }
    
    try {
        # Create session and configure context
        $Session = New-ZapSession
        if (-not $Session) {
            return 1
        }
        
        if (-not (Set-ZapContext)) {
            return 1
        }
        
        # Run scans
        if (-not $ActiveOnly) {
            if (-not (Start-ZapSpider)) {
                return 1
            }
        }
        
        if (-not $SpiderOnly) {
            if (-not (Start-ZapActiveScan)) {
                return 1
            }
        }
        
        # Generate reports
        $ScanResult = New-ZapReports
        
        return $ScanResult
    }
    finally {
        # Always stop ZAP daemon
        Stop-ZapDaemon
    }
}

# Main script execution
if ($Help) {
    Show-Usage
    exit 0
}

# Add required assemblies
Add-Type -AssemblyName System.Web

# Validate parameters
if (-not $TargetUrl.StartsWith("http")) {
    Write-Error "Target URL must start with http:// or https://"
    exit 1
}

# Run the security scan
$ExitCode = Start-SecurityScan
exit $ExitCode