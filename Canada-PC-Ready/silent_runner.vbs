' CarFix Canada Silent Background Runner
' Launches start_chrome_stealth.bat naturally (visible in taskbar) and canada_worker.exe in background

Set WshShell = CreateObject("WScript.Shell")
strCurDir = CreateObject("Scripting.FileSystemObject").GetParentFolderName(WScript.ScriptFullName)

' 1. Start Chrome Host normally so it has its standard taskbar icon and natural visibility state
WshShell.Run """" & strCurDir & "\start_chrome_stealth.bat""", 1, False

' 2. Wait 3 seconds for Chrome port 9222 to bind
WScript.Sleep 3000

' 3. Start Canada Worker daemon silently (without showing a black CMD window)
WshShell.Run """" & strCurDir & "\canada_worker.exe""", 0, False
