@echo off
REM Load Testing Runner Script for Windows
REM This script runs various load tests for the high-performance news website

setlocal enabledelayedexpansion

REM Configuration
if "%BASE_URL%"=="" set BASE_URL=http://localhost:8080
if "%TEST_USERNAME%"=="" set TEST_USERNAME=testuser
if "%TEST_PASSWORD%"=="" set TEST_PASSWORD=testpass123
if "%OUTPUT_DIR%"=="" set OUTPUT_DIR=.\results

echo [INFO] Starting load testing framework for high-performance news website
echo [INFO] Target: 50K daily articles, 100 concurrent users, sub-2s response times
echo [INFO] Server: %BASE_URL%

REM Check if k6 is installed
k6 version >nul 2>&1
if errorlevel 1 (
    echo [ERROR] k6 is not installed. Please install k6 first.
    echo [INFO] Visit: https://k6.io/docs/getting-started/installation/
    exit /b 1
)
echo [SUCCESS] k6 is installed

REM Check if server is accessible
curl -f -s "%BASE_URL%/health" >nul 2>&1
if errorlevel 1 (
    echo [ERROR] Server is not accessible at %BASE_URL%
    echo [INFO] Please ensure the server is running and accessible
    exit /b 1
)
echo [SUCCESS] Server is accessible

REM Create output directory
if not exist "%OUTPUT_DIR%" mkdir "%OUTPUT_DIR%"
echo [INFO] Results will be saved to: %OUTPUT_DIR%

REM Determine which test to run
set TEST_SCENARIO=%1
if "%TEST_SCENARIO%"=="" set TEST_SCENARIO=all

if "%TEST_SCENARIO%"=="baseline" goto run_baseline
if "%TEST_SCENARIO%"=="articles" goto run_articles
if "%TEST_SCENARIO%"=="database" goto run_database
if "%TEST_SCENARIO%"=="comprehensive" goto run_comprehensive
if "%TEST_SCENARIO%"=="all" goto run_all
if "%TEST_SCENARIO%"=="-h" goto show_help
if "%TEST_SCENARIO%"=="--help" goto show_help

echo [ERROR] Unknown test scenario: %TEST_SCENARIO%
echo [INFO] Usage: %0 [baseline^|articles^|database^|comprehensive^|all]
exit /b 1

:run_baseline
echo [INFO] Running performance baseline test...
k6 run --env BASE_URL="%BASE_URL%" --env TEST_USERNAME="%TEST_USERNAME%" --env TEST_PASSWORD="%TEST_PASSWORD%" --out json="%OUTPUT_DIR%\baseline-results.json" --summary-export="%OUTPUT_DIR%\baseline-summary.json" performance-baseline.js
if errorlevel 1 (
    echo [ERROR] Baseline test failed
    exit /b 1
)
echo [SUCCESS] Baseline test completed successfully
goto generate_report

:run_articles
echo [INFO] Running article creation test (35 articles/minute target)...
k6 run --env BASE_URL="%BASE_URL%" --env TEST_USERNAME="%TEST_USERNAME%" --env TEST_PASSWORD="%TEST_PASSWORD%" --out json="%OUTPUT_DIR%\article-creation-results.json" --summary-export="%OUTPUT_DIR%\article-creation-summary.json" article-creation-test.js
if errorlevel 1 (
    echo [ERROR] Article creation test failed
    exit /b 1
)
echo [SUCCESS] Article creation test completed successfully
goto generate_report

:run_database
echo [INFO] Running database bottleneck test...
k6 run --env BASE_URL="%BASE_URL%" --env TEST_USERNAME="%TEST_USERNAME%" --env TEST_PASSWORD="%TEST_PASSWORD%" --env K6_SCENARIO="query_performance" --out json="%OUTPUT_DIR%\database-results.json" --summary-export="%OUTPUT_DIR%\database-summary.json" database-bottleneck-test.js
if errorlevel 1 (
    echo [ERROR] Database test failed
    exit /b 1
)
echo [SUCCESS] Database test completed successfully
goto generate_report

:run_comprehensive
echo [INFO] Running comprehensive load test (100 concurrent users)...
k6 run --env BASE_URL="%BASE_URL%" --env TEST_USERNAME="%TEST_USERNAME%" --env TEST_PASSWORD="%TEST_PASSWORD%" --out json="%OUTPUT_DIR%\comprehensive-results.json" --summary-export="%OUTPUT_DIR%\comprehensive-summary.json" k6-setup.js
if errorlevel 1 (
    echo [ERROR] Comprehensive test failed
    exit /b 1
)
echo [SUCCESS] Comprehensive test completed successfully
goto generate_report

:run_all
echo [INFO] Running all test scenarios...
call :run_baseline
if errorlevel 1 exit /b 1
call :run_articles
if errorlevel 1 exit /b 1
call :run_database
if errorlevel 1 exit /b 1
call :run_comprehensive
if errorlevel 1 exit /b 1
goto generate_report

:generate_report
echo [INFO] Generating test report...
(
echo # Load Testing Report
echo.
echo Generated on: %date% %time%
echo Server: %BASE_URL%
echo Test Duration: Various scenarios
echo.
echo ## Test Results Summary
echo.
echo ### Performance Baseline
echo - **File**: baseline-results.json
echo - **Purpose**: Establish performance baselines for database operations and API responses
echo - **Key Metrics**: Database connection time, cache hit rate, query execution time
echo.
echo ### Article Creation Test
echo - **File**: article-creation-results.json  
echo - **Purpose**: Test article creation at 35 articles/minute rate (50K daily target^)
echo - **Key Metrics**: Article creation success rate, creation duration, database insert time
echo.
echo ### Database Bottleneck Test
echo - **File**: database-results.json
echo - **Purpose**: Identify bottlenecks in database queries and connection handling
echo - **Key Metrics**: Connection pool utilization, query execution time, slow query count
echo.
echo ### Comprehensive Load Test
echo - **File**: comprehensive-results.json
echo - **Purpose**: Test overall system performance with 100 concurrent users
echo - **Key Metrics**: Response times, error rates, throughput
echo.
echo ## Performance Requirements (from Requirement 22^)
echo.
echo ^| Metric ^| Target ^| Status ^|
echo ^|--------^|--------^|--------^|
echo ^| Article publishing ^| ^< 1 second ^| Check results ^|
echo ^| Homepage (cached^) ^| ^< 500ms ^| Check results ^|
echo ^| Homepage (dynamic^) ^| ^< 2 seconds ^| Check results ^|
echo ^| Search queries ^| ^< 200ms ^| Check results ^|
echo ^| API requests ^| ^< 100ms ^| Check results ^|
echo ^| Database queries ^| ^< 10ms ^| Check results ^|
echo ^| Static files ^| ^< 50ms ^| Check results ^|
echo ^| Concurrent users ^| 10,000+ ^| Check results ^|
echo ^| Daily articles ^| 50,000+ ^| Check results ^|
echo ^| Peak publishing ^| 1000/minute ^| Check results ^|
echo.
echo ## Files Generated
echo.
) > "%OUTPUT_DIR%\test-report.md"

for %%f in ("%OUTPUT_DIR%\*.json") do (
    echo - %%~nxf >> "%OUTPUT_DIR%\test-report.md"
)

echo [SUCCESS] Test report generated: %OUTPUT_DIR%\test-report.md
echo [SUCCESS] All tests completed successfully!
echo [INFO] Check results in: %OUTPUT_DIR%
goto end

:show_help
echo Load Testing Framework for High-Performance News Website
echo.
echo Usage: %0 [TEST_SCENARIO]
echo.
echo Test Scenarios:
echo   baseline      - Run performance baseline measurements
echo   articles      - Test article creation at 35/minute rate
echo   database      - Test database bottlenecks and connection handling
echo   comprehensive - Run comprehensive load test with 100 concurrent users
echo   all           - Run all test scenarios (default^)
echo.
echo Options:
echo   -h, --help    - Show this help message
echo.
echo Environment Variables:
echo   BASE_URL      - Server URL (default: http://localhost:8080^)
echo   TEST_USERNAME - Test user username (default: testuser^)
echo   TEST_PASSWORD - Test user password (default: testpass123^)
echo   OUTPUT_DIR    - Results output directory (default: .\results^)
echo.
echo Examples:
echo   %0                          # Run all tests
echo   %0 baseline                 # Run only baseline test
echo   %0 articles                 # Run only article creation test
echo   set BASE_URL=https://example.com ^& %0 comprehensive  # Test remote server
echo.
echo Requirements:
echo   - k6 load testing tool installed
echo   - Server running and accessible
echo   - Test user account configured
echo   - Sufficient system resources for load generation
echo.
echo Performance Targets (Requirement 22^):
echo   - Article publishing: ^< 1 second
echo   - Homepage (cached^): ^< 500ms  
echo   - Homepage (dynamic^): ^< 2 seconds
echo   - Search queries: ^< 200ms
echo   - API requests: ^< 100ms
echo   - Database queries: ^< 10ms
echo   - Support 10,000+ concurrent users
echo   - Handle 50,000+ articles per day
echo   - Peak publishing: 1000 articles/minute
goto end

:end
endlocal