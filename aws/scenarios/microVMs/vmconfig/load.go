package vmconfig

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"regexp"
)

func LoadConfigFile(filename string) (*Config, error) {
	cfg, err := loadFile(filename)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func validateCustomRecipe(vmset *VMSet) error {
	for _, kernel := range vmset.Kernels {
		if kernel.ImageSource != "" {
			return errors.New("cannot have source for custom kernels")
		}
	}

	if vmset.Img.ImageName == "" || vmset.Img.ImageSourceURI == "" {
		return errors.New("image needed for custom recipe")
	}

	return nil
}

func validateDistroRecipe(vmset *VMSet) error {
	for _, kernel := range vmset.Kernels {
		if kernel.ImageSource == "" {
			return errors.New("source required for distribution kernels")
		}
	}

	if vmset.Img.ImageName != "" || vmset.Img.ImageSourceURI != "" {
		return errors.New("cannot use global image for distribution kernels")
	}

	return nil
}

func defaultValues() *Config {
	return &Config{
		SSHUser: "root",
		Workdir: "/root",
	}
}

func loadFile(filename string) (*Config, error) {
	cfg := defaultValues()
	if filename == "" {
		return nil, fmt.Errorf("loadFile: no config file specified")
	}
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("loadFile: failed to read config file: %w", err)
	}
	if err := loadData(data, cfg); err != nil {
		return nil, fmt.Errorf("loadFile: failed to load data: %w", err)
	}
	return cfg, nil
}

func loadData(data []byte, cfg interface{}) error {
	// Remove comment lines starting with #.
	data = regexp.MustCompile(`(^|\n)\s*#[^\n]*`).ReplaceAll(data, nil)
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(cfg); err != nil {
		return fmt.Errorf("loadData: failed to parse config file: %w", err)
	}
	return nil
}
