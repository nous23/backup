Dim WinScriptHost
Set WinScriptHost = CreateObject("WScript.Shell")
WinScriptHost.Run "{{PROGRAM_PATH}} -v={{LOG_LEVEL}} -log_dir={{LOG_DIR}} -log_name=backup -append=true", 0, false
Set WinScriptHost = Nothing