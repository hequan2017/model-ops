#!/usr/bin/env bash
cd "$(dirname "$0")"
exec python3 app.py 2>/dev/null || exec python app.py
