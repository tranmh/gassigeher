@echo off
REM Gassigeher - Build and Test Script for Windows
REM Builds binaries for both Windows and Linux
REM Usage: bat.bat

echo ========================================
echo Gassigeher - Build and Test
echo (Windows + Linux)
echo ========================================
echo.

REM Check if Go is installed
where go >nul 2>nul
if %ERRORLEVEL% NEQ 0 (
    echo [ERROR] Go is not installed or not in PATH
    exit /b 1
)

echo [1/6] Checking Go version...
go version
echo.

echo [2/6] Downloading dependencies...
go mod download
if %ERRORLEVEL% NEQ 0 (
    echo [ERROR] Failed to download dependencies
    exit /b 1
)
echo [OK] Dependencies downloaded
echo.

echo [3/6] Preparing version info...
set VERSION=1.0
for /f %%i in ('git rev-parse --short HEAD 2^>nul') do set GIT_COMMIT=%%i
if "%GIT_COMMIT%"=="" set GIT_COMMIT=unknown
for /f %%i in ('powershell -command "Get-Date -Format 'yyyy-MM-ddTHH:mm:ssZ' -AsUTC"') do set BUILD_TIME=%%i
if "%BUILD_TIME%"=="" set BUILD_TIME=unknown
set LDFLAGS=-X github.com/tranmh/gassigeher/internal/version.Version=%VERSION% -X github.com/tranmh/gassigeher/internal/version.GitCommit=%GIT_COMMIT% -X github.com/tranmh/gassigeher/internal/version.BuildTime=%BUILD_TIME%
echo [OK] Version: %VERSION% (%GIT_COMMIT%)
echo.

echo [4/6] Building application for Windows...
go build -ldflags "%LDFLAGS%" -o gassigeher.exe ./cmd/server
if %ERRORLEVEL% NEQ 0 (
    echo [ERROR] Windows build failed
    exit /b 1
)
echo [OK] Windows build successful: gassigeher.exe v%VERSION% (%GIT_COMMIT%)
echo.

echo [5/6] Building application for Linux (cross-compile)...
set GOOS=linux
set GOARCH=amd64
go build -ldflags "%LDFLAGS%" -o gassigeher ./cmd/server
if %ERRORLEVEL% NEQ 0 (
    echo [ERROR] Linux build failed
    exit /b 1
)
echo [OK] Linux build successful: gassigeher v%VERSION% (%GIT_COMMIT%)
set GOOS=
set GOARCH=
echo.

echo [6/6] Running tests...
go test -v -cover ./...
if %ERRORLEVEL% NEQ 0 (
    echo [WARNING] Some tests failed
    echo.
) else (
    echo [OK] All tests passed
    echo.
)

echo ========================================
echo Build and Test Complete!
echo ========================================
echo.
echo Built binaries:
echo   Windows: gassigeher.exe
echo   Linux:   gassigeher
echo.
echo To run on Windows:
echo   .\gassigeher.exe
echo.
echo To run on Linux:
echo   chmod +x gassigeher ^&^& ./gassigeher
echo.
echo To run with custom port:
echo   set PORT=3000 ^&^& .\gassigeher.exe  (Windows)
echo   PORT=3000 ./gassigeher              (Linux)
echo.
