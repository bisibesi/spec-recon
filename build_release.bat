@echo off
echo [BUILD] Cleaning up...
rmdir /s /q dist
mkdir dist

echo [BUILD] Building for Windows (amd64)...
set GOOS=windows
set GOARCH=amd64
go build -o dist/windows/spec-recon.exe ./cmd/spec-recon
copy README.md dist\windows\

echo [BUILD] Building for Linux (amd64)...
set GOOS=linux
set GOARCH=amd64
go build -o dist/linux/spec-recon ./cmd/spec-recon
copy README.md dist\linux\

echo [BUILD] Building for Mac (arm64/M1)...
set GOOS=darwin
set GOARCH=arm64
go build -o dist/mac_arm64/spec-recon ./cmd/spec-recon
copy README.md dist\mac_arm64\

echo [BUILD] Compressing...
powershell Compress-Archive -Path dist\windows\* -DestinationPath dist\spec-recon-v1.0.0-windows.zip
powershell Compress-Archive -Path dist\linux\* -DestinationPath dist\spec-recon-v1.0.0-linux.zip
powershell Compress-Archive -Path dist\mac_arm64\* -DestinationPath dist\spec-recon-v1.0.0-mac-arm64.zip

echo [SUCCESS] Build Complete! Check the 'dist' folder.
pause