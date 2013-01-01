# Terminology

* a Client is a process initiated by the user requesting zooms to run a command.

* the Master is the Go program which mediates all the interaction between the other processes

* a Slave is a process managed by Zooms which is used to load dependencies for commands

* a Command process is one forked from a Slave and connected to a Client
