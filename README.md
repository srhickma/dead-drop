# dead-drop
Secure anonymous file transfer and storage in the cloud.

# Example Usage
Start server with self-signed TLS certificate:
```
$ mkdir -p ~/.dead-drop/keys
$ openssl req -x509 -nodes -newkey rsa:2048 -keyout ~/.dead-drop/server.key -out ~/.dead-drop/server.crt
$ bin/deadd
```
Generate an rsa key-pair, and copy the public key to the server:
```
$ bin/dead gen-key private.pem public.pem
Wrote private key to private.pem
Wrote public key to public.pem
$ cp public.pem ~/.dead-drop/keys/root
```
Create a secret for local encryption:
```
$ echo 'put your secret here' >> enc.key
```
Drop an object:
```
$ bin/dead drop README.md --private-key private.pem --encryption-key enc.key --key-name root --remote http://localhost:4444 --insecure-skip-verify
WARN: Skipping tls certificate verification, be careful!
Encrypting object with AES-CTR + HMAC-SHA-265 ...
Uploading object ...
Dropped README.md -> nidavyihdlxwbbda#O3vVpwfUHqC2mWPPDIEVekzuKT2IeQ4BeHbkbCYg8lk=
```
Pull the object:
```
$ bin/dead pull nidavyihdlxwbbda#O3vVpwfUHqC2mWPPDIEVekzuKT2IeQ4BeHbkbCYg8lk= dest-file --private-key private.pem --encryption-key enc.key --key-name root --remote http://localhost:4444 --insecure-skip-verify
WARN: Skipping tls certificate verification, be careful!
Downloading object ...
Verifying checksum ...
Decrypting object with AES-CTR + HMAC-SHA-265 ...
Pulled dest-file <- nidavyihdlxwbbda#O3vVpwfUHqC2mWPPDIEVekzuKT2IeQ4BeHbkbCYg8lk=
```
Verify the results:
```
$ diff dest-file README.md
```
NOTE: For simplicity, I have included the `private-key`, `encryption-key`, `key-name`, and `remote` flags in each command, however in general usage these should not change often, so they would be specified in a config file, and the drop/pull commands would be much less verbose.

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
The server provides the api for storing and loading objects, which should be run on some publicly accessible server.
```
Usage:
  deadd [flags]
```
### Configuration
The default config file location is `~/.dead-drop/conf.yml`, but different locations can be specified with the `--config` flag.

All config file fields are optional, and defaults will be used if they are not specified.
The following is the default configuration:
```
# Server configuration
addr: ":4444" # The hostname and port to start the server on.
data-dir: ~/dead-drop # The directory where objects will be stored.
keys-dir: ~/.dead-drop/keys # The directory where authorized rsa public keys should be stored.
tls-cert: ~/.dead-drop/server.crt # The tls certificate for the server.
tls-key: ~/.dead-drop/server.key # The tls key for the server.
ttl-min: 1440 # The number of minutes after which objects will be garbage collected.
destructive-read: true # If true, pulls will destroy objects.
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
Pushes a public key to the authorized-keys directory of the server, so that this key can make authenticated requests to the server.
Of course, this command requires authentication, so the very first (or "root") key will need to be added to the server manually (e.g. via `scp`).
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
All config file fields are optional, however flags may need to be passed from the command line if they are not present in the config file (e.g. `--remote ...` flag if `remote: ...` is not in the config).
The following is an example configuration:
```
# Client configuration
remote: https://localhost:4444 # The address of the server.
private-key: private.pem # The private key to use when authenticating.
encryption-key: encryption.key # The key to use when locally encrypting and decrypting objects.
key-name: root # The name of the authorized-key (public key) to use on the server.
insecure-skip-verify: false # If true, tls certificate verification will be skipped.
```
