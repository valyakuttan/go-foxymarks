package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/valyakuttan/foxymarks/internal/crypto"
	"github.com/valyakuttan/foxymarks/internal/term"

	"github.com/valyakuttan/foxymarks/internal/config"
)

const (
	BookmarksFile = "places.sqlite"
	MarkerFile    = ".foxymarks"
	ConfigFile    = "config.json"
	ConfigDir     = "$HOME/.config/foxymarks"
)

const usage = `Usage:
    foxymarks <command> [arguments]

The commands are:

	init
	add
	remove
	list
	save
	restore

Example:

	$ foxymarks list
	List config details

	$ foxymarks init <path to repo>
	Initialize bookmarks repo at path

	$ foxymarks add firefox <path to firefox profile>
	Add firefox to source list

	$ foxymarks remove firefox
	Remove firefox from source list

	$ foxymarks save
	backup bookmarks of all sources

	$ foxymarks save firefox
	backup bookmarks of firefox

	$ foxymarks restore
	restore bookmarks from backups for all sources

	$ foxymarks restore firefox
	restore firefox bookmarks from backup

`

type Source = config.Source

func main() {
	flag.Usage = func() { fmt.Fprintf(os.Stderr, "%s", usage) }

	cmd := "help"
	arg1 := "all"
	var arg2 string

	flag.Parse()
	if flag.NArg() > 3 {
		fmt.Fprintf(os.Stderr, "unknown arguments: %s", strings.Join(flag.Args()[1:], " "))
		os.Exit(0)
	}
	if flag.NArg() > 0 {
		cmd = flag.Arg(0)
	}
	if flag.NArg() > 1 {
		arg1 = flag.Arg(1)
	}
	if flag.NArg() > 2 {
		arg2 = flag.Arg(2)
	}

	switch cmd {
	case "help":
		flag.Usage()

	case "init":
		if repoInitialized() {
			fmt.Fprintf(os.Stderr, "Repo already initialized\n")
			os.Exit(0)
		}

		cfgDir := os.ExpandEnv(ConfigDir)
		if err := os.Mkdir(cfgDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "can not create directory %q\n", cfgDir)
			os.Exit(0)
		}

		var secret string
		prompt := "enter a pass phrase: "
		for {
			first, err := term.ReadSecret(prompt)
			if err != nil {
				fmt.Fprintf(os.Stderr, "can not read secret\n")
				os.Exit(0)
			}

			second, err := term.ReadSecret("enter the pass phrase again: ")
			if err != nil {
				fmt.Fprintf(os.Stderr, "can not read secret\n")
				os.Exit(0)
			}

			if slices.Equal(first, second) {
				secret = string(first)
				break
			}

			prompt = "pass phrases are not matching, " + prompt
		}

		cfg := config.ConfigData{}
		if arg1 == "all" || arg1 == "" {
			fmt.Fprintf(os.Stderr, "path is empty\n")
			os.Exit(0)
		}

		repoPath, err := filepath.Abs(arg1)
		if err != nil {
			fmt.Fprintf(os.Stderr, "unknown path %q \n", arg1)
			os.Exit(0)
		}

		if !pathExists(repoPath) {
			fmt.Fprintf(os.Stderr, "path %q does not exist\n", repoPath)
			os.Exit(0)
		}

		cfg.RepoPath = repoPath

		cfgFile := filepath.Join(cfgDir, ConfigFile)
		config.WriteToConfig(cfgFile, cfg)

		m := bytes.NewBuffer(crypto.RandBytes(256))
		magic, _ := crypto.EncryptData(m, secret)

		markerPath := path.Join(repoPath, MarkerFile)
		out, err := os.Create(markerPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "can not write to %s\n", repoPath)
			os.Exit(0)
		}
		if _, err = out.Write(magic); err != nil {
			fmt.Fprintf(os.Stderr, "can not write to %s\n", repoPath)
			os.Exit(0)
		}
		if err := out.Sync(); err != nil {
			fmt.Fprintf(os.Stderr, "can not write to %s\n", repoPath)
			os.Exit(0)
		}
		fmt.Fprintf(os.Stdout, "bookmarks repo created\n")

	case "add":
		if !repoInitialized() {
			fmt.Fprintf(os.Stderr, "Repo not initialized\n")
			os.Exit(0)
		}

		cfgFile := filepath.Join(os.ExpandEnv(ConfigDir), ConfigFile)
		cfg := config.ReadFromConfig(cfgFile)
		if arg1 != "all" && arg1 != "" && arg2 != "" {
			arg2, err := filepath.Abs(arg2)
			if err != nil {
				fmt.Fprintf(os.Stderr, "unknown path %q \n", arg2)
				os.Exit(0)
			}
			cfg.Sources[arg1] = config.Source{Name: arg1, Path: arg2}
			config.WriteToConfig(cfgFile, cfg)
			fmt.Fprintf(os.Stdout, "source: %s added\n", arg1)
		}

	case "remove":
		if !repoInitialized() {
			fmt.Fprintf(os.Stderr, "Repo not initialized\n")
			os.Exit(0)
		}

		cfgFile := filepath.Join(os.ExpandEnv(ConfigDir), ConfigFile)
		cfg := config.ReadFromConfig(cfgFile)

		if arg1 != "all" && arg1 != "" {
			if _, ok := cfg.Sources[arg1]; ok {
				delete(cfg.Sources, arg1)
				config.WriteToConfig(cfgFile, cfg)
				fmt.Fprintf(os.Stdout, "source: %s removed\n", arg1)
			} else {
				fmt.Fprintf(os.Stderr, "unknows source: %s\n", arg1)
			}
		}

	case "list":
		if !repoInitialized() {
			fmt.Fprintf(os.Stderr, "Repo not initialized\n")
			os.Exit(0)
		}

		cfgFile := filepath.Join(os.ExpandEnv(ConfigDir), ConfigFile)
		cfg := config.ReadFromConfig(cfgFile)

		if cfg.RepoPath != "" {
			fmt.Fprintf(os.Stdout, "repo path:\n %s\n\n", cfg.RepoPath)

		}

		if len(cfg.Sources) > 0 {
			fmt.Fprintf(os.Stdout, "sources:\n")
			for _, s := range cfg.Sources {
				fmt.Fprintf(os.Stdout, "name: %s, path: %q\n", s.Name, s.Path)
			}
		}

	case "save":
		if !repoInitialized() {
			fmt.Fprintf(os.Stderr, "Repo not initialized\n")
			os.Exit(0)
		}

		cfgFile := filepath.Join(os.ExpandEnv(ConfigDir), ConfigFile)
		cfg := config.ReadFromConfig(cfgFile)

		secret, err := term.ReadSecret("enter pass phrase")
		if err != nil {
			fmt.Fprintf(os.Stderr, "can not read secret\n")
			os.Exit(0)
		}

		markerPath := path.Join(cfg.RepoPath, MarkerFile)
		if !passphraseMatches(string(secret), markerPath) {
			fmt.Fprintf(os.Stderr, "pass phrase not matches\n")
			os.Exit(0)
		}

		do(cfg.Sources, arg1, func(s Source) {
			saveBookmarks(s, cfg.RepoPath, string(secret))
		})
	case "restore":
		if !repoInitialized() {
			fmt.Println("repo not initialized")
			os.Exit(0)
		}

		cfgFile := filepath.Join(os.ExpandEnv(ConfigDir), ConfigFile)
		cfg := config.ReadFromConfig(cfgFile)

		secret, err := term.ReadSecret("enter pass phrase")
		if err != nil {
			fmt.Fprintf(os.Stderr, "can not read secret\n")
			os.Exit(0)
		}

		markerPath := path.Join(cfg.RepoPath, MarkerFile)
		if !passphraseMatches(string(secret), markerPath) {
			fmt.Fprintf(os.Stderr, "pass phrase not matches\n")
			os.Exit(0)
		}

		do(cfg.Sources, arg1, func(s Source) {
			restoreBookmarks(s, cfg.RepoPath, string(secret))
		})
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s", strings.Join(flag.Args(), " "))
		flag.Usage()
	}
}

func passphraseMatches(secret string, markerPath string) bool {
	in, _ := os.Open(markerPath)
	if _, err := crypto.DecryptData(in, secret); err != nil {
		return false
	}
	return true
}

func repoInitialized() bool {
	cfgFile := filepath.Join(os.ExpandEnv(ConfigDir), ConfigFile)
	if !pathExists(cfgFile) {
		return false
	}

	cfg := config.ReadFromConfig(cfgFile)
	markerFile := filepath.Join(cfg.RepoPath, MarkerFile)
	return pathExists(markerFile)
}

func pathExists(path string) bool {
	if _, err := os.Stat(path); err != nil {
		return false
	}
	return true
}

func do(sources map[string]Source, arg string, f func(Source)) {
	if arg == "all" {
		for _, s := range sources {
			f(s)
		}
		return
	}
	s, ok := sources[arg]
	if !ok {
		fmt.Fprintf(os.Stderr, "unknown source: %q\n", arg)
		return
	}
	f(s)
}

func restoreBookmarks(source Source, repoPath string, secret string) {
	dstPath := path.Join(source.Path, BookmarksFile)
	srcPath := filepath.Join(repoPath, source.Name+"-bookmarks.sqlite.age")
	fmt.Printf("restoring %s's bookmarks\n", source.Name)
	DecryptFile(srcPath, dstPath, secret)
}

func saveBookmarks(source Source, repoPath string, secret string) {
	srcPath := path.Join(source.Path, BookmarksFile)
	dstName := source.Name + "-bookmarks.sqlite"

	// check whether the srcFile is modified
	if !bookmarksUpdated(srcPath, dstName, repoPath, secret) {
		fmt.Printf("%s's bookmarks are up to date. Nothing to save.\n", source.Name)
		return
	}

	fmt.Printf("%s's bookmarks modified. Saving changes..\n", source.Name)
	encryptBookmarks(srcPath, dstName, repoPath, secret)
}

func bookmarksUpdated(srcPath, dstName, repoPath, secret string) bool {
	dstPath := filepath.Join(repoPath, dstName+".age")
	if _, err := os.Stat(dstPath); err != nil {
		return true
	}

	tmpDir, err := os.MkdirTemp("", "")
	if err != nil {
		fmt.Fprintln(os.Stderr, "can't create temp directory")
		os.Exit(0)
	}
	out := filepath.Join(tmpDir, dstName)
	DecryptFile(dstPath, out, secret)
	return !crypto.HashEqual(srcPath, out)
}

func encryptBookmarks(srcPath, dstName, repoPath, secret string) {
	dstPath := filepath.Join(repoPath, dstName+".age")
	EncryptFile(srcPath, dstPath, secret)
}

func EncryptFile(inFile string, outFile string, secret string) {
	in, err := os.Open(inFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "can not open %q: %s\n", inFile, err)
		os.Exit(0)
	}
	defer in.Close()

	c, err := crypto.EncryptData(in, secret)
	if err != nil {
		fmt.Fprintf(os.Stderr, "encryption failed: %s\n", err)
		os.Exit(0)
	}
	out, err := os.Create(outFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "can not create %q: %s\n", outFile, err)
		os.Exit(0)
	}
	defer out.Close()

	if _, err := out.Write(c); err != nil {
		fmt.Fprintf(os.Stderr, "can not write to %q: %s\n", outFile, err)
		os.Exit(0)
	}

	if err := out.Sync(); err != nil {
		fmt.Fprintf(os.Stderr, "can not write to %q: %s\n", outFile, err)
		os.Exit(0)
	}
}

func DecryptFile(inFile string, outFile string, secret string) {
	in, err := os.Open(inFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "can not open %q: %s\n", inFile, err)
		os.Exit(0)
	}
	defer in.Close()

	data, err := crypto.DecryptData(in, secret)
	if err != nil {
		fmt.Fprintf(os.Stderr, "decryption failed: %s\n", err)
		os.Exit(0)
	}

	out, err := os.Create(outFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "can not create file %q: %s\n", outFile, err)
		os.Exit(0)
	}
	defer out.Close()

	if _, err := out.Write(data); err != nil {
		fmt.Fprintf(os.Stderr, "can not write to %q: %s\n", outFile, err)
		os.Exit(0)
	}

	if err := out.Sync(); err != nil {
		fmt.Fprintf(os.Stderr, "can not write to %q: %s\n", outFile, err)
		os.Exit(0)
	}
}
