<h1 align="center">
  <a href="https://github.com/dec0dOS/amazing-github-template">
    <img src="icon.ico" alt="Logo" width="250" height="250">
  </a>
</h1>

<div align="center">
  SAV-T client is an active measurement tool which attempts to send and receive a series of spoofed UDP packets to/from servers distributed throughout the world. 
  <br />
  <br />

  <a href="https://github.com/KI3-Team/savt-client/issues/new?assignees=&labels=bug&template=01_BUG_REPORT.md&title=bug%3A+">Report a Bug</a>
  ·
  <a href="https://github.com/KI3-Team/savt-client/issues/new?assignees=&labels=enhancement&template=02_FEATURE_REQUEST.md&title=feat%3A+">Request a Feature</a>
  .
  <a href="https://ki3.org.cn/#/contact">Contact Us</a>
</div>

<div align="center">
<br />

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
<a href="https://github.com/KI3-Team/savt-client/stargazers"><img alt="GitHub stars" src="https://img.shields.io/github/stars/KI3-Team/savt-client?style=social?style=social"></a>

<a href="https://github.com/KI3-Team/savt-client/releases"><img alt="GitHub release (latest by date)" src="https://img.shields.io/github/v/release/KI3-Team/savt-client"></a>

</div>


## About

SAV-T client is an active measurement tool which attempts to send and receive a series of spoofed UDP packets to/from servers distributed throughout the world. We use SAV-T clients to test a network’s ability of outbound or inbound spoofing periodically.

**By participating and running SAV-T**, you help build a global dataset of source address validation capabilities, contributing to Internet security research and infrastructure protection. The collected results are anonymously visualized at [KI3 SAV-T Results Dashboard](https://ki3.org.cn/#/sav?sub=savTest&children=recentTestResults).

<!-- Thank you for your interest in participating in our SAV-T measurement Task (hereinafter referred to as the “Task”). This Informed Consent Agreement (hereinafter referred to as the “Agreement”) is designed to inform you about the nature of the Task, its purpose, the procedures involved, potential risks and benefits, and your rights as a participant. Please read this Agreement carefully before deciding whether to participate in the Task.

**Purpose of the Task**: The purpose of the Task is to collect IP spoofing status of the Internet. Your participation in the Task will contribute to understand and minimize the Internet’s vulnerability to spoofed-source attacks, thus better protecting the Internet.

**Confidentiality and Data Protection**: Your participation in the Task is confidential. Your IP address will not be published in any form. Your personal information will not be collected or shared with any third parties. The data collected will be stored securely and will be used only for the purposes of the Research. 

**Features of SAV-T Software**: The SAV-T measurement platform includes multiple well-designed functional modules to ensure accurate and comprehensive evaluation of source address validation (SAV) deployments across the Internet.

- **Basic Measurement Module**:  
  This module conducts outbound and inbound SAV tests by crafting and sending a series of spoofed UDP packets between the client and server. It helps determine whether a network can block forged source IP addresses from being sent or received. The module includes fine-grained spoofing capability testing (e.g., common-prefix variation and private address tests) and leverages token-based verification to ensure result integrity and eliminate interference from unrelated traffic.

- **Tracefilter Module**:  
  This module pinpoints where SAV filtering occurs along the path of spoofed packets. It works by comparing normal and spoofed `traceroute` results. If intermediate routers fail to return ICMP replies when the spoofed source is used, the filtering position can be inferred. This helps identify which AS or router deploys SAV on the outbound path.

- **Automatic Measurement Module**:  
  This background service enables long-term monitoring by automatically scheduling and executing SAV tests. It supports periodic execution (e.g., weekly), failure retries, and triggers new measurements when the client’s network changes. This ensures continuous, autonomous operation without manual intervention.

- **GUI Module**:  
  A graphical interface (available for Windows and macOS) allows users to easily manage measurements, configure parameters, view logs in real time, and review historical results. It interacts with the core worker component via gRPC, offering a user-friendly entry point without requiring command-line interaction.

- **Cross-platform Support**:  
  SAV-T is available on Windows, macOS, and Linux, and supports both amd64 and arm64 architectures. It can run as a system service with auto-start on boot, ensuring stable and uninterrupted operation for long-term measurement. 

**Procedures Involved**: Participation in the Task will involve downloading and installing SAV-T software on your computer, running the software and submit your token proof.

**Compensation**: You will be compensated online through Prolific for your participation in the Task. Incomplete or non-compliant submissions will not be eligible for payment.  -->



## Quick Start
- [User Guide](https://ki3.org.cn/#/sav?sub=savTest&children=clientDownload)
- [Developer Guide](savt-client-doc/develop/README.md)

## Contact
- Website: [https://ki3.org.cn/#/contact](https://ki3.org.cn/#/contact)
- Email: [ki3contact@163.com](mailto:ki3contact@163.com)



