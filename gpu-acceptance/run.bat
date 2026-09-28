@echo off
chcp 65001 >nul
cd /d %~dp0
where python >nul 2>nul || (echo [错误] 未找到 python，请先安装 Python 3.8+ & pause & exit /b 1)
python app.py
pause
