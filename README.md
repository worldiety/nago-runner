# Nago Runner

A Nago Runner runs on a virtual or physical linux machine and is registered with the _worldiety hub_ respective _nago app console_.
The machine can be private; however, the runner opens a channel to the hub to receive hub specific control commands.
These commands usually contain idempotent configurations for applications, which must be build and run in isolation.

## pre-requisite

Currently, only Ubuntu 24.04 LTS is supported. Other debian based systems using systemd may work as well. 

## install

Installing a runner on a new machine is as simple as the following 3 steps on a bare machine:

```bash
# install whatever go version is available, it bootstraps any other required go compiler versions
sudo apt install golang -y

# configure the nago runner
sudo go run github.com/worldiety/nago-runner/cmd/nago-runner@latest -url=localhost -token=1234 configure 

# install the nago runner
sudo go run github.com/worldiety/nago-runner/cmd/nago-runner@latest -- install
```
