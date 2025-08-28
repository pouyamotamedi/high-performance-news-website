@echo off
echo High Performance News Website - Windows Setup
echo ============================================
echo.

REM Check Go installation
echo [1/4] Checking Go installation...
go version >nul 2>&1
if %errorlevel% neq 0 (
    echo ❌ Go is not installed
    echo.
    echo Please install Go first:
    echo 1. Go to https://golang.org/dl/
    echo 2. Download the Windows installer (.msi)
    echo 3. Run the installer
    echo 4. Restart your terminal
    echo 5. Run this script again
    echo.
    pause
    exit /b 1
) else (
    for /f "tokens=*" %%i in ('go version') do set GO_VERSION=%%i
    echo ✅ !GO_VERSION!
)

echo.
echo [2/4] Installing Go dependencies...
go mod download
if %errorlevel% neq 0 (
    echo ❌ Failed to download dependencies
    pause
    exit /b 1
)
echo ✅ Dependencies installed

echo.
echo [3/4] Checking Docker...
docker --version >nul 2>&1
if %errorlevel% neq 0 (
    echo ⚠️  Docker not found - you can still run without Docker
    echo   To use Docker: Install Docker Desktop from https://docs.docker.com/desktop/windows/
) else (
    echo ✅ Docker is available
    echo Starting Docker services...
    docker-compose -f docker-compose.windows.yml up -d
)

echo.
echo [4/4] Setup complete!
echo.
echo To start the development server:
echo   Option 1: Double-click run-server.bat
echo   Option 2: Run: go run cmd/server/main.go
echo.
echo The server will be available at: http://localhost:8080
echo.
pause