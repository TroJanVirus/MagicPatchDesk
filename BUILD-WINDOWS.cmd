@echo off
setlocal
cd /d "%~dp0"
where go >nul 2>&1 || (echo Go 1.23+ is required ONLY to build the source. The prebuilt EXE needs no runtime. & pause & exit /b 1)
set GOOS=windows
set GOARCH=amd64
set CGO_ENABLED=0
call go test ./core || (echo Tests failed & pause & exit /b 1)
call go build -trimpath -ldflags="-H windowsgui -s -w" -o MagicPatchDesk-Windows-x64.exe ./cmd/magicdesk || (echo Build failed & pause & exit /b 1)
echo Built MagicPatchDesk-Windows-x64.exe
pause
