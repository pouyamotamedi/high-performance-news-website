# Snyk Dependency Vulnerability Scanning Script (PowerShell)
# This script automates dependency vulnerability scanning using Snyk

param(
    [string]$ProjectPath = ".",
    [string]$ReportDir = "./reports/security/snyk",
    [string]$SnykToken = $env:SNYK_TOKEN,
    [string]$SeverityThreshold = "medium",
    [switch]$MonitorProject,
    [switch]$FixVulnerabilities,
    [switch]$JsonOutput,
    [switch]$Quiet,
    [switch]$Help
)

# Configuration
$SnykConfigFile = ".snyk"
$SupportedSeverities = @("low", "medium", "high", "critical")

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
Snyk Dependency Vulnerability Scanning Script

USAGE:
    .\snyk-scan.ps1 [OPTIONS]

OPTIONS:
    -ProjectPath PATH       Path to project directory (default: current directory)
    -ReportDir DIR          Report output directory (default: ./reports/security/snyk)
    -SnykToken TOKEN        Snyk authentication token (default: SNYK_TOKEN env var)
    -SeverityThreshold LEVEL Minimum severity to report (low|medium|high|critical)
    -MonitorProject         Add project to Snyk monitoring
    -FixVulnerabilities     Attempt to automatically fix vulnerabilities
    -JsonOutput             Output results in JSON format
    -Quiet                  Quiet mode
    -Help                   Show this help message

EXAMPLES:
    .\snyk-scan.ps1                                    # Scan current directory
    .\snyk-scan.ps1 -SeverityThreshold "high"         # Only report high/critical issues
    .\snyk-scan.ps1 -MonitorProject -JsonOutput       # Monitor project and output JSON
    .\snyk-scan.ps1 -FixVulnerabilities               # Scan and attempt to fix issues

ENVIRONMENT VARIABLES:
    SNYK_TOKEN              Snyk authentication token (required)

"@
}

# Check if Snyk is installed
function Test-SnykInstallation {
    Write-Info "Checking Snyk CLI installation..."
    
    try {
        $SnykVersion = & snyk --version 2>$null
        if ($LASTEXITCODE -eq 0) {
            Write-Success "Snyk CLI found: $SnykVersion"
            return $true
        }
    }
    catch {
        # Continue to installation check
    }
    
    Write-Error "Snyk CLI not found. Please install Snyk CLI:"
    Write-Host "  npm install -g snyk" -ForegroundColor $Colors.Yellow
    Write-Host "  or download from: https://github.com/snyk/snyk/releases" -ForegroundColor $Colors.Yellow
    return $false
}

# Authenticate with Snyk
function Initialize-SnykAuth {
    if (-not $SnykToken) {
        Write-Error "Snyk token not provided. Set SNYK_TOKEN environment variable or use -SnykToken parameter"
        return $false
    }
    
    Write-Info "Authenticating with Snyk..."
    
    try {
        $env:SNYK_TOKEN = $SnykToken
        $AuthResult = & snyk auth $SnykToken 2>&1
        
        if ($LASTEXITCODE -eq 0) {
            Write-Success "Successfully authenticated with Snyk"
            return $true
        }
        else {
            Write-Error "Failed to authenticate with Snyk: $AuthResult"
            return $false
        }
    }
    catch {
        Write-Error "Failed to authenticate with Snyk: $($_.Exception.Message)"
        return $false
    }
}

# Test project dependencies for vulnerabilities
function Test-ProjectVulnerabilities {
    Write-Info "Scanning project dependencies for vulnerabilities..."
    
    # Create report directory
    if (-not (Test-Path $ReportDir)) {
        New-Item -ItemType Directory -Path $ReportDir -Force | Out-Null
    }
    
    $Timestamp = Get-Date -Format "yyyyMMdd-HHmmss"
    $ReportFile = "$ReportDir\snyk-test-$Timestamp"
    
    # Build Snyk test command
    $SnykArgs = @("test")
    
    # Add severity threshold
    if ($SeverityThreshold -in $SupportedSeverities) {
        $SnykArgs += "--severity-threshold=$SeverityThreshold"
    }
    
    # Add JSON output if requested
    if ($JsonOutput) {
        $SnykArgs += "--json"
        $ReportFile += ".json"
    }
    else {
        $ReportFile += ".txt"
    }
    
    # Add project path
    $SnykArgs += "--file=go.mod"
    
    # Show vulnerable paths
    $SnykArgs += "--show-vulnerable-paths=all"
    
    try {
        Write-Info "Running Snyk test with arguments: $($SnykArgs -join ' ')"
        
        # Change to project directory
        Push-Location $ProjectPath
        
        # Run Snyk test
        $TestOutput = & snyk @SnykArgs 2>&1
        $ExitCode = $LASTEXITCODE
        
        # Save output to report file
        $TestOutput | Out-File -FilePath $ReportFile -Encoding UTF8
        
        # Parse results
        if ($JsonOutput -and $ExitCode -ne 0) {
            try {
                $JsonResult = $TestOutput | ConvertFrom-Json
                $VulnCount = if ($JsonResult.vulnerabilities) { $JsonResult.vulnerabilities.Count } else { 0 }
                $UniqueVulns = if ($JsonResult.uniqueCount) { $JsonResult.uniqueCount } else { 0 }
                
                Write-Info "Vulnerabilities found: $VulnCount (unique: $UniqueVulns)"
                
                # Create summary
                $Summary = @{
                    timestamp = Get-Date -Format "yyyy-MM-ddTHH:mm:ssZ"
                    projectPath = $ProjectPath
                    totalVulnerabilities = $VulnCount
                    uniqueVulnerabilities = $UniqueVulns
                    severityThreshold = $SeverityThreshold
                    reportFile = $ReportFile
                    exitCode = $ExitCode
                }
                
                $Summary | ConvertTo-Json | Out-File -FilePath "$ReportDir\snyk-summary-$Timestamp.json" -Encoding UTF8
            }
            catch {
                Write-Warning "Could not parse JSON output: $($_.Exception.Message)"
            }
        }
        else {
            # Parse text output for summary
            $VulnLines = $TestOutput | Where-Object { $_ -match "✗.*vulnerabilities?" }
            if ($VulnLines) {
                Write-Info "Scan completed. Check report for details: $ReportFile"
            }
        }
        
        # Interpret exit codes
        switch ($ExitCode) {
            0 {
                Write-Success "No vulnerabilities found above $SeverityThreshold severity threshold"
                return @{ Success = $true; VulnerabilitiesFound = $false; ExitCode = $ExitCode }
            }
            1 {
                Write-Warning "Vulnerabilities found above $SeverityThreshold severity threshold"
                return @{ Success = $true; VulnerabilitiesFound = $true; ExitCode = $ExitCode }
            }
            2 {
                Write-Error "Snyk test failed due to an error"
                return @{ Success = $false; VulnerabilitiesFound = $false; ExitCode = $ExitCode }
            }
            default {
                Write-Error "Snyk test completed with unexpected exit code: $ExitCode"
                return @{ Success = $false; VulnerabilitiesFound = $false; ExitCode = $ExitCode }
            }
        }
    }
    catch {
        Write-Error "Failed to run Snyk test: $($_.Exception.Message)"
        return @{ Success = $false; VulnerabilitiesFound = $false; ExitCode = -1 }
    }
    finally {
        Pop-Location
    }
}

# Monitor project with Snyk
function Add-ProjectMonitoring {
    Write-Info "Adding project to Snyk monitoring..."
    
    try {
        Push-Location $ProjectPath
        
        $MonitorArgs = @("monitor", "--file=go.mod")
        $MonitorOutput = & snyk @MonitorArgs 2>&1
        $ExitCode = $LASTEXITCODE
        
        if ($ExitCode -eq 0) {
            Write-Success "Project added to Snyk monitoring"
            
            # Extract project URL if available
            $ProjectUrl = $MonitorOutput | Where-Object { $_ -match "https://app.snyk.io" }
            if ($ProjectUrl) {
                Write-Info "Monitor project at: $ProjectUrl"
            }
            
            return $true
        }
        else {
            Write-Error "Failed to add project to monitoring: $MonitorOutput"
            return $false
        }
    }
    catch {
        Write-Error "Failed to monitor project: $($_.Exception.Message)"
        return $false
    }
    finally {
        Pop-Location
    }
}

# Attempt to fix vulnerabilities
function Repair-Vulnerabilities {
    Write-Info "Attempting to fix vulnerabilities..."
    
    try {
        Push-Location $ProjectPath
        
        # Check if go.mod exists
        if (-not (Test-Path "go.mod")) {
            Write-Error "go.mod file not found in project directory"
            return $false
        }
        
        # For Go projects, we'll use go get to update dependencies
        Write-Info "Updating Go dependencies to latest versions..."
        
        # Get list of direct dependencies
        $GoListOutput = & go list -m -json all 2>&1
        if ($LASTEXITCODE -ne 0) {
            Write-Error "Failed to list Go modules: $GoListOutput"
            return $false
        }
        
        # Parse go list output and update dependencies
        $Modules = $GoListOutput | ConvertFrom-Json
        $UpdatedCount = 0
        
        foreach ($Module in $Modules) {
            if ($Module.Main -eq $true -or $Module.Indirect -eq $true) {
                continue
            }
            
            try {
                Write-Info "Updating module: $($Module.Path)"
                $UpdateOutput = & go get -u $Module.Path 2>&1
                
                if ($LASTEXITCODE -eq 0) {
                    $UpdatedCount++
                    Write-Success "Updated: $($Module.Path)"
                }
                else {
                    Write-Warning "Could not update $($Module.Path): $UpdateOutput"
                }
            }
            catch {
                Write-Warning "Error updating $($Module.Path): $($_.Exception.Message)"
            }
        }
        
        if ($UpdatedCount -gt 0) {
            # Tidy up go.mod
            Write-Info "Tidying go.mod..."
            & go mod tidy 2>&1 | Out-Null
            
            Write-Success "Updated $UpdatedCount dependencies"
            Write-Info "Please run tests to ensure updates don't break functionality"
            return $true
        }
        else {
            Write-Info "No dependencies were updated"
            return $true
        }
    }
    catch {
        Write-Error "Failed to fix vulnerabilities: $($_.Exception.Message)"
        return $false
    }
    finally {
        Pop-Location
    }
}

# Generate comprehensive security report
function New-SecurityReport {
    param(
        [hashtable]$TestResult
    )
    
    $Timestamp = Get-Date -Format "yyyyMMdd-HHmmss"
    $ReportPath = "$ReportDir\snyk-security-report-$Timestamp.html"
    
    Write-Info "Generating comprehensive security report..."
    
    $HtmlContent = @"
<!DOCTYPE html>
<html>
<head>
    <title>Snyk Security Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .header { background-color: #f0f0f0; padding: 20px; border-radius: 5px; margin-bottom: 20px; }
        .summary { background-color: #e8f5e8; padding: 15px; border-radius: 5px; margin-bottom: 20px; }
        .vulnerabilities { background-color: #ffe8e8; padding: 15px; border-radius: 5px; margin-bottom: 20px; }
        .success { background-color: #e8f5e8; }
        .warning { background-color: #fff3cd; }
        .error { background-color: #f8d7da; }
        .info { background-color: #d1ecf1; }
        table { width: 100%; border-collapse: collapse; margin-top: 10px; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #f2f2f2; }
        .severity-critical { color: #721c24; font-weight: bold; }
        .severity-high { color: #856404; font-weight: bold; }
        .severity-medium { color: #0c5460; }
        .severity-low { color: #155724; }
    </style>
</head>
<body>
    <div class="header">
        <h1>Snyk Security Scan Report</h1>
        <p><strong>Project:</strong> $ProjectPath</p>
        <p><strong>Scan Date:</strong> $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')</p>
        <p><strong>Severity Threshold:</strong> $SeverityThreshold</p>
        <p><strong>Scan Status:</strong> $(if ($TestResult.Success) { 'Completed' } else { 'Failed' })</p>
    </div>
    
    <div class="summary $(if ($TestResult.VulnerabilitiesFound) { 'warning' } else { 'success' })">
        <h2>Summary</h2>
        <p><strong>Vulnerabilities Found:</strong> $(if ($TestResult.VulnerabilitiesFound) { 'Yes' } else { 'No' })</p>
        <p><strong>Exit Code:</strong> $($TestResult.ExitCode)</p>
        <p><strong>Report Directory:</strong> $ReportDir</p>
    </div>
    
    <div class="info">
        <h2>Recommendations</h2>
        <ul>
            <li>Review the detailed Snyk report files in the reports directory</li>
            <li>Update vulnerable dependencies to secure versions</li>
            <li>Consider using <code>.\snyk-scan.ps1 -FixVulnerabilities</code> to automatically update dependencies</li>
            <li>Add the project to Snyk monitoring with <code>.\snyk-scan.ps1 -MonitorProject</code></li>
            <li>Regularly scan dependencies as part of your CI/CD pipeline</li>
        </ul>
    </div>
    
    <div class="info">
        <h2>Next Steps</h2>
        <ol>
            <li>Review all identified vulnerabilities in the detailed reports</li>
            <li>Prioritize fixes based on severity and exploitability</li>
            <li>Test application functionality after updating dependencies</li>
            <li>Set up continuous monitoring for new vulnerabilities</li>
            <li>Integrate Snyk scanning into your development workflow</li>
        </ol>
    </div>
    
    <div class="info">
        <h2>Files Generated</h2>
        <ul>
"@

    # List generated files
    $ReportFiles = Get-ChildItem -Path $ReportDir -Filter "snyk-*" | Sort-Object LastWriteTime -Descending | Select-Object -First 10
    foreach ($File in $ReportFiles) {
        $HtmlContent += "            <li><strong>$($File.Name)</strong> - $(Get-Date $File.LastWriteTime -Format 'yyyy-MM-dd HH:mm:ss')</li>`n"
    }

    $HtmlContent += @"
        </ul>
    </div>
</body>
</html>
"@

    try {
        $HtmlContent | Out-File -FilePath $ReportPath -Encoding UTF8
        Write-Success "Security report generated: $ReportPath"
        return $ReportPath
    }
    catch {
        Write-Error "Failed to generate security report: $($_.Exception.Message)"
        return $null
    }
}

# Main execution function
function Start-SnykScan {
    Write-Info "Starting Snyk dependency vulnerability scan..."
    
    # Validate parameters
    if ($SeverityThreshold -notin $SupportedSeverities) {
        Write-Error "Invalid severity threshold: $SeverityThreshold. Supported values: $($SupportedSeverities -join ', ')"
        return 1
    }
    
    # Check if project directory exists
    if (-not (Test-Path $ProjectPath)) {
        Write-Error "Project path does not exist: $ProjectPath"
        return 1
    }
    
    # Check for go.mod file
    $GoModPath = Join-Path $ProjectPath "go.mod"
    if (-not (Test-Path $GoModPath)) {
        Write-Error "go.mod file not found in project directory: $ProjectPath"
        return 1
    }
    
    # Check Snyk installation
    if (-not (Test-SnykInstallation)) {
        return 1
    }
    
    # Authenticate with Snyk
    if (-not (Initialize-SnykAuth)) {
        return 1
    }
    
    # Create report directory
    if (-not (Test-Path $ReportDir)) {
        New-Item -ItemType Directory -Path $ReportDir -Force | Out-Null
    }
    
    # Fix vulnerabilities if requested (before testing)
    if ($FixVulnerabilities) {
        if (-not (Repair-Vulnerabilities)) {
            Write-Warning "Failed to fix some vulnerabilities, continuing with scan..."
        }
    }
    
    # Test for vulnerabilities
    $TestResult = Test-ProjectVulnerabilities
    
    if (-not $TestResult.Success) {
        Write-Error "Vulnerability scan failed"
        return 2
    }
    
    # Monitor project if requested
    if ($MonitorProject) {
        if (-not (Add-ProjectMonitoring)) {
            Write-Warning "Failed to add project to monitoring, but scan completed successfully"
        }
    }
    
    # Generate comprehensive report
    $ReportPath = New-SecurityReport -TestResult $TestResult
    
    # Return appropriate exit code
    if ($TestResult.VulnerabilitiesFound) {
        Write-Warning "Vulnerabilities found. Review the reports and take appropriate action."
        return 1
    }
    else {
        Write-Success "No vulnerabilities found above the specified threshold."
        return 0
    }
}

# Main script execution
if ($Help) {
    Show-Usage
    exit 0
}

# Run the Snyk scan
$ExitCode = Start-SnykScan
exit $ExitCode