# 使用说明

通过运行 SAV-T 客户端，您可以测量所在网络的源地址验证部署情况。

## 1. 依赖安装

您需要首先安装`pcap`才能使用源地址验证部署测量客户端。

- Windows: 您可以选择在安装SAT-T客户端的时候进行附加安装。您也可以在[npcap官网](https://npcap.com/)进行下载。
- MacOS
    ```shell
    brew install libpcap
    ```
- **Linux**
    - Ubuntu/Debian
    ```shell
    apt-get install libpcap0.8
    ```
    
    - RHEL/CentOS/Fedora
    ```shell
    dnf install libpcap
    yum install libpcap
    ```

    - 您也可以从[源码](https://github.com/the-tcpdump-group/libpcap)编译。

## 2. 下载客户端 & 运行

### 2.1 确认自己的操作系统架构

| 系统      | 工具         | 命令                          | 示意    |
|---------|------------|-----------------------------|-------|
| Mac     | Shell      | uname -m                    | arm64 |
| Linux   | Shell      | uname -m                    | arm64 |
| Windows | PowerShell | $Env:PROCESSOR_ARCHITECTURE | ARM64 |

### 2.2 本软件安装指引页里，下载、安装、运行 即可

PS: Linux 版请参考README.md 里的安装指引

### 3. 旧版本`SAV-T Runner`卸载（如果有在`Windows`里安装过）

- 方法一： 控制面板->程序->已安装程序->搜索`SAV-T Runner`->卸载
- 方法二： 进入`C:\Program Files\SAV-T Runner`目录，点击`unins000`文件


