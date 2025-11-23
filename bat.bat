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

echo [1/4] Checking Go version...
go version
echo.

echo [2/4] Downloading dependencies...
go mod download
if %ERRORLEVEL% NEQ 0 (
    echo [ERROR] Failed to download dependencies
    exit /b 1
)
echo [OK] Dependencies downloaded
echo.

echo [3/5] Building application for Windows...
go build -o gassigeher.exe cmd/server/main.go
if %ERRORLEVEL% NEQ 0 (
    echo [ERROR] Windows build failed
    exit /b 1
)
echo [OK] Windows build successful: gassigeher.exe
echo.

echo [4/5] Building application for Linux (cross-compile)...
set GOOS=linux
set GOARCH=amd64
go build -o gassigeher cmd/server/main.go
if %ERRORLEVEL% NEQ 0 (
    echo [ERROR] Linux build failed
    exit /b 1
)
echo [OK] Linux build successful: gassigeher (using pure Go SQLite)
set GOOS=
set GOARCH=
echo.

echo [5/5] Running tests...
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
