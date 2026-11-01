' CarFix Canada Silent Background Runner
' Launches start_chrome_stealth.bat and canada_worker.exe with ZERO visible console windows

Set WshShell = CreateObject("WScript.Shell")
strCurDir = CreateObject("Scripting.FileSystemObject").GetParentFolderName(WScript.ScriptFullName)

' 1. Start Off-Screen Chrome Host silently
WshShell.Run """" & strCurDir & "\start_chrome_stealth.bat""", 0, False

' 2. Wait 3 seconds for Chrome port 9222 to bind
WScript.Sleep 3000

' 3. Start Canada Worker daemon silently
WshShell.Run """" & strCurDir & "\canada_worker.exe""", 0, False
