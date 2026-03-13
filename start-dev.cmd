@echo off
setlocal

set "ROOT_DIR=%~dp0"
set "SCRIPT_PATH=%ROOT_DIR%scripts\dev-all.ps1"

if not exist "%SCRIPT_PATH%" (
  echo Could not find "%SCRIPT_PATH%"
  pause
  exit /b 1
)

powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%SCRIPT_PATH%" -OpenBrowser
set "EXIT_CODE=%ERRORLEVEL%"

if not "%EXIT_CODE%"=="0" (
  echo.
  echo Startup failed with exit code %EXIT_CODE%.
  pause
)

endlocal & exit /b %EXIT_CODE%
