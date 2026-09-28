@echo off
chcp 65001 >nul
cd /d %~dp0
if not exist gpu-acceptance.exe (
  where go >nul 2>nul || (echo [错误] 未找到 go，也未找到 gpu-acceptance.exe & pause & exit /b 1)
  echo 首次运行，正在构建 gpu-acceptance.exe ...
  go build -o gpu-acceptance.exe . || (pause & exit /b 1)
)
gpu-acceptance.exe
pause
