# Expose your TCP application to WEB

You (and maybe only you) can do it right now: expose your fancy application over 
http CONNECT tunnel.

This utility has very simple idea: it brings up http server and route CONNECT requests
to backend over TCP connection

Let me show you possible benefit:

For example we want to connect to our SSH over internet, but we can only do it through 
http connection. 

Solution:

```

+----------+                                    +----------+
|          |         Http proxy                 |          |            +------+
| Client   | -- CONNECT server:22 HTTP/1.1 -->  | http2tcp | -- TCP --> | SSHD |
|          |                                    |          |            +------+
+----------+                                    +----------+

```


You have to ask me: why do this if we have Squid? Answer is simple: because it's much
simple to configure and control access. Remote client can connect only to specified
address

### Installation

Just as always

`go install github.com/reddec/http2tcp`

Sometimes I put binary releases into Releases tab in github, but they maybe outdated

### Usage:

`http2tcp [-b binding] <configuration files...>`

### Configuration:


```
# Yes, this is comment


# ... empty line are allowed ....

# serviceName:port      targetIp:targetPort
# port for service is optional. In fact, this is just Request URI (RFC 2616, Section 5.1) =))

# Example

myssh:22 127.0.0.1:22

# oops, I forget: you can use environment variables as GoLang template keys
# for example, if you have env HOSTNAME, you can use it like this (dummy example)

{{.HOSTNAME}}:22 127.0.0.1:22

# and the last - all powerfull of golang template are here (for-loops, if-else end e.t.c)
```


Goooooooood luck!