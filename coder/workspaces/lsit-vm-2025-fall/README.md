# Dreamlab Coder Workspaces

Coder workspaces are virtual machines (VMs) running a Linux-based operating system on computing infrastructure provided by the [College of Letters and Science IT](https://www.lsit.ucsb.edu/). Anyone with an active UCSB Net ID may create a workspace. 

## Workspace Policies

* **Data stored on the workspace is not backed up.** You are responsible for maintaining backups of your data.  
* All workspaces are automatically deleted at the end of each academic quarter.
* You may create and use **one workspace at a time**. If you have multiple active workspaces, they may be deleted without notice.  
* Workspaces may be inaccessible during scheduled maintenance windows, which will be announced by email.  
* Your workspaces will “stop” (shut off) automatically after **four hours** of inactivity to conserve resources. To resume the workspace, click the “start” button in the top right corner of the workspace page.  
* You may install additional software on the VM as needed.   
* Do not store confidential or sensitive information in your workspace. (Workspace data is accessible by DREAM Lab and LSIT staff).

## Workspace Resources

Each workspace includes the following:

| Resource         | Description                                               |
| :--------------- | :-------------------------------------------------------- |
| CPU              | 4 vCPUs                                                   |
| Memory           | 16 GiB                                                    |
| Disk Storage     | 15 GB for OS and 64 GB for user data (‘/home’ directory)  |
| Operating System | Ubuntu Linux (24.04 LTS)                                  |
| Software         | R (v4.5.1) ; RStudio Server (2025.09.0); Pixi (v 0.55.0). |

You have root access to your workspace VM, allowing you to install software using the `sudo apt install…` command.

## Getting Help

If you have questions or encounter a technical issue, please [submit a ticket](https://github.com/ucsb-dreamlab/infrastructure/issues/new?template=Blank+issue). You may also email [dreamlab@library.ucsb.edu](mailto:dreamlab@library.ucsb.edu).