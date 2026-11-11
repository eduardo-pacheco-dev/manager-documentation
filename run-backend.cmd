@echo off
cd /d "%~dp0backend"
if not exist "..\tmp\server.exe" exit /b 1
..\tmp\server.exe
