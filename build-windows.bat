@echo off
setlocal
cd /d "%~dp0"
if not exist dist mkdir dist

echo [1/3] Preparing Windows icon resource...
go run github.com/akavel/rsrc@latest -ico assets\ui\icon.ico -o rsrc_windows_amd64.syso
if errorlevel 1 exit /b 1

echo [2/3] Downloading Go modules...
go mod tidy
if errorlevel 1 exit /b 1

echo [3/3] Building LClub.exe...
set GOOS=windows
set GOARCH=amd64
go build -trimpath -ldflags="-s -w -H=windowsgui" -o dist\lclub-win64.exe .
if errorlevel 1 exit /b 1

echo Done: dist\LClub.exe
endlocal
