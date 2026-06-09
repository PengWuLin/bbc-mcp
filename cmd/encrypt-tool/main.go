package main

import (
	"fmt"
	"os"

	"bbc-mcp/internal/crypto"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "encrypt":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "用法: encrypt-tool encrypt <明文密码>")
			os.Exit(1)
		}
		runEncrypt(os.Args[2])

	case "decrypt":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "用法: encrypt-tool decrypt <密文>")
			os.Exit(1)
		}
		runDecrypt(os.Args[2])

	case "genkey":
		runGenKey()

	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, `用法:
  encrypt-tool encrypt <明文密码>    使用 BBC_MCP_KEY 环境变量加密密码
  encrypt-tool decrypt <密文>        使用 BBC_MCP_KEY 环境变量解密密码
  encrypt-tool genkey                生成随机主密钥`)
}

func runEncrypt(plaintext string) {
	key, err := crypto.GetMasterKey()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	encrypted, err := crypto.Encrypt(plaintext, key)
	if err != nil {
		fmt.Fprintf(os.Stderr, "加密失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(encrypted)
}

func runDecrypt(encoded string) {
	key, err := crypto.GetMasterKey()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	plaintext, err := crypto.Decrypt(encoded, key)
	if err != nil {
		fmt.Fprintf(os.Stderr, "解密失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(plaintext)
}

func runGenKey() {
	key, err := crypto.GenerateKey()
	if err != nil {
		fmt.Fprintf(os.Stderr, "生成密钥失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(key)
}
