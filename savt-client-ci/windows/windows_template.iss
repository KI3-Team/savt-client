;----------------------------------------
; Basic Information
;----------------------------------------
#define MyAppName "savt-client"
#define MyAppVersion "2.0.0"
#define MyAppPublisher "KI3"
#define MyAppURL "https://ki3.org.cn"
#define OS "windows"
#define ARCH "arm64"
#define MyAppExeName "savt-client-gui-"
#define MyWorkerExeName "savt-client-worker-"
#define MyCliExeName "savt-client-cli-"
#define NpcapInstaller "npcap-1.82.exe"

[Setup]
AppId={{A19E3A39-762E-41F1-9433-1E614CA30031}}
AppName={#MyAppName}
AppVersion={#MyAppVersion}
AppPublisher={#MyAppPublisher}
AppPublisherURL={#MyAppURL}
AppSupportURL={#MyAppURL}
AppUpdatesURL={#MyAppURL}
DefaultDirName={autopf}\{#MyAppName}
OutputDir=.
OutputBaseFilename={#MyAppName}-{#MyAppVersion}-{#OS}-{#ARCH}
SetupIconFile=icon.ico
Compression=lzma
SolidCompression=yes
WizardStyle=modern
PrivilegesRequired=admin
AlwaysRestart=no

;----------------------------------------
; Files to Install
;----------------------------------------
[Files]
; Main application files
Source: "{#MyAppExeName}{#MyAppVersion}-{#OS}-{#ARCH}.exe"; DestDir: "{app}"; DestName: "{#MyAppExeName}{#MyAppVersion}-{#OS}-{#ARCH}.exe"; Flags: ignoreversion restartreplace
Source: "{#MyWorkerExeName}{#MyAppVersion}-{#OS}-{#ARCH}.exe"; DestDir: "{app}"; DestName: "{#MyWorkerExeName}{#MyAppVersion}-{#OS}-{#ARCH}.exe"; Flags: ignoreversion restartreplace
Source: "{#MyCliExeName}{#MyAppVersion}-{#OS}-{#ARCH}.exe"; DestDir: "{app}"; DestName: "{#MyCliExeName}{#MyAppVersion}-{#OS}-{#ARCH}.exe"; Flags: ignoreversion restartreplace
Source: "service.json"; DestDir: "{app}"; Flags: ignoreversion restartreplace
Source: "{#NpcapInstaller}"; DestDir: "{tmp}"; Flags: deleteafterinstall

; nssm.exe
Source: "nssm-2.24\win64\nssm.exe"; DestDir: "{app}"; Flags: ignoreversion restartreplace
Source: "icon.ico"; DestDir: "{app}"; Flags: ignoreversion restartreplace
;----------------------------------------
; Icons
;----------------------------------------
[Icons]
; Application shortcuts
Name: "{autoprograms}\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}{#MyAppVersion}-{#OS}-{#ARCH}.exe"; IconFilename: "{app}\icon.ico"; Tasks: startmenuicon
Name: "{autodesktop}\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}{#MyAppVersion}-{#OS}-{#ARCH}.exe"; IconFilename: "{app}\icon.ico"; Tasks: desktopicon

;----------------------------------------
; Post-install Commands (Create and Configure Services)
;----------------------------------------
[Tasks]
; Optional desktop shortcut creation
Name: "desktopicon"; Description: "Create a shortcut on the desktop"; GroupDescription: "Additional icons:";
Name: "startmenuicon"; Description: "Create a shortcut in the Start Menu"; GroupDescription: "Additional icons:";

[Run]

; Reinstall the service with new executable
Filename: "{app}\nssm.exe"; \
    Parameters: "install ""savt-client.savt-client-worker"" ""{app}\{#MyWorkerExeName}{#MyAppVersion}-{#OS}-{#ARCH}.exe"""; \
    StatusMsg: "Registering new service (NSSM)..."; \
    Flags: runhidden

; Set service
Filename: "{app}\nssm.exe"; \
    Parameters: "set ""savt-client.savt-client-worker"" DisplayName ""SAV Client Worker"""; \
    StatusMsg: "Setting service display name..."; \
    Flags: runhidden
Filename: "{app}\nssm.exe"; \
    Parameters: "set ""savt-client.savt-client-worker"" Description ""SAV Client Worker Service"""; \
    StatusMsg: "Setting service description..."; \
    Flags: runhidden
Filename: "{app}\nssm.exe"; \
    Parameters: "set ""savt-client.savt-client-worker"" Start SERVICE_AUTO_START"; \
    StatusMsg: "Setting service to auto-start..."; \
    Flags: runhidden

; Start the service
Filename: "net"; \
    Parameters: "start ""savt-client.savt-client-worker"""; \
    StatusMsg: "Starting service..."; \
    Flags: runhidden

; Run the application after installation if the user selects the option
Filename: "{app}\{#MyAppExeName}{#MyAppVersion}-{#OS}-{#ARCH}.exe"; \
    Description: "Launch {#MyAppName}"; \
    Flags: nowait postinstall skipifsilent

;----------------------------------------
; Uninstall Commands (Remove Services)
;----------------------------------------
[UninstallRun]
; Stop and remove savt-client.savt-client-worker service
Filename: "sc.exe"; \
    Parameters: "stop ""savt-client.savt-client-worker"""; \
    StatusMsg: "Stopping savt-client.savt-client-worker service if running..."; \
    Flags: runhidden waituntilterminated

Filename: "sc.exe"; \
    Parameters: "delete ""savt-client.savt-client-worker"""; \
    StatusMsg: "Removing savt-client.savt-client-worker service..."; \
    Flags: runhidden waituntilterminated

; Stop and remove sav-client.sav-client-worker service
Filename: "sc.exe"; \
    Parameters: "stop ""sav-client.sav-client-worker"""; \
    StatusMsg: "Stopping sav-client.sav-client-worker service if running..."; \
    Flags: runhidden waituntilterminated

Filename: "sc.exe"; \
    Parameters: "delete ""sav-client.sav-client-worker"""; \
    StatusMsg: "Removing sav-client.sav-client-worker service..."; \
    Flags: runhidden waituntilterminated

; Delete ProgramData folders
Filename: "cmd.exe"; \
    Parameters: "/C rmdir /S /Q ""%ProgramData%\savt-client"""; \
    StatusMsg: "Deleting ProgramData\savt-client folder..."; \
    Flags: runhidden waituntilterminated

Filename: "cmd.exe"; \
    Parameters: "/C rmdir /S /Q ""%ProgramData%\sav-client"""; \
    StatusMsg: "Deleting ProgramData\sav-client folder..."; \
    Flags: runhidden waituntilterminated


[Code]
function IsNpcapInstalled: Boolean;
var
  RegValue: String;
begin
  if RegQueryStringValue(HKEY_LOCAL_MACHINE, 'SOFTWARE\WOW6432Node\Npcap', '', RegValue) then
  begin
    Log('Npcap is installed under WOW6432Node.');
    Result := True;
  end
  else if RegQueryStringValue(HKEY_LOCAL_MACHINE, 'SOFTWARE\Npcap', '', RegValue) then
  begin
    Log('Npcap is installed under SOFTWARE.');
    Result := True;
  end
  else
  begin
    Log('Npcap is not installed.');
    Result := False;
  end;
end;

function InstallNpcap: Boolean;
var
  ResultCode: Integer;
  NpcapInstallerPath: String;
begin
  NpcapInstallerPath := ExpandConstant('{tmp}\{#NpcapInstaller}');
  
  if not FileExists(NpcapInstallerPath) then
  begin
    Log('Npcap installer not found at: ' + NpcapInstallerPath);
    MsgBox('Npcap installer not found. Please contact support.', mbError, MB_OK);
    Result := False;
    Exit;
  end;

  Log('Starting Npcap installation...');
  if not ShellExec('runas', NpcapInstallerPath, '/S', '', SW_SHOW, ewWaitUntilTerminated, ResultCode) then
  begin
    Log('Failed to run Npcap installer. Error code: ' + IntToStr(ResultCode));
    MsgBox('Failed to install Npcap. Please try again.', mbError, MB_OK);
    Result := False;
    Exit;
  end;

  // 等待一段时间确保安装完成
  Sleep(5000);
  
  // 再次检查是否安装成功
  if IsNpcapInstalled then
  begin
    Log('Npcap installation completed successfully.');
    Result := True;
  end
  else
  begin
    Log('Npcap installation may have failed. Please check manually.');
    MsgBox('Npcap installation may have failed. Please check if Npcap is installed correctly.', mbWarning, MB_OK);
    Result := False;
  end;
end;

function ServiceExists(ServiceName: String): Boolean;
var
  ResultCode: Integer;
begin
  Result := Exec('sc', 'query "' + ServiceName + '"', '', SW_HIDE, ewWaitUntilTerminated, ResultCode) and (ResultCode = 0);
  if Result then
    Log('Service "' + ServiceName + '" exists.')
  else
    Log('Service "' + ServiceName + '" does not exist or query failed. ResultCode: ' + IntToStr(ResultCode));
end;

procedure RunHiddenCommand(Command, Parameters: String);
var
  ErrorCode: Integer;
begin
  if not ShellExec('open', Command, Parameters, '', SW_HIDE, ewWaitUntilTerminated, ErrorCode) then
  begin
    Log('Failed to run command: ' + Command + ' ' + Parameters + '. ErrorCode: ' + IntToStr(ErrorCode));
    MsgBox('An error occurred while running the command. Please check the logs for details.', mbError, MB_OK);
  end;
end;

procedure StopAndRemoveService(ServiceName: String);
begin
  RunHiddenCommand('sc.exe', 'stop "' + ServiceName + '"');
  RunHiddenCommand('sc.exe', 'delete "' + ServiceName + '"');
end;

procedure DeleteProgramDataFolder(FolderName: String);
var
  ProgramDataPath: String;
begin
  ProgramDataPath := ExpandConstant('{commonappdata}') + '\' + FolderName;

  if DirExists(ProgramDataPath) then
  begin
    if not DelTree(ProgramDataPath, True, True, True) then
      Log('Failed to delete folder "' + ProgramDataPath + '". Please delete it manually.')
    else
      Log('Folder "' + ProgramDataPath + '" deleted successfully.');
  end
  else
    Log('Folder "' + ProgramDataPath + '" does not exist.');
end;

function InitializeSetup: Boolean;
var
  Services: array[0..1] of String;
  I: Integer;
begin
  // 首先检查并安装 Npcap
  if not IsNpcapInstalled then
  begin
    if not InstallNpcap then
    begin
      MsgBox('Npcap installation is required to continue. Setup will now exit.', mbError, MB_OK);
      Result := False;
      Exit;
    end;
  end;

  // Define service names to process
  Services[0] := 'savt-client.savt-client-worker';
  Services[1] := 'sav-client.sav-client-worker';

  try
    // Iterate through service list and process each one
    for I := 0 to High(Services) do
    begin
      if ServiceExists(Services[I]) then
      begin
        Log('Service "' + Services[I] + '" exists. Attempting to stop and remove it...');
        StopAndRemoveService(Services[I]);
      end
      else
      begin
        Log('Service "' + Services[I] + '" does not exist. No action needed.');
      end;
    end;

    // Delete ProgramData directory
    DeleteProgramDataFolder('savt-client');
    DeleteProgramDataFolder('sav-client');
    // Initialization successful
    Log('Initialization completed successfully.');
    Result := True;
  except
    // Catch all exceptions, log and abort installation
    Log('Initialization failed due to an unexpected error.');
    MsgBox('An error occurred during setup initialization. Please check the logs for details.', mbError, MB_OK);
    Result := False; // Abort installation
  end;
end;

procedure InitializeWizard;
begin
  // 移除原有的 Npcap 检查逻辑，因为已经在 InitializeSetup 中处理
  Log('Wizard initialization completed.');
end;


