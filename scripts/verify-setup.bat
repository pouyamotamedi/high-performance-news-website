@echo off
REM Verification script for High Performance News Website setup (Windows)

echo 🔍 Verifying High Performance News Website setup...
echo.

REM Check Go installation
echo Checking Go installation...
go version >nul 2>&1
if %errorlevel% == 0 (
    for /f "tokens=*" %%i in ('go version') do set GO_VERSION=%%i
    echo ✅ Go is installed: !GO_VERSION!
) else (
    echo ❌ Go is not installed
    echo    Please install Go: https://golang.org/dl/
)
echo.

REM Check Docker installation
echo Checking Docker installation...
docker --version >nul 2>&1
if %errorlevel% == 0 (
    for /f "tokens=*" %%i in ('docker --version') do set DOCKER_VERSION=%%i
    echo ✅ Docker is installed: !DOCKER_VERSION!
) else (
    echo ❌ Docker is not installed
    echo    Please install Docker Desktop: https://docs.docker.com/desktop/windows/
)
echo.

REM Check Docker Compose installation
echo Checking Docker Compose installation...
docker-compose --version >nul 2>&1
if %errorlevel% == 0 (
    for /f "tokens=*" %%i in ('docker-compose --version') do set COMPOSE_VERSION=%%i
    echo ✅ Docker Compose is installed: !COMPOSE_VERSION!
) else (
    docker compose version >nul 2>&1
    if %errorlevel% == 0 (
        for /f "tokens=*" %%i in ('docker compose version') do set COMPOSE_VERSION=%%i
        echo ✅ Docker Compose (plugin) is installed: !COMPOSE_VERSION!
    ) else (
        echo ❌ Docker Compose is not installed
        echo    Usually included with Docker Desktop
    )
)
echo.

REM Check project structure
echo Checking project structure...
set REQUIRED_DIRS=cmd internal pkg web migrations configs
set REQUIRED_FILES=go.mod docker-compose.yml Makefile .air.toml

for %%d in (%REQUIRED_DIRS%) do (
    if exist "%%d" (
        echo ✅ Directory exists: %%d
    ) else (
        echo ❌ Missing directory: %%d
    )
)

for %%f in (%REQUIRED_FILES%) do (
    if exist "%%f" (
        echo ✅ File exists: %%f
    ) else (
        echo ❌ Missing file: %%f
    )
)
echo.

REM Check Go module
echo Checking Go module...
if exist "go.mod" (
    go version >nul 2>&1
    if %errorlevel% == 0 (
        echo Running go mod tidy...
        go mod tidy
        if %errorlevel% == 0 (
            echo ✅ Go module is valid
        ) else (
            echo ❌ Go module has issues
        )
    ) else (
        echo ⚠️  Cannot verify Go module (Go not installed)
    )
) else (
    echo ❌ go.mod file not found
)
echo.

echo 🎉 Setup verification complete!
echo.
echo Next steps:
echo 1. Install any missing dependencies listed above
echo 2. Copy .env.example to .env and configure as needed
echo 3. Run 'make dev-setup' to complete development environment setup
echo 4. Run 'make dev' to start the development server
echo.
echo Note: On Windows, you may need to use PowerShell or WSL for some commands
pause