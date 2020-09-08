package generator

import (
	"io/ioutil"
	"os"
	"testing"

	"gotest.tools/assert"
)

func TestGenerateTemplate(t *testing.T) {

	_ = os.MkdirAll("testdata/generated", 0777)

	bundlePath := "testdata/bundle.json"
	generatedOutputPath := "testdata/generated/azuredeploy-generated.json"
	expectedOutputPath := "testdata/azuredeploy.json"

	file, err := os.OpenFile(generatedOutputPath, os.O_RDWR|os.O_CREATE, 0644)

	if err != nil {
		t.Errorf("Error opening output file: %w", err)
	}

	defer file.Close()

	options := GenerateTemplateOptions{
		BundleLoc: bundlePath,
		Indent:    true,
		Writer:    file,
	}

	err = GenerateTemplate(options)
	if err != nil {
		t.Errorf("GenerateTemplate failed: %s", err.Error())
	}

	expectedBytes, err := ioutil.ReadFile(expectedOutputPath)
	if err != nil {
		t.Fatalf("failed reading expected output: %s", err)
	}
	expected := string(expectedBytes)

	generatedBytes, err := ioutil.ReadFile(generatedOutputPath)
	if err != nil {
		t.Fatalf("failed reading generated output: %s", err)
	}
	generated := string(generatedBytes)

	assert.Equal(t, expected, generated)
}

func TestGenerateSimpleTemplate(t *testing.T) {

	_ = os.MkdirAll("testdata/generated", 0777)

	bundlePath := "testdata/bundle.json"
	generatedOutputPath := "testdata/generated/azuredeploy-simple-generated.json"
	expectedOutputPath := "testdata/azuredeploy-simple.json"

	file, err := os.OpenFile(generatedOutputPath, os.O_RDWR|os.O_CREATE, 0644)

	if err != nil {
		t.Errorf("Error opening output file: %w", err)
	}

	defer file.Close()

	options := GenerateTemplateOptions{
		BundleLoc: bundlePath,
		Indent:    true,
		Writer:    file,
		Simplify:  true,
	}

	err = GenerateTemplate(options)
	if err != nil {
		t.Errorf("GenerateTemplate failed: %s", err.Error())
	}

	expectedBytes, err := ioutil.ReadFile(expectedOutputPath)
	if err != nil {
		t.Fatalf("failed reading expected output: %s", err)
	}
	expected := string(expectedBytes)

	generatedBytes, err := ioutil.ReadFile(generatedOutputPath)
	if err != nil {
		t.Fatalf("failed reading generated output: %s", err)
	}
	generated := string(generatedBytes)

	assert.Equal(t, expected, generated)
}
