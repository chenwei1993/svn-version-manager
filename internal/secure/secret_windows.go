//go:build windows

package secure

import (
	"encoding/base64"
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

const cryptProtectUIForbidden = 0x1

// Encrypt 使用 Windows DPAPI 加密字符串（仅当前 Windows 用户可解密）
func Encrypt(plain string) (string, error) {
	if plain == "" {
		return "", nil
	}
	in := []byte(plain)
	inBlob := windows.DataBlob{Size: uint32(len(in)), Data: &in[0]}
	var out windows.DataBlob
	if err := windows.CryptProtectData(&inBlob, nil, nil, 0, nil, cryptProtectUIForbidden, &out); err != nil {
		return "", fmt.Errorf("加密密码失败: %w", err)
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(out.Data)))
	data := unsafe.Slice(out.Data, out.Size)
	return base64.StdEncoding.EncodeToString(data), nil
}

// Decrypt 解密由 Encrypt 生成的密文
func Decrypt(encoded string) (string, error) {
	if encoded == "" {
		return "", nil
	}
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("解析密码失败: %w", err)
	}
	inBlob := windows.DataBlob{Size: uint32(len(raw)), Data: &raw[0]}
	var out windows.DataBlob
	if err := windows.CryptUnprotectData(&inBlob, nil, nil, 0, nil, cryptProtectUIForbidden, &out); err != nil {
		return "", fmt.Errorf("解密密码失败: %w", err)
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(out.Data)))
	data := unsafe.Slice(out.Data, out.Size)
	return string(data), nil
}
