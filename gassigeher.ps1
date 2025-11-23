Get-NetTCPConnection -LocalPort 8888 -ErrorAction SilentlyContinue | ForEach-Object { Stop-Process -Id $_.OwningProcess -Force }

.\gassigeher.exe