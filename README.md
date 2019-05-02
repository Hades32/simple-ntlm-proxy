# Simple NTLM ProxyCommand for SSH

This tool allows you to easily use SSH behind a corporate NTLM firewall, *without providing any credentials*.

## Requirements

* Windows (sorry, the lib that magically gets your credentials is Windows-only)
* corporate proxy address
* another (https) proxy that allows `CONNECT` to any host:port

## Configuration 

Your `~/.ssh/config` could look like this then: 

```
Host github.com
  User git
  ProxyCommand /c/some/where/simple-ntlm-proxy -dest %h:%p -corp-proxy "some-proxy.corpintra.net:3128" -hop-proxy "https://user:pass@some.proxy.de:443"
```
