package svnx

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"project-manage/internal/model"
)

// Client 封装 svn 命令行
type Client struct {
	Exe string
}

// Candidates 返回常见的 svn.exe 位置
func Candidates() []string {
	var list []string
	if v := os.Getenv("SVN_EXE"); v != "" {
		list = append(list, v)
	}
	if p, err := exec.LookPath("svn"); err == nil {
		list = append(list, p)
	}
	programFiles := os.Getenv("ProgramFiles")
	programFilesX86 := os.Getenv("ProgramFiles(x86)")
	localAppData := os.Getenv("LOCALAPPDATA")
	list = append(list,
		filepath.Join(programFiles, "TortoiseSVN", "bin", "svn.exe"),
		filepath.Join(programFilesX86, "TortoiseSVN", "bin", "svn.exe"),
		filepath.Join(localAppData, "Programs", "TortoiseSVN", "bin", "svn.exe"),
		`D:\Program Files\TortoiseSVN\bin\svn.exe`,
		filepath.Join(programFiles, "SlikSvn", "bin", "svn.exe"),
		filepath.Join(programFilesX86, "SlikSvn", "bin", "svn.exe"),
		`C:\ProgramData\chocolatey\bin\svn.exe`,
	)
	return list
}

// Detect 自动探测可用的 svn.exe
func Detect() (string, bool) {
	for _, p := range Candidates() {
		if p == "" {
			continue
		}
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p, true
		}
	}
	return "", false
}

// Version 获取 svn 版本号
func (c *Client) Version() (string, error) {
	out, err := c.run("--version", "--quiet")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func (c *Client) run(args ...string) ([]byte, error) {
	if c.Exe == "" {
		return nil, fmt.Errorf("未找到 svn 命令行工具，请在设置中指定 svn.exe 路径")
	}
	cmd := exec.Command(c.Exe, args...)
	hideWindow(cmd)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return stdout.Bytes(), fmt.Errorf("svn %s 执行失败: %s", args[0], msg)
	}
	return stdout.Bytes(), nil
}

// ---------- info ----------

type infoXML struct {
	Entry struct {
		Revision string `xml:"revision,attr"`
		URL      string `xml:"url"`
		Relative string `xml:"relative-url"`
		Repo     struct {
			Root string `xml:"root"`
			UUID string `xml:"uuid"`
		} `xml:"repository"`
	} `xml:"entry"`
}

// Info 读取工作副本信息
func (c *Client) Info(path string) (model.SvnProjectInfo, error) {
	var info model.SvnProjectInfo
	out, err := c.run("info", "--xml", "--non-interactive", path)
	if err != nil {
		return info, err
	}
	var x infoXML
	if err := xml.Unmarshal(out, &x); err != nil {
		return info, fmt.Errorf("解析 svn info 失败: %w", err)
	}
	info.URL = strings.TrimSpace(x.Entry.URL)
	info.RepoRoot = strings.TrimSpace(x.Entry.Repo.Root)
	info.UUID = strings.TrimSpace(x.Entry.Repo.UUID)
	info.RelativeURL = strings.TrimSpace(strings.TrimPrefix(x.Entry.Relative, "^"))
	if info.RelativeURL == "" {
		info.RelativeURL = strings.TrimPrefix(info.URL, strings.TrimRight(info.RepoRoot, "/"))
	}
	fmt.Sscanf(strings.TrimSpace(x.Entry.Revision), "%d", &info.Revision)
	return info, nil
}

// ---------- log ----------

type logXML struct {
	Entries []struct {
		Revision string `xml:"revision,attr"`
		Author   string `xml:"author"`
		Date     string `xml:"date"`
		Message  string `xml:"msg"`
		Paths    []struct {
			Action string `xml:"action,attr"`
			Kind   string `xml:"kind,attr"`
			Path   string `xml:",chardata"`
		} `xml:"paths>path"`
	} `xml:"logentry"`
}

// Log 读取提交历史
func (c *Client) Log(path string, limit int) ([]model.SvnCommit, error) {
	if limit <= 0 {
		limit = 100
	}
	args := []string{"log", "--xml", "--verbose", "--non-interactive", "-r", "HEAD:1", "-l", fmt.Sprint(limit)}
	if t := strings.TrimSpace(path); t != "" {
		args = append(args, t)
	}
	out, err := c.run(args...)
	if err != nil {
		return nil, err
	}
	var x logXML
	if err := xml.Unmarshal(out, &x); err != nil {
		return nil, fmt.Errorf("解析 svn log 失败: %w", err)
	}
	list := make([]model.SvnCommit, 0, len(x.Entries))
	for _, e := range x.Entries {
		cm := model.SvnCommit{
			Author:  strings.TrimSpace(e.Author),
			Date:    formatSvnDate(e.Date),
			Message: strings.TrimSpace(e.Message),
			Files:   make([]model.SvnChangedFile, 0, len(e.Paths)),
		}
		fmt.Sscanf(strings.TrimSpace(e.Revision), "%d", &cm.Revision)
		for _, p := range e.Paths {
			cm.Files = append(cm.Files, model.SvnChangedFile{
				Path:   strings.TrimSpace(p.Path),
				Action: strings.TrimSpace(p.Action),
				Kind:   strings.TrimSpace(p.Kind),
			})
		}
		list = append(list, cm)
	}
	return list, nil
}

func formatSvnDate(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02T15:04:05.000000Z"} {
		if t, err := time.Parse(layout, raw); err == nil {
			return t.Local().Format("2006-01-02 15:04:05")
		}
	}
	return raw
}

// ---------- export ----------

// RepoURL 把仓库根地址和仓库内路径拼成可访问的 URL
func RepoURL(repoRoot, repoPath string) string {
	repoRoot = strings.TrimRight(strings.TrimSpace(repoRoot), "/")
	repoPath = strings.TrimSpace(repoPath)
	if repoPath == "" {
		return repoRoot
	}
	segs := strings.Split(strings.Trim(repoPath, "/"), "/")
	for i, s := range segs {
		segs[i] = url.PathEscape(s)
	}
	return repoRoot + "/" + strings.Join(segs, "/")
}

// Export 从仓库导出指定版本的文件到 dest
func (c *Client) Export(repoRoot, repoPath string, revision int64, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	if _, err := c.run("export", "--force", "--non-interactive",
		"-r", fmt.Sprint(revision), RepoURL(repoRoot, repoPath), dest); err != nil {
		return err
	}
	return nil
}

// ExportWorkingCopy 备用方案：通过工作副本 peg 版本导出
func (c *Client) ExportWorkingCopy(wcPath string, revision int64, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	src := fmt.Sprintf("%s@%d", wcPath, revision)
	_, err := c.run("export", "--force", "--non-interactive",
		"-r", fmt.Sprint(revision), src, dest)
	return err
}
