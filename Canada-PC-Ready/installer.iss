; Inno Setup Script for CarFix Canada Stealth Harvester Node
; Generates a clean 1-click installer that updates canada_worker.exe without ever losing Chrome login cookies

#define MyAppName "CarFix Canada Node"
#define MyAppVersion "2.1.0"
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
CloseApplications=no

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"

[Files]
Source: "canada_worker.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "start_canada_node.bat"; DestDir: "{app}"; Flags: ignoreversion
Source: "start_chrome_stealth.bat"; DestDir: "{app}"; Flags: ignoreversion
Source: "run_worker.bat"; DestDir: "{app}"; Flags: ignoreversion
Source: "silent_runner.vbs"; DestDir: "{app}"; Flags: ignoreversion
Source: ".env"; DestDir: "{app}"; Flags: ignoreversion onlyifdoesntexist

[Icons]
Name: "{group}\CarFix Canada Node"; Filename: "{app}\start_canada_node.bat"
Name: "{group}\Start Chrome Only"; Filename: "{app}\start_chrome_stealth.bat"
Name: "{group}\Start Worker Only"; Filename: "{app}\run_worker.bat"
Name: "{commondesktop}\CarFix Canada Node"; Filename: "{app}\start_canada_node.bat"
Name: "{autostartup}\CarFix Canada Node"; Filename: "{app}\start_canada_node.bat"

[Run]
Filename: "{app}\start_canada_node.bat"; Description: "Launch CarFix Canada Node now (Preserves Chrome Login Session)"; Flags: nowait postinstall skipifsilent

[Code]
function PrepareToInstall(var NeedsRestart: Boolean): String;
var
  ResultCode: Integer;
begin
  // Stop only canada_worker.exe so the binary can be updated cleanly without touching Chrome!
  Exec('taskkill.exe', '/F /IM canada_worker.exe', '', SW_HIDE, ewWaitUntilTerminated, ResultCode);
  Result := '';
end;
