# process-output-socket

- golang app
- listens on tcp port 25560
- upon first connection launches a process and all stdout and stderr are broadcast to all clients in realtime
- currently it launches a program that sends a timestamp 4 times to stdout
- this stdout is broadcast to all connected clients
- after the process finishes if a new client connects the process is launched again
- clients can send bytes followed by a newline char (enter key) and will be broadcast to all other connected clients

# The PIZZA handshake protocol
- This application uses the PIZZA handshake protocol
- RFC Pending
- Upon new clients connecting they are presented with a PI
- Within 4 seconds they must respond with a ZZ otherwise the connection is closed
- The server will acknowledge with a PIZZA! and begin to send output to the client
