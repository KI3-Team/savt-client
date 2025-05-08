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
  // 定义需要处理的服务名称
  Services[0] := 'savt-client.savt-client-worker';
  Services[1] := 'sav-client.sav-client-worker';

  try
    // 遍历服务列表，逐一处理
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

   // 删除 ProgramData 下的目录
    DeleteProgramDataFolder('savt-client');
    DeleteProgramDataFolder('sav-client');
    // 初始化成功
    Log('Initialization completed successfully.');
    Result := True;
  except
    // 捕获所有异常，记录日志并中止安装
    Log('Initialization failed due to an unexpected error.');
    MsgBox('An error occurred during setup initialization. Please check the logs for details.', mbError, MB_OK);
    Result := False; // 中止安装
  end;
end;


procedure InitializeWizard;
var
  ErrorCode: Integer;
begin
  try
    // 检查是否已安装 Npcap
    if not IsNpcapInstalled then
    begin
      MsgBox('Npcap is not installed on your system. The installer will redirect you to the Npcap download page.', mbInformation, MB_OK);
      if not ShellExec('open', 'https://npcap.com', '', '', SW_SHOWNORMAL, ewNoWait, ErrorCode) then
      begin
        Log('Failed to open the Npcap download page. Error code: ' + IntToStr(ErrorCode));
        MsgBox('Failed to open the Npcap download page. Please install Npcap manually and retry.', mbError, MB_OK);
      end;
    end
    else
    begin
      Log('Npcap is already installed. Proceeding with installation...');
    end;
  except
    // 捕获所有异常，记录日志并提示用户
    Log('Error during wizard initialization.');
    MsgBox('An unexpected error occurred during wizard initialization. Please check the logs for details.', mbError, MB_OK);
  end;
end;


