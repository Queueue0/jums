package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/BurntSushi/toml"
)

var lock = &sync.Mutex{}

type config struct {
	Domain   string
	Mxdomain string
	BoxesDir string
	CertFile string
	KeyFile  string
}

var confInstance *config

func GetConfig() *config {
	if confInstance == nil {
		lock.Lock()
		defer lock.Unlock()
		if confInstance == nil {
			// initialize config instance
			initializeConfig()
		}
	}

	return confInstance
}

func initializeConfig() {
	configFname := "config.toml"
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		configDir = filepath.Join(os.Getenv("HOME"), ".config")
	}
	configDir = filepath.Join(configDir, "jums")

	// Check for existence of file, create it with initial values if not found
	if _, err := os.Stat(configDir); errors.Is(err, os.ErrNotExist) {
		err = createConfigFile(configDir, configFname)
		if err != nil {
			panic(err)
		}
	}

	confInstance = &config{}
	md, err := toml.DecodeFile(filepath.Join(configDir, configFname), confInstance)
	if err != nil {
		// appropriate situation to panic since we flatly cannot proceed without the config
		panic(err)
	}

	fmt.Printf("%+v\n", md.Keys())
}

func createConfigFile(dir, fname string) error {
	err := os.MkdirAll(dir, os.ModeDir)
	if err != nil {
		return fmt.Errorf("createConfigFile: %w", err)
	}
	cf, err := os.Create(filepath.Join(dir, fname))
	if err != nil {
		return fmt.Errorf("createConfigFile: %w", err)
	}

	c := config{
		Domain: "localhost",
		Mxdomain: "localhost",
		BoxesDir: "~/.jums/mailboxes",
		KeyFile: "",
		CertFile: "",
	}

	err = toml.NewEncoder(cf).Encode(c)
	if err != nil {
		cf.Close()
		return fmt.Errorf("createConfigFile: %w", err)
	}

	if err = cf.Sync(); err != nil {
		return fmt.Errorf("createConfigFile: %w", err)
	}
	if err = cf.Close(); err != nil {
		return fmt.Errorf("createConfigFile: %w", err)
	}

	return nil
}
