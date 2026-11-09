;----------------------------------------
; Basic Information
;----------------------------------------
#define MyAppName "savt-client"
#define MyAppVersion "2.0.0"
#define MyAppPublisher "KI3"
#define MyAppURL "https://ki3.org.cn"
#define OS "windows"
#define ARCH "arm64"
#define MyAppExeName "savt-client-gui.exe"
#define MyWorkerExeName "savt-client-worker.exe"
#define MyCliExeName "savt-client-cli.exe"
#define MyUpgraderExeName "savt-client-upgrader.exe"
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
RestartIfNeededByRun=no

;----------------------------------------
; Remove legacy versioned binaries on upgrade
;----------------------------------------
[InstallDelete]
Type: files; Name: "{app}\savt-client-gui-*.exe"
Type: files; Name: "{app}\savt-client-worker-*.exe"
Type: files; Name: "{app}\savt-client-cli-*.exe"
Type: files; Name: "{app}\savt-client-upgrader-*.exe"

;----------------------------------------
; Files to Install
;----------------------------------------
[Files]
; Main application files
Source: "{#MyAppExeName}"; DestDir: "{app}"; DestName: "{#MyAppExeName}"; Flags: ignoreversion restartreplace
Source: "{#MyWorkerExeName}"; DestDir: "{app}"; DestName: "{#MyWorkerExeName}"; Flags: ignoreversion restartreplace
Source: "{#MyCliExeName}"; DestDir: "{app}"; DestName: "{#MyCliExeName}"; Flags: ignoreversion restartreplace
Source: "{#MyUpgraderExeName}"; DestDir: "{app}"; DestName: "{#MyUpgraderExeName}"; Flags: ignoreversion restartreplace
Source: "service.json"; DestDir: "{app}"; Flags: ignoreversion restartreplace
Source: "{#NpcapInstaller}"; DestDir: "{app}"; Flags: deleteafterinstall

; nssm.exe
Source: "nssm-2.24\win64\nssm.exe"; DestDir: "{app}"; Flags: ignoreversion restartreplace
Source: "icon.ico"; DestDir: "{app}"; Flags: ignoreversion restartreplace
;----------------------------------------
; Icons
;----------------------------------------
[Icons]
; Application shortcuts
Name: "{autoprograms}\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"; IconFilename: "{app}\icon.ico"; Tasks: startmenuicon
Name: "{autodesktop}\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"; IconFilename: "{app}\icon.ico"; Tasks: desktopicon

;----------------------------------------
; Post-install Commands (Create and Configure Services)
;----------------------------------------
[Tasks]
; Optional desktop shortcut creation
Name: "desktopicon"; Description: "Create a shortcut on the desktop"; GroupDescription: "Additional icons:";
Name: "startmenuicon"; Description: "Create a shortcut in the Start Menu"; GroupDescription: "Additional icons:";

[Run]
; Install Npcap if not already installed
Filename: "{app}\{#NpcapInstaller}"; \
    StatusMsg: "Installing Npcap..."; \
    Check: ShouldInstallNpcap; \
    Flags: waituntilterminated shellexec

; Reinstall the service with new executable
Filename: "{app}\nssm.exe"; \
    Parameters: "install ""savt-client.savt-client-worker"" ""{app}\{#MyWorkerExeName}"""; \
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

; Start the worker service
Filename: "net"; \
    Parameters: "start ""savt-client.savt-client-worker"""; \
    StatusMsg: "Starting service..."; \
    Flags: runhidden

; Register savt-client-upgrader service
Filename: "{app}\nssm.exe"; \
    Parameters: "install ""savt-client.savt-client-upgrader"" ""{app}\{#MyUpgraderExeName}"""; \
    StatusMsg: "Registering upgrader service (NSSM)..."; \
    Flags: runhidden
Filename: "{app}\nssm.exe"; \
    Parameters: "set ""savt-client.savt-client-upgrader"" DisplayName ""SAV Client Upgrader"""; \
    StatusMsg: "Setting upgrader service display name..."; \
    Flags: runhidden
Filename: "{app}\nssm.exe"; \
    Parameters: "set ""savt-client.savt-client-upgrader"" Description ""SAV Client Upgrader Service"""; \
    StatusMsg: "Setting upgrader service description..."; \
    Flags: runhidden
Filename: "{app}\nssm.exe"; \
    Parameters: "set ""savt-client.savt-client-upgrader"" Start SERVICE_AUTO_START"; \
    StatusMsg: "Setting upgrader service to auto-start..."; \
    Flags: runhidden
Filename: "net"; \
    Parameters: "start ""savt-client.savt-client-upgrader"""; \
    StatusMsg: "Starting upgrader service..."; \
    Flags: runhidden

; Run the application after installation if the user selects the option
Filename: "{app}\{#MyAppExeName}"; \
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

; Stop and remove savt-client.savt-client-upgrader service
Filename: "sc.exe"; \
    Parameters: "stop ""savt-client.savt-client-upgrader"""; \
    StatusMsg: "Stopping savt-client.savt-client-upgrader service if running..."; \
    Flags: runhidden waituntilterminated

Filename: "sc.exe"; \
    Parameters: "delete ""savt-client.savt-client-upgrader"""; \
    StatusMsg: "Removing savt-client.savt-client-upgrader service..."; \
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

; Legacy folder: full delete.
Filename: "cmd.exe"; \
    Parameters: "/C if exist ""%ProgramData%\sav-client"" rmdir /S /Q ""%ProgramData%\sav-client"""; \
    StatusMsg: "Deleting ProgramData\sav-client folder..."; \
    Flags: runhidden waituntilterminated

; Clean savt-client but keep durable device.id.
; Backup outside the folder (NOT {tmp} — Inno deletes {tmp} before usPostUninstall).
Filename: "cmd.exe"; \
    Parameters: "/C if exist ""%ProgramData%\savt-client\device.id"" copy /Y ""%ProgramData%\savt-client\device.id"" ""%ProgramData%\savt-client.device.id.bak"" & if exist ""%ProgramData%\savt-client"" rmdir /S /Q ""%ProgramData%\savt-client"" & mkdir ""%ProgramData%\savt-client"" & if exist ""%ProgramData%\savt-client.device.id.bak"" (copy /Y ""%ProgramData%\savt-client.device.id.bak"" ""%ProgramData%\savt-client\device.id"" & del /F /Q ""%ProgramData%\savt-client.device.id.bak"")"; \
    StatusMsg: "Cleaning ProgramData\savt-client (keeping device.id)..."; \
    Flags: runhidden waituntilterminated

[UninstallDelete]
; Legacy install-dir device id only. Durable id is %ProgramData%\savt-client\device.id
; and must survive uninstall/reinstall.
Type: files; Name: "{app}\savt-client.device.id"
; Leftover versioned binaries from older installers.
Type: files; Name: "{app}\savt-client-gui-*.exe"
Type: files; Name: "{app}\savt-client-worker-*.exe"
Type: files; Name: "{app}\savt-client-cli-*.exe"
Type: files; Name: "{app}\savt-client-upgrader-*.exe"

[Code]
var
  UserAcceptedNpcap: Boolean;

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

procedure InitializeWizard;
begin
  UserAcceptedNpcap := True;  // 默认接受
  
  if not IsNpcapInstalled then
  begin
    if MsgBox('This application requires Npcap to function properly.' + #13#10 + #13#10 +
              'Npcap is not installed. Do you want to install it during setup?' + #13#10 + #13#10 +
              'Click Yes to install Npcap and continue.' + #13#10 +
              'Click No to exit the installation.',
              mbConfirmation, MB_YESNO) = IDNO then
    begin
      UserAcceptedNpcap := False;
      MsgBox('Installation cannot continue without Npcap. Setup will now exit.', mbError, MB_OK);
      Abort;
    end;
  end;
end;

function ShouldInstallNpcap: Boolean;
begin
  Result := UserAcceptedNpcap and (not IsNpcapInstalled);
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

// Clean ProgramData\<FolderName> but keep durable device.id for reinstall.
// Backup to a sibling under ProgramData — never {tmp} (deleted before usPostUninstall).
procedure CleanProgramDataFolderKeepDeviceID(FolderName: String);
var
  ProgramDataPath, DeviceIdPath, PreservePath: String;
  HadDeviceId: Boolean;
begin
  ProgramDataPath := ExpandConstant('{commonappdata}') + '\' + FolderName;
  DeviceIdPath := ProgramDataPath + '\device.id';
  PreservePath := ExpandConstant('{commonappdata}') + '\' + FolderName + '.device.id.bak';
  HadDeviceId := False;

  if not DirExists(ProgramDataPath) then
  begin
    Log('Folder "' + ProgramDataPath + '" does not exist.');
    // Still try to restore from a leftover backup.
    if FileExists(PreservePath) then
    begin
      ForceDirectories(ProgramDataPath);
      if FileCopy(PreservePath, DeviceIdPath, False) then
      begin
        Log('Restored durable device.id from leftover backup "' + PreservePath + '"');
        DeleteFile(PreservePath);
      end;
    end;
    Exit;
  end;

  if FileExists(DeviceIdPath) then
  begin
    if FileCopy(DeviceIdPath, PreservePath, False) then
    begin
      HadDeviceId := True;
      Log('Preserved durable device.id to "' + PreservePath + '"');
    end
    else
      Log('Failed to backup device.id from "' + DeviceIdPath + '"');
  end
  else if FileExists(PreservePath) then
    HadDeviceId := True;

  if not DelTree(ProgramDataPath, True, True, True) then
    Log('Failed to delete folder "' + ProgramDataPath + '". Please delete it manually.')
  else
    Log('Folder "' + ProgramDataPath + '" cleaned.');

  if HadDeviceId and FileExists(PreservePath) then
  begin
    if not DirExists(ProgramDataPath) then
      ForceDirectories(ProgramDataPath);
    if FileCopy(PreservePath, DeviceIdPath, False) then
      Log('Restored durable device.id to "' + DeviceIdPath + '"')
    else
      Log('Failed to restore device.id to "' + DeviceIdPath + '"');
    DeleteFile(PreservePath);
  end;
end;

procedure CurUninstallStepChanged(CurUninstallStep: TUninstallStep);
begin
  // usUninstall: backup+clean early. usPostUninstall: ensure restore if anything wiped the folder.
  if (CurUninstallStep = usUninstall) or (CurUninstallStep = usPostUninstall) then
  begin
    CleanProgramDataFolderKeepDeviceID('savt-client');
    if CurUninstallStep = usUninstall then
      DeleteProgramDataFolder('sav-client');
  end;
end;

function InitializeSetup: Boolean;
var
  Services: array[0..2] of String;
  I: Integer;
begin
  // Define service names to process
  Services[0] := 'savt-client.savt-client-worker';
  Services[1] := 'sav-client.sav-client-worker';
  Services[2] := 'savt-client.savt-client-upgrader';

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

    // Clean ProgramData directory but keep durable device.id for reinstall.
    CleanProgramDataFolderKeepDeviceID('savt-client');
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


