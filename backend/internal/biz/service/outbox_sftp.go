package service

import (
	"fmt"
	"net"
	"path"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

func deliverSFTPRemote(ep outboxEndpoint, fileName string, payload []byte) error {
	port := ep.Port
	if port <= 0 {
		port = 22
	}
	if ep.Username == "" {
		return fmt.Errorf("sftp username required")
	}
	timeout := time.Duration(ep.TimeoutSec) * time.Second
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	cfg := &ssh.ClientConfig{
		User:            ep.Username,
		Auth:            []ssh.AuthMethod{ssh.Password(ep.Password)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // 生产应换 known_hosts 校验
		Timeout:         timeout,
	}
	addr := net.JoinHostPort(ep.Host, fmt.Sprintf("%d", port))
	conn, err := ssh.Dial("tcp", addr, cfg)
	if err != nil {
		return fmt.Errorf("sftp dial: %w", err)
	}
	defer conn.Close()

	client, err := sftp.NewClient(conn)
	if err != nil {
		return fmt.Errorf("sftp client: %w", err)
	}
	defer client.Close()

	remoteDir := ep.Path
	if remoteDir == "" {
		remoteDir = "."
	}
	_ = client.MkdirAll(remoteDir)
	remotePath := path.Join(remoteDir, fileName)
	f, err := client.Create(remotePath)
	if err != nil {
		return fmt.Errorf("sftp create %s: %w", remotePath, err)
	}
	defer f.Close()
	if _, err := f.Write(payload); err != nil {
		return fmt.Errorf("sftp write: %w", err)
	}
	return nil
}
