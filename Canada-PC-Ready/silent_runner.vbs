' CarFix Canada Silent Background Runner & Auto-Restart Watchdog
' Keeps canada_worker.exe running 24/7 with zero visible CMD windows

Set WshShell = CreateObject("WScript.Shell")
Set FSO = CreateObject("Scripting.FileSystemObject")
strCurDir = FSO.GetParentFolderName(WScript.ScriptFullName)
WshShell.CurrentDirectory = strCurDir

' Wait 2.5 seconds on initial startup so Chrome port 9222 finishes binding before worker checks it
WScript.Sleep 2500

' Infinite Self-Healing Watchdog Loop: runs canada_worker.exe completely hidden (0)
' and automatically restarts it within 3 seconds if it ever exits or crashes!
Do
    WshShell.Run """" & strCurDir & "\canada_worker.exe""", 0, True
    WScript.Sleep 3000
Loop
