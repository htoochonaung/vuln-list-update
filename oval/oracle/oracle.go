package oracle

import (
	"bytes"
	"compress/bzip2"
	"encoding/xml"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/aquasecurity/vuln-list-update/utils"
	"github.com/spf13/afero"
	"golang.org/x/xerrors"
	pb "gopkg.in/cheggaaa/pb.v1"
)

const (
	ovalDir   = "oval"
	oracleDir = "oracle"
	ovalURL   = "https://linux.oracle.com/security/oval/com.oracle.elsa-all.xml.bz2"
	retry     = 5
)

type Config struct {
	VulnListDir string
	URL         string
	AppFs       afero.Fs
	Retry       int
}

func NewConfig() Config {
	return Config{
		VulnListDir: utils.VulnListDir(),
		URL:         ovalURL,
		AppFs:       afero.NewOsFs(),
		Retry:       retry,
	}
}

func (c Config) Update() error {
	log.Printf("Fetching Oracle")

	res, err := utils.FetchURL(c.URL, "", c.Retry)
	if err != nil {
		return xerrors.Errorf("failed to fetch Oracle Linux OVAL: %w", err)
	}

	ovalReader := bzip2.NewReader(bytes.NewReader(res))

	var ov Oval
	err = xml.NewDecoder(ovalReader).Decode(&ov)
	if err != nil {
		return xerrors.Errorf("failed to decode Oracle Linux OVAL XML: %w", err)
	}

	dir := filepath.Join(ovalDir, oracleDir)
	bar := pb.StartNew(len(ov.Definitions))
	for _, def := range ov.Definitions {
		//def.Title example: "\nELSA-2019-4827:  docker-engine docker-cli security update (IMPORTANT)"
		elsaID := strings.TrimLeft(strings.Split(def.Title, ":")[0], "\n")
		if err = c.saveELSAPerYear(dir, elsaID, def); err != nil {
			return err
		}
		bar.Increment()
	}
	bar.Finish()

	return nil
}

func (c Config) saveELSAPerYear(dirName string, elsaID string, data interface{}) error {
	s := strings.Split(elsaID, "-")
	yearDir := filepath.Join(c.VulnListDir, dirName, s[1])
	if err := os.MkdirAll(yearDir, os.ModePerm); err != nil {
		return err
	}

	filePath := filepath.Join(yearDir, fmt.Sprintf("%s.json", elsaID))

	fs := utils.NewFs(c.AppFs)
	if err := fs.WriteJSON(filePath, data); err != nil {
		return xerrors.Errorf("failed to write file: %w", err)
	}
	return nil
}