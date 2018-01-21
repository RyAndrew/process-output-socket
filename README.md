# process-output-socket

- golang app
- listens on tcp port 25500
- send command START
- starts a process
- all stdout & stderr from the process are sent to all listeners
- closes connections on process termination
