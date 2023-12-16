@echo off
setlocal enabledelayedexpansion

for /L %%i in (1, 1, 10000) do (
    set "url=https://www.example.com/%%i"
    curl -X POST -d "url=!url!" http://localhost:7070/shorten
    ping -n 1 -w 50 127.0.0.1 >nul
    echo.
)