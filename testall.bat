@echo off
set back=%cd%
for /d %%i in (*) do (
set test=true
echo %%i
if %%i == "bin" set test=
if %%i == "vendor" set test=

if defined test (
    cd %%i 
    go test --tags="integration"
    cd ../
) 

)
cd %back%