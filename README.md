# dockersshell

Login shell that spawns a docker instance and initiates an SSH connection with it

## /etc/dockersshell.yaml

```yaml
endpoints:
  - "http://127.0.0.1:4243"
image: ssh
user: ubuntu
```
