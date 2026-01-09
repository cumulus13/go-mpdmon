@echo off

echo Building mpdmon.exe for Windows...

for /f "tokens=* delims=" %%a in ('date /t') do set current_date=%%a
for /f "tokens=* delims=" %%b in ('time /t') do set current_time=%%b

REM Clean previous build
if exist mpdmon.exe del mpdmon.exe

REM Build
pt -b main.go -c

go build -v -ldflags="-s -w" -o mpdmon.exe main.go

if %ERRORLEVEL% EQU 0 (
    echo.
    echo ✅ Build successful: mpdmon.exe
    sendgrowl.exe "GoBuilder" build "SUCCESS %current_date% %current_time%" "[%current_date% %current_time%] ✅ Build successful: mpdmon.exe" -i "c:\TOOLS\EXE\go.png" -H 127.0.0.1
    echo.
    dir mpdmon.exe
    copy /y mpdmon.exe c:\TOOLS\EXE\

) else (
    echo.
    echo ❌ Build failed!
    sendgrowl.exe "GoBuilder" build "FAILED %current_date% %current_time%" "[%current_date% %current_time%] ❌ Build failed!: mpdmon.exe" -i "c:\TOOLS\EXE\go.png" -H 127.0.0.1
    exit /b 1
)