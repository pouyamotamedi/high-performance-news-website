@echo off
REM Comprehensive Test Runner Script for Windows
REM Implements requirements for >95% coverage, parallel execution, and comprehensive validation

setlocal enabledelayedexpansion

REM Configuration
set COVERAGE_THRESHOLD=95
set TEST_TIMEOUT=30m
set BENCHMARK_TIME=10s
set RESULTS_DIR=test-results

REM Colors (limited support in Windows)
set INFO=[INFO]
set SUCCESS=[SUCCESS]
set WARNING=[WARNING]
set ERROR=[ERROR]

echo %INFO% Starting comprehensive test suite...

REM Create results directory
if not exist "%RESULTS_DIR%" mkdir "%RESULTS_DIR%"

REM Check if Go is available
go version >nul 2>&1
if errorlevel 1 (
    echo %ERROR% Go is not installed or not in PATH
    exit /b 1
)

echo %INFO% Running unit tests with coverage tracking...

REM Run unit tests with coverage
set CGO_ENABLED=1
go test -v -coverprofile="%RESULTS_DIR%\coverage.out" -covermode=atomic -timeout=%TEST_TIMEOUT% ./internal/models/... ./internal/repositories/... ./internal/services/... ./internal/api/... ./internal/auth/... ./internal/validation/... ./pkg/... > "%RESULTS_DIR%\unit-tests.log" 2>&1

if errorlevel 1 (
    echo %ERROR% Unit tests failed
    exit /b 1
)

echo %SUCCESS% Unit tests completed successfully

REM Generate coverage report
go tool cover -html="%RESULTS_DIR%\coverage.out" -o "%RESULTS_DIR%\coverage.html"

REM Extract coverage percentage (simplified for Windows)
go tool cover -func="%RESULTS_DIR%\coverage.out" | findstr "total" > "%RESULTS_DIR%\coverage-temp.txt"

REM Parse coverage (basic implementation)
for /f "tokens=3" %%i in (%RESULTS_DIR%\coverage-temp.txt) do (
    set COVERAGE_RAW=%%i
    set COVERAGE=!COVERAGE_RAW:~0,-1!
)

echo %INFO% Current coverage: !COVERAGE!%%

REM Simple coverage validation (Windows doesn't have bc)
if !COVERAGE! LSS %COVERAGE_THRESHOLD% (
    echo %ERROR% Coverage !COVERAGE!%% is below required %COVERAGE_THRESHOLD%%%
    exit /b 1
) else (
    echo %SUCCESS% Coverage !COVERAGE!%% meets requirement (>=%COVERAGE_THRESHOLD%%%)
)

echo !COVERAGE! > "%RESULTS_DIR%\coverage-percentage.txt"

REM Run integration tests if database is available
echo %INFO% Running integration tests...
go test -v -tags=integration -timeout=%TEST_TIMEOUT% ./internal/integration/... ./internal/repositories/... > "%RESULTS_DIR%\integration-tests.log" 2>&1

if errorlevel 1 (
    echo %WARNING% Integration tests failed or skipped
) else (
    echo %SUCCESS% Integration tests completed successfully
)

REM Run benchmarks
echo %INFO% Running performance benchmarks...
go test -bench=. -benchmem -benchtime=%BENCHMARK_TIME% -timeout=10m ./internal/models/... ./internal/repositories/... ./internal/services/... ./pkg/... > "%RESULTS_DIR%\benchmark.txt" 2>&1

if errorlevel 1 (
    echo %WARNING% Benchmarks failed or incomplete
) else (
    echo %SUCCESS% Benchmarks completed successfully
)

REM Generate report
echo %INFO% Generating comprehensive test report...

echo # Comprehensive Test Report > "%RESULTS_DIR%\test-report.md"
echo. >> "%RESULTS_DIR%\test-report.md"
echo Generated: %date% %time% >> "%RESULTS_DIR%\test-report.md"
echo. >> "%RESULTS_DIR%\test-report.md"
echo ## Summary >> "%RESULTS_DIR%\test-report.md"
echo. >> "%RESULTS_DIR%\test-report.md"
echo - **Coverage**: !COVERAGE!%% >> "%RESULTS_DIR%\test-report.md"

if !COVERAGE! GEQ %COVERAGE_THRESHOLD% (
    echo - **Coverage Status**: ✅ PASSED (>=%COVERAGE_THRESHOLD%%%) >> "%RESULTS_DIR%\test-report.md"
) else (
    echo - **Coverage Status**: ❌ FAILED (^<%COVERAGE_THRESHOLD%%%) >> "%RESULTS_DIR%\test-report.md"
)

echo. >> "%RESULTS_DIR%\test-report.md"
echo ## Files Generated >> "%RESULTS_DIR%\test-report.md"
echo. >> "%RESULTS_DIR%\test-report.md"
echo - Coverage Report: [coverage.html](coverage.html) >> "%RESULTS_DIR%\test-report.md"
echo - Unit Test Log: [unit-tests.log](unit-tests.log) >> "%RESULTS_DIR%\test-report.md"
echo - Integration Test Log: [integration-tests.log](integration-tests.log) >> "%RESULTS_DIR%\test-report.md"
echo - Benchmark Results: [benchmark.txt](benchmark.txt) >> "%RESULTS_DIR%\test-report.md"

echo %SUCCESS% Comprehensive testing completed successfully!
echo %INFO% Results available in: %RESULTS_DIR%\

REM Cleanup temp files
del "%RESULTS_DIR%\coverage-temp.txt" 2>nul

endlocal