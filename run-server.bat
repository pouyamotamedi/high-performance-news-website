@echo off
echo Starting High Performance News Website...
echo.

REM Check if Go is installed
go version >nul 2>&1
if %errorlevel% neq 0 (
    echo ❌ Go is not installed or not in PATH
    echo Please install Go from: https://golang.org/dl/
    echo After installation, restart your terminal and try again.
    pause
    exit /b 1
)

echo ✅ Go is available
echo.

REM Download dependencies first
echo Downloading dependencies...
go mod download
if %errorlevel% neq 0 (
    echo ❌ Failed to download dependencies
    pause
    exit /b 1
)

REM Run the server
echo Starting development server...
echo Server will be available at: http://localhost:8080
echo Press Ctrl+C to stop the server
echo.
go run cmd/server/main.go

pause