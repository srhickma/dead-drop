# dead-drop
Secure anonymous file transfer and storage in the cloud.

# Building
Install some dependencies:
```
$ go get -u github.com/jteeuwen/go-bindata/...
```

Build the client (`bin/dead`) and server (`bin/deadd`) binaries:
```
$ make build
```

# Server
The server provides the api for storing and loading objects, which should be run on some publically acessible server.
```
Usage:
  deadd [flags]
```
### Configuration
The default config file location is `~/.dead-drop/conf.yml`, but different locations can be specified with the `--config` flag.

All config file fields are optional, and defaults will be used if they are not specified. The following is the default configuration:
```
# Server configuration
addr: ":4444" # The hostname and port to start the server on.
data_dir: ~/dead-drop # The directory where objects will be stored.
keys_dir: ~/.dead-drop/keys # The directory where authorized rsa public keys should be stored.
```

# Client
The client is a cli application which serves as a local wrapper around the server api, making it easier for clients to use the api, generate authentication keys, etc.
### Subcommands
#### `drop`
Pushes a local object to remote, and prints its remote oid.
```
Usage:
  dead drop <file path> [flags]
```
#### `pull`
Fetches a remote object by its oid, and saves it locally.
```
Usage:
  dead pull <oid> <destination path> [flags]
```
#### `add-key`
Pushes a public key to the authorized-keys directory of the server, so that this key can make authenticated requests to the server. Of course, this command requires authentication, so the very first (or "root") key will need to be added to the server manually (e.g. via `scp`).
```
Usage:
  dead add-key <public key path> <key name> [flags]
```
#### `gen-key`
Generates a new private and public key pair, for use authenticating requests with the server.
```
Usage:
  dead gen-key <private key path> <public key path> [flags]
```
### Configuration
The default config file location is `~/.dead-drop/conf.yml`, but different locations can be specified with the `--config` flag.
All config file fields are optional, however flags may need to be passed from the command line if they are not present in the config file (e.g. `--remote ...` flag if `remote: ...` is not in the config). The following is an example configuration:
```
# Client configuration
remote: http://localhost:4444 # The address of the server.
private-key: private.pem # The private key to use when authenticating.
key-name: root # The name of the authorized-key (public key) to use on the server.
```
