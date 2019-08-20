@echo off
setlocal EnableDelayedExpansion
SET back=%cd%
for /d %%i in (*) do (

set notest="true"
echo %%i
if "%%i" == "bin" set notest="false"
if "%%i" == "vendor" set notest="false"

if !notest! == "true" (
    copy vm\v8\windows\lib\*.dll %%i
    
    cd %%i
    go test -tags=integration -c 

    set "testfile=%%i.test.exe"
    echo "!testfile!"

    !testfile!

    if not "%%i" == "dapp" DEL *.dll

    cd ..
)
)
cd %back%
