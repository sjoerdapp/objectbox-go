/*
 * Copyright 2018 ObjectBox Ltd. All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package generator

import (
	"bytes"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/objectbox/objectbox-go/internal/generator"
	"github.com/objectbox/objectbox-go/test/assert"
)

// generateAllDirs walks through the "data" and generates bindings for each subdirectory
// set overwriteExpected to TRUE to update all ".expected" files with the generated content
func generateAllDirs(t *testing.T, overwriteExpected bool) {
	var datadir = "testdata"
	folders, err := ioutil.ReadDir(datadir)
	assert.NoErr(t, err)

	for _, folder := range folders {
		if !folder.IsDir() {
			continue
		}

		var dir = filepath.Join(datadir, folder.Name())

		modelInfoFile := generator.ModelInfoFile(dir)
		modelInfoExpectedFile := modelInfoFile + ".expected"
		modelInfoInitialFile := modelInfoFile + ".initial"

		modelFile := generator.ModelFile(modelInfoFile)
		modelExpectedFile := modelFile + ".expected"

		// run the generation twice, first time with deleting old modelInfo
		for i := 0; i <= 1; i++ {
			if i == 0 {
				t.Logf("Testing %s without model info JSON", folder.Name())
				os.Remove(modelInfoFile)
			} else {
				t.Logf("Testing %s with previous model info JSON", folder.Name())
			}

			if fileExists(modelInfoInitialFile) {
				assert.NoErr(t, copyFile(modelInfoInitialFile, modelInfoFile))
			}

			generateAllFiles(t, overwriteExpected, dir, modelInfoFile)

			assertSameFile(t, modelInfoFile, modelInfoExpectedFile, overwriteExpected)
			assertSameFile(t, modelFile, modelExpectedFile, overwriteExpected)
		}
	}
}

func assertSameFile(t *testing.T, file string, expectedFile string, overwriteExpected bool) {
	if !fileExists(expectedFile) {
		assert.Eq(t, false, fileExists(file))
		return
	}

	content, err := ioutil.ReadFile(file)
	assert.NoErr(t, err)

	if overwriteExpected {
		assert.NoErr(t, copyFile(file, expectedFile))
	}

	contentExpected, err := ioutil.ReadFile(expectedFile)
	assert.NoErr(t, err)

	if 0 != bytes.Compare(content, contentExpected) {
		assert.Failf(t, "generated file %s is not the same as %s", file, expectedFile)
	}
}

func generateAllFiles(t *testing.T, overwriteExpected bool, dir string, modelInfoFile string) {
	// NOTE test-only - avoid changes caused by random numbers by fixing them to the same seed all the time
	rand.Seed(0)

	var modelFile = generator.ModelFile(modelInfoFile)

	// process all *.go files in the directory
	inputFiles, err := filepath.Glob(filepath.Join(dir, "*.go"))
	assert.NoErr(t, err)
	for _, sourceFile := range inputFiles {
		// skip generated files & "expected results" files
		if strings.HasSuffix(sourceFile, ".obx.go") ||
			strings.HasSuffix(sourceFile, "expected") ||
			strings.HasSuffix(sourceFile, "initial") ||
			sourceFile == modelFile {
			continue
		}

		t.Logf("  %s", filepath.Base(sourceFile))

		err = generator.Process(sourceFile, modelInfoFile)

		// handle negative test
		var shouldFail = strings.HasPrefix(filepath.Base(sourceFile), "_")
		if shouldFail {
			if err == nil {
				assert.Failf(t, "Unexpected PASS on a negative test %s", sourceFile)
			} else {
				continue
			}
		}

		assert.NoErr(t, err)

		var bindingFile = generator.BindingFile(sourceFile)
		var expectedFile = bindingFile + ".expected"
		assertSameFile(t, bindingFile, expectedFile, overwriteExpected)
	}
}
