package utils

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/otiai10/gosseract/v2"
)

func SaveBytesToFile(b []byte, path string) error {
	return os.WriteFile(path, b, 0644)
}

func ConvertPDFToPNGs(pdfPath string, outDir string) ([]string, error) {
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return nil, err
	}
	outPrefix := filepath.Join(outDir, "page")
	cmd := exec.Command("pdftoppm", "-png", pdfPath, outPrefix)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("pdftoppm error: %v: %s", err, stderr.String())
	}
	files, err := filepath.Glob(filepath.Join(outDir, "page*.png"))
	if err != nil {
		return nil, err
	}
	return files, nil
}

// OCRImagePath fait l'OCR sur une image PNG/JPG
func OCRImagePath(imgPath string) (string, error) {
	client := gosseract.NewClient()
	defer client.Close()
	if err := client.SetImage(imgPath); err != nil {
		return "", err
	}
	return client.Text()
}

// OCRFromPDFBytes convertit PDF (ou image) en texte
func OCRFromPDFBytes(pdfBytes []byte) (string, error) {
	tmpDir, err := os.MkdirTemp("", "ocrpdf-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmpDir)

	pdfPath := filepath.Join(tmpDir, "upload.pdf")
	if err := SaveBytesToFile(pdfBytes, pdfPath); err != nil {
		return "", err
	}

	imgDir := filepath.Join(tmpDir, "images")
	imgs, err := ConvertPDFToPNGs(pdfPath, imgDir)
	if err != nil {
		// fallback : essayer OCR direct sur PDF (si image)
		imgPath := filepath.Join(tmpDir, "fallback.png")
		if werr := SaveBytesToFile(pdfBytes, imgPath); werr == nil {
			return OCRImagePath(imgPath)
		}
		return "", err
	}

	var fullText strings.Builder
	for _, img := range imgs {
		t, err := OCRImagePath(img)
		if err != nil {
			continue
		}
		fullText.WriteString(t)
		fullText.WriteString("\n")
	}
	return fullText.String(), nil
}
