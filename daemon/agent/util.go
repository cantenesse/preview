package agent

import (
	"bytes"
	"fmt"
	"github.com/ngerakines/preview/common"
	"image"
	"image/jpeg"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var pdfPageCount = regexp.MustCompile(`Pages:\s+(\d+)`)

// pdfinfo ~/Desktop/ChefConf2014schedule.pdf
func getPdfPageCount(file string) (int, error) {
	_, err := exec.LookPath("pdfinfo")
	if err != nil {
		log.Println("pdfinfo command not found")
		return 0, err
	}
	out, err := exec.Command("pdfinfo", file).Output()
	if err != nil {
		log.Fatal(err)
		return 0, err
	}
	matches := pdfPageCount.FindStringSubmatch(string(out))
	if len(matches) == 2 {
		return strconv.Atoi(matches[1])
	}
	return 0, nil
}

func createPdf(source, destination string) error {
	_, err := exec.LookPath("soffice")
	if err != nil {
		log.Println("soffice command not found")
		return err
	}

	// TODO: Make this path configurable.
	cmd := exec.Command("soffice", "--headless", "--nologo", "--nofirststartwizard", "--convert-to", "pdf", source, "--outdir", destination)
	log.Println(cmd)

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	err = cmd.Run()
	log.Println(buf.String())
	if err != nil {
		log.Println("error running command", err)
		return err
	}

	return nil
}

func imageFromPdf(source, destination string, size, page, density int) error {
	_, err := exec.LookPath("convert")
	if err != nil {
		log.Println("convert command not found")
		return err
	}

	cmd := exec.Command("convert", "-density", strconv.Itoa(density), "-colorspace", "RGB", fmt.Sprintf("%s[%d]", source, page), "-resize", strconv.Itoa(size), "-flatten", "+adjoin", destination)
	log.Println(cmd)

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	err = cmd.Run()
	if err != nil {
		return err
	}
	log.Println(buf.String())

	return nil
}

func resize(source, destination string, size int) error {
	_, err := exec.LookPath("convert")
	if err != nil {
		log.Println("convert command not found")
		return err
	}

	cmd := exec.Command("convert", source, "-resize", strconv.Itoa(size), destination)
	log.Println(cmd)

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	err = cmd.Run()
	if err != nil {
		return err
	}
	log.Println(buf.String())

	return nil
}

func getRenderedFiles(path string) ([]string, error) {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		log.Println("Error reading files in placeholder base directory:", err)
		return nil, err
	}
	paths := make([]string, 0, 0)
	for _, file := range files {
		if !file.IsDir() {
			// NKG: The convert command will create files of the same name but with the ".pdf" extension.
			if strings.HasSuffix(file.Name(), ".pdf") {
				paths = append(paths, filepath.Join(path, file.Name()))
			}
		}
	}
	return paths, nil
}

func firstGifFrame(source, destination string, size int) error {
	_, err := exec.LookPath("convert")
	if err != nil {
		log.Println("convert command not found")
		return err
	}

	cmd := exec.Command("convert", fmt.Sprintf("%s[0]", source), "-resize", strconv.Itoa(size), destination)
	log.Println(cmd)

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	err = cmd.Run()
	if err != nil {
		return err
	}
	log.Println(buf.String())

	return nil
}

func getBounds(path string) (*image.Rectangle, error) {
	reader, err := os.Open(path)
	if err != nil {
		log.Println("os.Open error", err)
		return nil, err
	}
	defer reader.Close()
	image, err := jpeg.Decode(reader)
	if err != nil {
		log.Println("jpeg.Decode error", err)
		return nil, err
	}
	bounds := image.Bounds()
	return &bounds, nil
}

func getSize(template *common.Template) (int, error) {
	rawSize, err := common.GetFirstAttribute(template, common.TemplateAttributeHeight)
	if err == nil {
		sizeValue, err := strconv.Atoi(rawSize)
		if err == nil {
			return sizeValue, nil
		}
		return 0, err
	}
	return 0, err
}

func getDensity(template *common.Template) (int, error) {
	rawDensity, err := common.GetFirstAttribute(template, common.TemplateAttributeDensity)
	if err == nil {
		density, err := strconv.Atoi(rawDensity)
		if err == nil {
			return density, nil
		}
		return 0, err
	}
	return 0, err
}

func getGeneratedAssetPage(generatedAsset *common.GeneratedAsset) (int, error) {
	rawPage, err := common.GetFirstAttribute(generatedAsset, common.GeneratedAssetAttributePage)
	if err == nil {
		pageValue, err := strconv.Atoi(rawPage)
		if err == nil {
			return pageValue, nil
		}
		return 0, err
	}
	return 0, err
}

func getSourceAssetFileType(sourceAsset *common.SourceAsset) (string, error) {
	fileType, err := common.GetFirstAttribute(sourceAsset, common.SourceAssetAttributeType)
	if err == nil {
		return fileType, nil
	}
	return "unknown", err
}
