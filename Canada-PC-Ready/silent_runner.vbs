' CarFix Canada Silent Background Runner & Auto-Restart Watchdog
' Ensures Chrome Host is open (preserving login session) and keeps canada_worker.exe running 24/7 with zero visible CMD windows

Set WshShell = CreateObject("WScript.Shell")
Set FSO = CreateObject("Scripting.FileSystemObject")
strCurDir = FSO.GetParentFolderName(WScript.ScriptFullName)
WshShell.CurrentDirectory = strCurDir

' 1. Ensure Chrome Host is running on port 9222 (does nothing if already open)
WshShell.Run """" & strCurDir & "\start_chrome_stealth.bat""", 0, True

' 2. Wait 2 seconds for Chrome port 9222 to bind
WScript.Sleep 2000

' 3. Infinite Self-Healing Watchdog Loop: runs canada_worker.exe completely hidden (0)
'    and automatically restarts it within 3 seconds if it ever exits or crashes!
Do
    WshShell.Run """" & strCurDir & "\canada_worker.exe""", 0, True
    WScript.Sleep 3000
Loop
