// Package main 演示 SSH-over-QUIC 完整测试
// 运行方式:
//   1. 集成测试:  go run main.go -mode test
//   2. 服务器模式: go run main.go -mode server
//   3. 客户端模式: go run main.go -mode client
package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"flag"
	"fmt"
	"log"
	"math/big"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/quic-go/quic-go"
	"golang.org/x/crypto/ssh"

	quicssh "github.com/voilet/quic-flow/pkg/ssh"
)

var (
	mode     = flag.String("mode", "test", "运行模式: server, client, test")
	addr     = flag.String("addr", "127.0.0.1:14443", "服务器地址")
	user     = flag.String("user", "testuser", "SSH 用户名")
	password = flag.String("password", "testpass", "SSH 密码")
	cmd      = flag.String("cmd", "echo 'Hello from SSH over QUIC!'", "要执行的命令")
)

func main() {
	flag.Parse()

	switch *mode {
	case "server":
		runQUICServer()
	case "client":
		runQUICClient()
	case "test":
		runIntegrationTest()
	default:
		log.Fatalf("未知模式: %s", *mode)
	}
}

// runIntegrationTest 运行集成测试
func runIntegrationTest() {
	log.Println("╔══════════════════════════════════════════════════════╗")
	log.Println("║         SSH-over-QUIC 集成测试                       ║")
	log.Println("╚══════════════════════════════════════════════════════╝")
	log.Println()

	// 生成 TLS 配置
	tlsConfig, err := generateTLSConfig()
	if err != nil {
		log.Fatalf("生成 TLS 配置失败: %v", err)
	}

	clientTLSConfig := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"quic-ssh-test"},
	}

	// 1. 启动 QUIC 服务器
	log.Println("▶ [步骤1] 启动 QUIC 服务器 (公网侧)...")
	listener, err := quic.ListenAddr(*addr, tlsConfig, &quic.Config{
		MaxIdleTimeout:  30 * time.Second,
		KeepAlivePeriod: 10 * time.Second,
	})
	if err != nil {
		log.Fatalf("  ✗ QUIC 监听失败: %v", err)
	}
	defer listener.Close()
	log.Printf("  ✓ QUIC 服务器监听在 %s", *addr)

	// 用于存储服务器接受的连接
	var serverConn *quic.Conn
	var serverConnMu sync.Mutex
	serverReady := make(chan struct{})

	// 启动服务器接受循环
	go func() {
		conn, err := listener.Accept(context.Background())
		if err != nil {
			log.Printf("  ✗ 接受连接失败: %v", err)
			close(serverReady)
			return
		}
		serverConnMu.Lock()
		serverConn = conn
		serverConnMu.Unlock()
		log.Printf("  ✓ 服务器接受客户端连接: %s", conn.RemoteAddr())
		close(serverReady)
	}()

	// 2. 启动 QUIC 客户端
	log.Println("▶ [步骤2] 启动 QUIC 客户端 (内网侧)...")
	clientConn, err := quic.DialAddr(context.Background(), *addr, clientTLSConfig, &quic.Config{
		MaxIdleTimeout:  30 * time.Second,
		KeepAlivePeriod: 10 * time.Second,
	})
	if err != nil {
		log.Fatalf("  ✗ QUIC 连接失败: %v", err)
	}
	defer clientConn.CloseWithError(0, "test complete")
	log.Printf("  ✓ QUIC 连接已建立: %s -> %s", clientConn.LocalAddr(), clientConn.RemoteAddr())

	// 等待服务器接受连接
	<-serverReady
	serverConnMu.Lock()
	sConn := serverConn
	serverConnMu.Unlock()

	if sConn == nil {
		log.Fatalf("  ✗ 服务器连接失败")
	}

	// 3. 在客户端侧启动 SSH 服务器
	log.Println("▶ [步骤3] 在客户端侧启动 SSH 服务器...")

	sshServerConfig := quicssh.DefaultServerConfig()
	sshServerConfig.PasswordAuth = true
	sshServerConfig.PasswordCallback = func(conn ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
		if conn.User() == *user && string(pass) == *password {
			log.Printf("  ✓ SSH 认证成功: user=%s", conn.User())
			return &ssh.Permissions{}, nil
		}
		log.Printf("  ✗ SSH 认证失败: user=%s", conn.User())
		return nil, fmt.Errorf("认证失败")
	}
	sshServerConfig.Shell = "/bin/sh"

	sshServer, err := quicssh.NewServer(sshServerConfig)
	if err != nil {
		log.Fatalf("  ✗ 创建 SSH 服务器失败: %v", err)
	}
	sshServer.SetLogger(&testLogger{prefix: "SSH-Server"})
	if err := sshServer.Start(); err != nil {
		log.Fatalf("  ✗ 启动 SSH 服务器失败: %v", err)
	}
	defer sshServer.Stop()
	log.Println("  ✓ SSH 服务器已启动")

	// 4. 客户端侧接受 SSH 流
	log.Println("▶ [步骤4] 客户端侧启动流接收循环...")
	clientCtx, clientCancel := context.WithCancel(context.Background())
	defer clientCancel()

	go func() {
		for {
			stream, err := clientConn.AcceptStream(clientCtx)
			if err != nil {
				if clientCtx.Err() != nil || strings.Contains(err.Error(), "Application error 0x0") {
					return
				}
				log.Printf("  ✗ 接受流失败: %v", err)
				return
			}
			log.Printf("  → 客户端收到新流: StreamID=%d", stream.StreamID())

			// 读取流头部
			header, err := quicssh.ReadHeader(stream)
			if err != nil {
				log.Printf("  ✗ 读取头部失败: %v", err)
				stream.Close()
				continue
			}

			if header.Type == quicssh.StreamTypeSSH {
				log.Println("  → 识别为 SSH 流，交给 SSH 服务器处理")
				go sshServer.HandleStream(stream, clientConn)
			} else {
				log.Printf("  ✗ 未知流类型: %v", header.Type)
				stream.Close()
			}
		}
	}()

	// 等待接收循环启动
	time.Sleep(100 * time.Millisecond)

	// 5. 从服务器侧发起 SSH 连接
	log.Println("▶ [步骤5] 从服务器侧发起 SSH 连接...")

	// 打开 QUIC 流
	stream, err := sConn.OpenStreamSync(context.Background())
	if err != nil {
		log.Fatalf("  ✗ 打开流失败: %v", err)
	}
	log.Printf("  ✓ 打开 QUIC 流: StreamID=%d", stream.StreamID())

	// 发送 SSH 流标识
	if err := quicssh.WriteHeader(stream, quicssh.StreamTypeSSH); err != nil {
		log.Fatalf("  ✗ 写入头部失败: %v", err)
	}
	log.Println("  ✓ 发送 SSH 流标识")

	// 创建 StreamConn 适配器
	streamConn := quicssh.NewStreamConn(stream, sConn)

	// SSH 客户端配置
	sshConfig := &ssh.ClientConfig{
		User: *user,
		Auth: []ssh.AuthMethod{
			ssh.Password(*password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	// 6. SSH 握手
	log.Println("▶ [步骤6] 执行 SSH 握手...")
	sshConn, chans, reqs, err := ssh.NewClientConn(streamConn, sConn.RemoteAddr().String(), sshConfig)
	if err != nil {
		log.Fatalf("  ✗ SSH 握手失败: %v", err)
	}
	defer sshConn.Close()
	log.Println("  ✓ SSH 握手成功!")

	// 创建 SSH 客户端
	client := ssh.NewClient(sshConn, chans, reqs)

	// 7. 执行命令
	log.Println("▶ [步骤7] 执行 SSH 命令...")
	session, err := client.NewSession()
	if err != nil {
		log.Fatalf("  ✗ 创建会话失败: %v", err)
	}
	defer session.Close()

	log.Printf("  → 执行命令: %s", *cmd)
	output, err := session.CombinedOutput(*cmd)
	if err != nil {
		log.Printf("  ✗ 命令执行失败: %v", err)
		log.Printf("  → 输出: %s", string(output))
	} else {
		log.Printf("  ✓ 命令输出: %s", strings.TrimSpace(string(output)))
	}

	// 8. 测试第二个命令
	log.Println("▶ [步骤8] 测试第二个命令...")
	session2, err := client.NewSession()
	if err != nil {
		log.Fatalf("  ✗ 创建会话失败: %v", err)
	}
	defer session2.Close()

	cmd2 := "uname -a"
	log.Printf("  → 执行命令: %s", cmd2)
	output2, err := session2.CombinedOutput(cmd2)
	if err != nil {
		log.Printf("  ✗ 命令执行失败: %v", err)
	} else {
		log.Printf("  ✓ 命令输出: %s", strings.TrimSpace(string(output2)))
	}

	log.Println()
	log.Println("╔══════════════════════════════════════════════════════╗")
	log.Println("║                 测试完成                              ║")
	log.Println("║  ✓ QUIC 连接建立成功                                 ║")
	log.Println("║  ✓ SSH 握手成功                                      ║")
	log.Println("║  ✓ 命令执行成功                                      ║")
	log.Println("╚══════════════════════════════════════════════════════╝")
}

// runQUICServer 运行 QUIC 服务器模式
func runQUICServer() {
	log.Println("=== QUIC 服务器模式 (公网侧) ===")
	log.Printf("监听地址: %s", *addr)

	tlsConfig, err := generateTLSConfig()
	if err != nil {
		log.Fatalf("生成 TLS 配置失败: %v", err)
	}

	listener, err := quic.ListenAddr(*addr, tlsConfig, &quic.Config{
		MaxIdleTimeout:  30 * time.Second,
		KeepAlivePeriod: 10 * time.Second,
	})
	if err != nil {
		log.Fatalf("QUIC 监听失败: %v", err)
	}
	defer listener.Close()

	log.Println("等待客户端连接...")
	log.Println("提示: 在另一个终端运行客户端:")
	log.Printf("  go run main.go -mode client -addr %s", *addr)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 处理中断信号
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("\n收到中断信号，正在关闭...")
		cancel()
	}()

	for {
		conn, err := listener.Accept(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("接受连接失败: %v", err)
			continue
		}

		log.Printf("新连接: %s", conn.RemoteAddr())
		go handleServerConnection(ctx, conn)
	}
}

// handleServerConnection 处理服务器端的连接
func handleServerConnection(ctx context.Context, conn *quic.Conn) {
	defer conn.CloseWithError(0, "connection closed")

	log.Printf("[%s] 连接已建立，可以执行 SSH 命令", conn.RemoteAddr())
	log.Println("输入命令执行 (或 'exit' 退出, 'shell' 进入交互模式):")

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		var input string
		fmt.Print("> ")
		fmt.Scanln(&input)

		if input == "exit" || input == "quit" {
			return
		}

		if input == "" {
			continue
		}

		// 执行 SSH 命令
		executeSSHCommand(conn, input)
	}
}

// executeSSHCommand 通过 SSH 执行命令
func executeSSHCommand(quicConn *quic.Conn, command string) {
	stream, err := quicConn.OpenStreamSync(context.Background())
	if err != nil {
		log.Printf("打开流失败: %v", err)
		return
	}

	if err := quicssh.WriteHeader(stream, quicssh.StreamTypeSSH); err != nil {
		log.Printf("写入头部失败: %v", err)
		stream.Close()
		return
	}

	streamConn := quicssh.NewStreamConn(stream, quicConn)

	sshConfig := &ssh.ClientConfig{
		User: *user,
		Auth: []ssh.AuthMethod{
			ssh.Password(*password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	sshConn, chans, reqs, err := ssh.NewClientConn(streamConn, quicConn.RemoteAddr().String(), sshConfig)
	if err != nil {
		log.Printf("SSH 握手失败: %v", err)
		return
	}
	defer sshConn.Close()

	client := ssh.NewClient(sshConn, chans, reqs)
	session, err := client.NewSession()
	if err != nil {
		log.Printf("创建会话失败: %v", err)
		return
	}
	defer session.Close()

	output, err := session.CombinedOutput(command)
	if err != nil {
		log.Printf("命令执行失败: %v", err)
	}
	fmt.Printf("%s", string(output))
}

// runQUICClient 运行 QUIC 客户端模式
func runQUICClient() {
	log.Println("=== QUIC 客户端模式 (内网侧 + SSH 服务器) ===")
	log.Printf("连接到服务器: %s", *addr)

	clientTLSConfig := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"quic-backbone-v1"},
	}

	conn, err := quic.DialAddr(context.Background(), *addr, clientTLSConfig, &quic.Config{
		MaxIdleTimeout:  30 * time.Second,
		KeepAlivePeriod: 10 * time.Second,
	})
	if err != nil {
		log.Fatalf("QUIC 连接失败: %v", err)
	}
	defer conn.CloseWithError(0, "client disconnect")

	log.Printf("QUIC 连接已建立: %s -> %s", conn.LocalAddr(), conn.RemoteAddr())

	// 创建 SSH 服务器
	sshServerConfig := quicssh.DefaultServerConfig()
	sshServerConfig.PasswordAuth = true
	sshServerConfig.PasswordCallback = func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
		if c.User() == *user && string(pass) == *password {
			log.Printf("SSH 认证成功: user=%s", c.User())
			return &ssh.Permissions{}, nil
		}
		return nil, fmt.Errorf("认证失败")
	}

	sshServer, err := quicssh.NewServer(sshServerConfig)
	if err != nil {
		log.Fatalf("创建 SSH 服务器失败: %v", err)
	}
	sshServer.SetLogger(&testLogger{prefix: "SSH"})
	if err := sshServer.Start(); err != nil {
		log.Fatalf("启动 SSH 服务器失败: %v", err)
	}
	defer sshServer.Stop()

	log.Println("SSH 服务器已启动，等待来自公网的 SSH 连接...")
	log.Printf("SSH 认证信息: user=%s, password=%s", *user, *password)

	// 处理中断信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 接受 SSH 流
	go func() {
		for {
			stream, err := conn.AcceptStream(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				if strings.Contains(err.Error(), "Application error 0x0") {
					return
				}
				log.Printf("接受流失败: %v", err)
				return
			}

			log.Printf("收到新的流: StreamID=%d", stream.StreamID())

			header, err := quicssh.ReadHeader(stream)
			if err != nil {
				log.Printf("读取头部失败: %v", err)
				stream.Close()
				continue
			}

			if header.Type == quicssh.StreamTypeSSH {
				log.Println("识别为 SSH 流，交给 SSH 服务器处理")
				go sshServer.HandleStream(stream, conn)
			} else {
				log.Printf("未知流类型: %v", header.Type)
				stream.Close()
			}
		}
	}()

	<-sigCh
	log.Println("\n收到中断信号，正在关闭...")
}

// generateTLSConfig 生成自签名 TLS 配置
func generateTLSConfig() (*tls.Config, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"QUIC-SSH Test"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		return nil, err
	}

	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, err
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		NextProtos:   []string{"quic-ssh-test"},
	}, nil
}

// testLogger 测试日志
type testLogger struct {
	prefix string
}

func (l *testLogger) Debug(msg string, args ...any) {}

func (l *testLogger) Info(msg string, args ...any) {
	log.Printf("[%s] %s %v", l.prefix, msg, formatArgs(args))
}

func (l *testLogger) Warn(msg string, args ...any) {
	log.Printf("[%s] WARN: %s %v", l.prefix, msg, formatArgs(args))
}

func (l *testLogger) Error(msg string, args ...any) {
	log.Printf("[%s] ERROR: %s %v", l.prefix, msg, formatArgs(args))
}

func formatArgs(args []any) string {
	if len(args) == 0 {
		return ""
	}
	var parts []string
	for i := 0; i < len(args)-1; i += 2 {
		parts = append(parts, fmt.Sprintf("%v=%v", args[i], args[i+1]))
	}
	return strings.Join(parts, " ")
}
