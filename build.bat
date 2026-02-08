@echo off
echo Building Spec Recon...
go build -o spec-recon.exe ./cmd/spec-recon
if %errorlevel% neq 0 (
    echo Build failed!
    exit /b %errorlevel%
)
echo Build successful: spec-recon.exe
pause
