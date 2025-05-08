;----------------------------------------
;  基本信息
;----------------------------------------
[Setup]
AppName=SAV Client
AppVersion=1.0
DefaultDirName={pf}\savt-client
; 确保需要管理员权限来安装服务
PrivilegesRequired=admin

; 生成的安装文件名
OutputBaseFilename=SavClientInstaller
Compression=lzma
SolidCompression=yes

;----------------------------------------
;  要安装的文件列表
;----------------------------------------
[Files]
; 将主程序 savt-client-worker.exe 复制到 {app} 目录
Source: "path\to\savt-client-worker.exe"; DestDir: "{app}"; Flags: ignoreversion

; 将 nssm.exe 也打包到安装目录（或者你喜欢的子目录）
; 请把 nssm.exe 放在与脚本同级的 Source 路径下
Source: "path\to\nssm.exe"; DestDir: "{app}"; Flags: ignoreversion

;----------------------------------------
;  安装后运行命令 (创建并配置服务)
;----------------------------------------
[Run]
; 1) 创建服务：指定服务名 "savt-client.savt-client-worker" 并让它执行 {app}\savt-client-worker.exe
Filename: "{app}\nssm.exe"; \
    Parameters: "install ""savt-client.savt-client-worker"" ""{app}\savt-client-worker.exe"""; \
    StatusMsg: "正在注册服务 (NSSM)..."; \
    Flags: runhidden

; 2) 可选：设置服务显示名 (非必须，仅示例)
Filename: "{app}\nssm.exe"; \
    Parameters: "set ""savt-client.savt-client-worker"" DisplayName ""SAV Client Worker"""; \
    StatusMsg: "设置服务显示名称..."; \
    Flags: runhidden

; 3) 可选：设置服务描述 (非必须，仅示例)
Filename: "{app}\nssm.exe"; \
    Parameters: "set ""savt-client.savt-client-worker"" Description ""SAV Client Worker Service"""; \
    StatusMsg: "设置服务描述..."; \
    Flags: runhidden

; 4) 设置服务为开机自动启动
Filename: "{app}\nssm.exe"; \
    Parameters: "set ""savt-client.savt-client-worker"" Start SERVICE_AUTO_START"; \
    StatusMsg: "设置服务为开机自启动..."; \
    Flags: runhidden

; 5) 启动服务
Filename: "net"; \
    Parameters: "start ""savt-client.savt-client-worker"""; \
    StatusMsg: "正在启动服务..."; \
    Flags: runhidden

;----------------------------------------
;  卸载时运行命令 (删除服务)
;----------------------------------------
[UninstallRun]
; 卸载时先停止并删除服务
Filename: "{app}\nssm.exe"; \
    Parameters: "stop ""savt-client.savt-client-worker"" confirm"; \
    StatusMsg: "正在卸载服务..."; \
    Flags: runhidden

Filename: "{app}\nssm.exe"; \
    Parameters: "remove ""savt-client.savt-client-worker"" confirm"; \
    StatusMsg: "正在卸载服务..."; \
    Flags: runhidden

;----------------------------------------
;  完
;----------------------------------------
