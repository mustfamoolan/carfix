; Inno Setup Script for CarFix Canada Stealth Harvester Node
; Generates a clean 1-click installer that sets up Off-Screen Chrome & Canada Worker

#define MyAppName "CarFix Canada Node"
#define MyAppVersion "2.0.0"
#define MyAppPublisher "CarFix Platform"
#define MyAppURL "https://vinreport.tech"

[Setup]
AppId={{E679EA08-53D2-4073-B76E-16ECFEFFD69A}
AppName={#MyAppName}
AppVersion={#MyAppVersion}
AppPublisher={#MyAppPublisher}
AppPublisherURL={#MyAppURL}
DefaultDirName=C:\CarFixCanadaNode
DisableDirPage=yes
DefaultGroupName={#MyAppName}
DisableProgramGroupPage=yes
OutputBaseFilename=CarFix-Canada-Node-Setup
Compression=lzma2/max
SolidCompression=yes
WizardStyle=modern
PrivilegesRequired=admin

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"

[Files]
Source: "canada_worker.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "start_chrome_stealth.bat"; DestDir: "{app}"; Flags: ignoreversion
Source: "run_worker.bat"; DestDir: "{app}"; Flags: ignoreversion
Source: "silent_runner.vbs"; DestDir: "{app}"; Flags: ignoreversion
Source: ".env"; DestDir: "{app}"; Flags: ignoreversion onlyifdoesntexist

[Icons]
Name: "{group}\Start Canada Node (Silent)"; Filename: "wscript.exe"; Parameters: """{app}\silent_runner.vbs"""
Name: "{group}\Start Chrome Stealth (Diagnostic)"; Filename: "{app}\start_chrome_stealth.bat"
Name: "{group}\Start Worker (Console)"; Filename: "{app}\run_worker.bat"
Name: "{autostartup}\CarFix Canada Node"; Filename: "wscript.exe"; Parameters: """{app}\silent_runner.vbs"""

[Run]
Filename: "wscript.exe"; Parameters: """{app}\silent_runner.vbs"""; Description: "Launch CarFix Canada Node now (Silent Background Mode)"; Flags: nowait postinstall skipifsilent
