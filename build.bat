@echo off
setlocal

echo [1/2] Checking Go environment...
go version
if %errorlevel% neq 0 (
    echo ERROR: Go environment not found
    pause
    exit /b 1
)

echo.
echo [2/2] Building cmd/main...
go build -o mifer.exe ./cmd/main

if %errorlevel% equ 0 (
    echo.
    echo BUILD SUCCESS!
    echo Output: mifer.exe
    pause
) else (
    echo.
    echo BUILD FAILED!
    pause
    exit /b 1
)