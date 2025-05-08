# User Guide

By running the SAV-T client, you can measure the Source Address Validation (SAV) deployment status of your network.

## 1. Dependency Installation

You need to install `pcap` before using the Source Address Validation measurement client.

- **Windows**: You can install it as an additional option when installing the SAV-T client. Or you can download it from [its official website](https://npcap.com/).
- **MacOS**
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

    - Or you can build it from [source](https://github.com/the-tcpdump-group/libpcap).

## 2. Download & Run the Client

### 2.1 Confirm Your Operating System Architecture

| System    | Tool        | Command                        | Example Output |
|-----------|-------------|--------------------------------|----------------|
| Mac       | Shell       | uname -m                      | arm64          |
| Linux     | Shell       | uname -m                      | arm64          |
| Windows   | PowerShell  | $Env:PROCESSOR_ARCHITECTURE    | ARM64          |

### 2.2 Follow the Installation Guide

Refer to the software installation guide page to download, install, and run the client.

**Note**: For the Linux version, please refer to the installation guide below.

```markdown
# install
chmod +x install.sh
sudo ./install.sh install

# uninstall
sudo ./install.sh uninstall

# run a probe
./savt-client-cli prober

# set configure
./savt-client-cli config --enable=false --retrylimit=5

# get help
./savt-client-cli config --help

# check version
./savt-client-cli --version
```



## 3. Uninstalling Older Versions of `SAV-T Runner` (if previously installed on Windows)

- **Method 1**: Control Panel -> Programs -> Installed Programs -> Search for `SAV-T Runner` -> Uninstall.
- **Method 2**: Navigate to `C:\Program Files\SAV-T Runner` directory and run the `unins000` file.

# SAV-T User Acknowledgement

## Project Description
KI3 SAV-T is an active measurement tool designed to test networks' ability to handle spoofed traffic by sending and receiving specially crafted UDP packets through globally distributed servers. The tool performs periodic measurements to evaluate both outbound and inbound spoofing capabilities.

## Collected Information
The tool primarily collects information about whether your client can successfully receive or send spoofed packets. The only network information collected is your public IP address. No additional personal or identifying information is gathered beyond the spoofablitiy of your network.

## Information Disclosure
Users may choose whether to publish test results on our website. When opting for disclosure:

- We **will not** disclose exact IP addresses
- We **will** disclose:
  - /24 IPv4 network prefix (and/or /40 IPv6 network prefix)
  - ASN (Autonomous System Number)
  - Country
  - AS organization
  - RIR (Regional Internet Registry)
  - NAT status
  - Network spoofability status

## ISP Security Considerations

To date, we have received no complaints from ISPs regarding our testing activities. The volume of spoofed packets used in our measurements is intentionally kept minimal to avoid raising concerns from ISP's intrusion detection system.

Users should comply with their ISP's regulations. If the ISP prohibits its use, users should stop running it.

## Screening Your Results
Network operators may find their networks listed on our website because SAV-T operates as a crowdsourced platform where any user within a network can choose to disclose measurement results. If you wish to have your network information removed from our disclosures, please contact us at [ki3_contact@163.com](mailto:ki3_contact@163.com) with your request.

## Data Access for Researchers
Researchers may apply for the dataset at:  
[https://ki3.org.cn/#/datasetRequest](https://ki3.org.cn/#/datasetRequest?dataset=sav%2Fki3_measurement&name=SAV%20Deployment)

### Usage Terms
By using our data, you agree to:
1. **Restrict data sharing** to only:
  - Your institute's employees
  - Direct collaborators assisting with your research
2. **Prohibited activities**:
  - Distributing, disclosing, or transferring data to unauthorized parties
  - Using data for cyber attacks
3. **Privacy protection**:
  - Anonymize all identifiable information (IP addresses, ASNs, etc.) in publications
  - Respect the privacy of potentially identifiable individuals