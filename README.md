# tcp-server-fragmented-packets
Test task: implement a tcp server for fragmented packets


implement a tcp server for fragmented packets
server functions:
1. send message to all clients connected.
2. send message to clients specified (you should tag the client)

implement a client connect to the server

message format is :
| 2 bytes | x  bytes |
| content length | content|

* sever and client use same message format for communication

for example
message1: [00 02 65 66]     => content length: 2 , content : AB
message2: [00 03 65 66 65]   => content length: 3, content: ABA

test case:

10 clients, get broadcast messages and client 1 send message to client 2.

