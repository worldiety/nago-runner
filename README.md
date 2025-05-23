# Nago Runner

A Nago Runner runs on a virtual or physical linux machine and is registered with the _worldiety hub_ respective _nago app console_.
The machine can be private; however, the runner opens a channel to the hub to receive hub specific control commands.
These commands usually contain idempotent configurations for applications, which must be build and run in isolation.

## pre-requisite

Currently, only Ubuntu 24.04 LTS is supported. Other debian based systems using systemd may work as well. 

## install

Installing a runner on a new machine is as simple as the following 3 steps on a bare machine:

```bash
# Install whatever go version is available, it bootstraps any other required go compiler versions
sudo apt install golang -y

# Install and configure the nago runner as a systemd service. 
# Keep url and token empty to just apply an update
sudo go run github.com/worldiety/nago-runner/cmd/nago-runner-install@latest -url=http://localhost:3000 -token=1234 
```
