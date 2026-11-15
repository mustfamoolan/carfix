; Inno Setup Script for CarFix Canada Stealth Harvester Node
; Generates a clean 1-click installer that updates canada_worker.exe without ever losing Chrome login cookies

#define MyAppName "CarFix Canada Node"
#define MyAppVersion "2.3.0"
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
Filename: "{app}\start_canada_node.bat"; Description: "Launch CarFix Canada Node now (Preserves Chrome Login Session)"; Flags: nowait postinstall skipifsilent runasoriginaluser

[Code]
var
  NodePage: TInputOptionWizardPage;

procedure InitializeWizard;
begin
  NodePage := CreateInputOptionPage(wpWelcome,
    'Select Canada Node Identity / اختيار رقم هذه الحاسبة',
    'Which Canada PC are you installing this worker on?',
    'Choose the node ID for this computer (you can install on multiple PCs and they will share the queue automatically):',
    True, False);
  NodePage.Add('Canada Node 1 (Primary) — حاسبة كندا 1 (الأساسية) [canada-node-1]');
  NodePage.Add('Canada Node 2 (Backup)  — حاسبة كندا 2 (الاحتياطية) [canada-node-2]');
  NodePage.Add('Canada Node 3 (Extra)   — حاسبة كندا 3 (إضافية) [canada-node-3]');
  NodePage.SelectedValueIndex := 0;
end;

function PrepareToInstall(var NeedsRestart: Boolean): String;
var
  ResultCode: Integer;
begin
  // Stop background wscript watchdog and canada_worker.exe so the binary can be updated cleanly without touching Chrome!
  Exec('taskkill.exe', '/F /IM wscript.exe', '', SW_HIDE, ewWaitUntilTerminated, ResultCode);
  Exec('taskkill.exe', '/F /IM canada_worker.exe', '', SW_HIDE, ewWaitUntilTerminated, ResultCode);
  Result := '';
end;

procedure CurStepChanged(CurStep: TSetupStep);
var
  EnvLines: TArrayOfString;
  WorkerID, WorkerName: String;
begin
  if CurStep = ssPostInstall then
  begin
    case NodePage.SelectedValueIndex of
      1:
        begin
          WorkerID := 'canada-node-2';
          WorkerName := 'حاسبة كندا 2 (الاحتياطية)';
        end;
      2:
        begin
          WorkerID := 'canada-node-3';
          WorkerName := 'حاسبة كندا 3 (إضافية)';
        end;
    else
      begin
        WorkerID := 'canada-node-1';
        WorkerName := 'حاسبة كندا 1 (الأساسية)';
      end;
    end;

    SetArrayLength(EnvLines, 6);
    EnvLines[0] := 'WORKER_ID=' + WorkerID;
    EnvLines[1] := 'WORKER_NAME=' + WorkerName;
    EnvLines[2] := 'SERVER_BASE_URL=https://vinreport.tech';
    EnvLines[3] := 'WORKER_SECRET_KEY=almoq3-canada-secret-2026';
    EnvLines[4] := 'CDP_WS_URL=http://127.0.0.1:9222';
    EnvLines[5] := 'POLL_INTERVAL_SEC=1';

    SaveStringsToUTF8File(ExpandConstant('{app}\.env'), EnvLines, False);
  end;
end;
