;----------------------------------------
;  Basic Information
;----------------------------------------
[Setup]
AppName=SAV Client
AppVersion=1.0
DefaultDirName={pf}\savt-client
; Ensure admin privileges are required to install service
PrivilegesRequired=admin

; Generated installer filename
OutputBaseFilename=SavClientInstaller
Compression=lzma
SolidCompression=yes

;----------------------------------------
;  Files to Install
;----------------------------------------
[Files]
; Copy main program savt-client-worker.exe to {app} directory
Source: "path\to\savt-client-worker.exe"; DestDir: "{app}"; Flags: ignoreversion

; Package nssm.exe to installation directory (or your preferred subdirectory)
; Please place nssm.exe in the Source path at the same level as the script
Source: "path\to\nssm.exe"; DestDir: "{app}"; Flags: ignoreversion

;----------------------------------------
;  Post-installation Commands (Create and Configure Service)
;----------------------------------------
[Run]
; 1) Create service: Specify service name "savt-client.savt-client-worker" and make it execute {app}\savt-client-worker.exe
Filename: "{app}\nssm.exe"; \
    Parameters: "install ""savt-client.savt-client-worker"" ""{app}\savt-client-worker.exe"""; \
    StatusMsg: "Registering service (NSSM)..."; \
    Flags: runhidden

; 2) Optional: Set service display name (not required, example only)
Filename: "{app}\nssm.exe"; \
    Parameters: "set ""savt-client.savt-client-worker"" DisplayName ""SAV Client Worker"""; \
    StatusMsg: "Setting service display name..."; \
    Flags: runhidden

; 3) Optional: Set service description (not required, example only)
Filename: "{app}\nssm.exe"; \
    Parameters: "set ""savt-client.savt-client-worker"" Description ""SAV Client Worker Service"""; \
    StatusMsg: "Setting service description..."; \
    Flags: runhidden

; 4) Set service to start automatically at boot
Filename: "{app}\nssm.exe"; \
    Parameters: "set ""savt-client.savt-client-worker"" Start SERVICE_AUTO_START"; \
    StatusMsg: "Setting service to start automatically..."; \
    Flags: runhidden

; 5) Start service
Filename: "net"; \
    Parameters: "start ""savt-client.savt-client-worker"""; \
    StatusMsg: "Starting service..."; \
    Flags: runhidden

;----------------------------------------
;  Uninstall Commands (Remove Service)
;----------------------------------------
[UninstallRun]
; Stop and remove service during uninstallation
Filename: "{app}\nssm.exe"; \
    Parameters: "stop ""savt-client.savt-client-worker"" confirm"; \
    StatusMsg: "Uninstalling service..."; \
    Flags: runhidden

Filename: "{app}\nssm.exe"; \
    Parameters: "remove ""savt-client.savt-client-worker"" confirm"; \
    StatusMsg: "Uninstalling service..."; \
    Flags: runhidden

;----------------------------------------
;  End
;----------------------------------------
