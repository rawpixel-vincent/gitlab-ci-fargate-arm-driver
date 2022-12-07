package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
)

const httpPort = 8888
const sshdPort = 2222
const sshConfig = `
Port %d
ListenAddress 0.0.0.0
HostKey %s
PermitRootLogin yes
PubkeyAuthentication yes
PasswordAuthentication no
ChallengeResponseAuthentication no
AuthorizedKeysFile      .ssh/authorized_keys
AllowTcpForwarding no
GatewayPorts no
X11Forwarding no
PrintMotd no
AcceptEnv LANG LC_*
Subsystem       sftp    /usr/lib/openssh/sftp-server
`
const authorizedKeysFilePath = "/root/.ssh/authorized_keys"
const username = "root"

type Info struct {
	Port           int
	Username       string
	HostPrivateKey string
	HostPublicKey  string
	UserPrivateKey string
	UserPublicKey  string
}

type key struct {
	PrivateKeyPath string
	PublicKeyPath  string

	PrivateKey []byte
	PublicKey  []byte
}

func main() {
	ctx, cancel := getSignalContext()
	defer cancel()

	hostKey, err := generateKey("host")
	if err != nil {
		panic(err)
	}

	userKey, err := generateKey("user")
	if err != nil {
		panic(err)
	}

	configFilePath, err := createConfigFile(hostKey)
	if err != nil {
		panic(err)
	}

	err = writeAuthorizedKeys(userKey)
	if err != nil {
		panic(err)
	}

	run(ctx, configFilePath, hostKey, userKey)
}

func getSignalContext() (context.Context, func()) {
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

		select {
		case sig := <-signals:
			fmt.Printf("Received %v, exitting\n", sig)
			cancel()
		case <-ctx.Done():
		}
	}()

	return ctx, cancel
}

func generateKey(name string) (key, error) {
	fmt.Printf("Generating %s key\n", name)

	privateKeyDir, err := ioutil.TempDir("", "ssh-key")
	if err != nil {
		return key{}, fmt.Errorf("creating key storage directory: %w", err)
	}

	privateKeyFilePath := filepath.Join(privateKeyDir, "id_rsa")
	publicKeyFilePath := privateKeyFilePath + ".pub"

	cmd := exec.Command("ssh-keygen", "-t", "rsa", "-b", "2048", "-N", "", "-f", privateKeyFilePath)
	err = cmd.Run()
	if err != nil {
		return key{}, fmt.Errorf("executing %q command: %w", cmd.String(), err)
	}

	privateKey, err := ioutil.ReadFile(privateKeyFilePath)
	if err != nil {
		return key{}, fmt.Errorf("reading private key file %q: %w", privateKeyFilePath, err)
	}

	publicKey, err := ioutil.ReadFile(publicKeyFilePath)
	if err != nil {
		return key{}, fmt.Errorf("reading public key file %q: %w", publicKeyFilePath, err)
	}

	key := key{
		PrivateKeyPath: privateKeyFilePath,
		PublicKeyPath:  publicKeyFilePath,
		PrivateKey:     privateKey,
		PublicKey:      publicKey,
	}

	return key, nil
}

func createConfigFile(key key) (string, error) {
	fmt.Println("Creating SSHD configuration file")

	fileContent := fmt.Sprintf(sshConfig, sshdPort, key.PrivateKeyPath)

	configDir, err := ioutil.TempDir("", "sshd-config")
	if err != nil {
		return "", fmt.Errorf("creating key storage directory: %w", err)
	}

	configFilePath := filepath.Join(configDir, "sshd_config")
	err = ioutil.WriteFile(configFilePath, []byte(fileContent), 0600)
	if err != nil {
		return "", fmt.Errorf("writing configuration file %q: %w", configFilePath, err)
	}

	return configFilePath, nil
}

func writeAuthorizedKeys(key key) error {
	fmt.Printf("Creating %q file\n", authorizedKeysFilePath)

	authorizedKeysPath := filepath.Dir(authorizedKeysFilePath)
	err := os.MkdirAll(authorizedKeysPath, 0700)
	if err != nil {
		return fmt.Errorf("creating authorized keys directory %q: %w", authorizedKeysPath, err)
	}

	err = ioutil.WriteFile(authorizedKeysFilePath, key.PublicKey, 0600)
	if err != nil {
		return fmt.Errorf("writing authorized keys file %q: %w", authorizedKeysFilePath, err)
	}

	return nil
}

func run(ctx context.Context, configFilePath string, hostKey key, userKey key) {
	sshWait := startSSHServer(ctx, configFilePath)
	httpWait := startHTTPServer(ctx, serveConfig(hostKey, userKey))

	select {
	case err := <-sshWait:
		panic(fmt.Sprintf("ssh server exited with error: %v", err))
	case err := <-httpWait:
		panic(fmt.Sprintf("http server exited with error: %v", err))
	case <-ctx.Done():
	}
}

func startSSHServer(ctx context.Context, configFilePath string) chan error {
	fmt.Printf("Starting SSH server with %q configuration file at port %d\n", configFilePath, sshdPort)

	wait := make(chan error)

	go func() {
		cmd := exec.CommandContext(ctx, "/usr/sbin/sshd", "-D", "-f", configFilePath)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		wait <- cmd.Run()
	}()

	return wait
}

func startHTTPServer(ctx context.Context, handler http.Handler) chan error {
	addr := fmt.Sprintf(":%d", httpPort)

	fmt.Printf("Starting HTTP server at %s\n", addr)

	wait := make(chan error)

	s := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	go func() {
		<-ctx.Done()
		err := s.Close()
		if err != nil {
			wait <- fmt.Errorf("HTTP server graceful shutdown: %w", err)
		}
	}()

	go func() {
		err := s.ListenAndServe()
		if err != nil {
			wait <- fmt.Errorf("HTTP server listener: %w", err)
		}
	}()

	return wait
}

func serveConfig(hostKey key, userKey key) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		sshInfo := Info{
			Username:       username,
			Port:           sshdPort,
			HostPrivateKey: string(hostKey.PrivateKey),
			HostPublicKey:  string(hostKey.PublicKey),
			UserPrivateKey: string(userKey.PrivateKey),
			UserPublicKey:  string(userKey.PublicKey),
		}

		encoder := json.NewEncoder(rw)
		err := encoder.Encode(sshInfo)
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			_, _ = fmt.Fprintf(rw, "couldn't encode response: %v", err)
			return
		}
	}
}
