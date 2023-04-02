package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"strings"

	"github.com/go-openapi/loads"
	"github.com/go-openapi/spec"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/validate"
	"github.com/gosuri/uilive"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var allCmd = &cobra.Command{
	Use:   "all",
	Short: "Validate all API specifications",
	Long:  `Validate all API specifications`,
	RunE:  runAllValidations,
}

var (
	pTrue = true
)

type Result struct {
	file string
	errs []SpecErrorOrInconsistency
}

func runAllValidations(cmd *cobra.Command, args []string) error {
	validationErrors := map[string][]SpecErrorOrInconsistency{}
	fmt.Println("Validating all API specifications...")
	sourceDir := viper.GetString("source")
	outputFile := viper.GetString("output")
	categoriesMap := map[string]*bool{
		ErrorSchemaValidationFailed:             nil,
		ErrorTypeIncorrectSchemaVersion:         &pTrue,
		ErrorPreviewSchemaWithoutPreviewVersion: &pTrue,
		ErrorStableSchemaWithPreviewVersion:     &pTrue,
		InconsistencyListOperationUsingPost:     &pTrue,
	}
	if viper.IsSet("categories") {
		categories := viper.GetStringSlice("categories")
		for category := range categoriesMap {
			categoriesMap[category] = nil
		}
		for _, category := range categories {
			categoriesMap[category] = &pTrue
		}
	}

	fmt.Println("Source directory:", sourceDir)
	// get each openapi spec file inside the specification directory
	specFiles, err := getSpecFiles(sourceDir)
	if err != nil {
		return err
	}
	fmt.Println("Total number of files to validate:", len(specFiles))
	fmt.Println("Validating files...")
	inconsistenciesChannel := make(chan Result, len(specFiles))
	jobs := make(chan string, len(specFiles))
	for _, specFile := range specFiles {
		jobs <- specFile
	}
	for i := 0; i < 10; i++ {
		go func(jobs <-chan string, inconsistenciesChannel chan Result) {
			for specFile := range jobs {
				doc, err := loads.JSONSpec(specFile)
				if err != nil {
					fmt.Println(err)
				}
				inconsistencies := validateErrorTypes(doc, specFile, categoriesMap)

				if categoriesMap[ErrorSchemaValidationFailed] != nil {
					validation := validate.NewSchemaValidator(spec.MustLoadSwagger20Schema(), nil, "", strfmt.Default).Validate(doc.Spec())
					if validation != nil && len(validation.Errors) > 0 {
						for _, err := range validation.Errors {
							inconsistencies = append(inconsistencies, SpecErrorOrInconsistency{
								Inconsistency: ErrorSchemaValidationFailed,
								Err:           err.Error(),
							})
						}
					}
				}
				inconsistenciesChannel <- Result{
					file: specFile,
					errs: inconsistencies,
				}
			}
		}(jobs, inconsistenciesChannel)
	}
	writer := uilive.New()
	writer.Start()

	for i := 0; i < len(specFiles); i++ {
		result := <-inconsistenciesChannel
		if len(result.errs) > 0 {
			validationErrors[result.file] = result.errs
		}
		fmt.Fprintf(writer, "%d/%d files validated, with %d files with errors\n", i+1, len(specFiles), len(validationErrors))
	}
	fmt.Fprintf(writer, "\n%d files validated, %d files with errors found\n", len(specFiles), len(validationErrors))
	fmt.Println()
	jsonOutput, err := json.MarshalIndent(validationErrors, "", "  ")
	if err != nil {
		return err
	}
	ioutil.WriteFile(outputFile, jsonOutput, 0644)

	// return error if any validation fails
	return nil
}

const (
	ErrorSchemaValidationFailed             = "SchemaValidationFailed"
	ErrorTypeIncorrectSchemaVersion         = "IncorrectSchemaVersion"
	ErrorPreviewSchemaWithoutPreviewVersion = "PreviewSchemaWithoutPreviewVersion"
	ErrorStableSchemaWithPreviewVersion     = "StableSchemaWithPreviewVersion"
	InconsistencyListOperationUsingPost     = "ListOperationUsingPost"
)

type SpecErrorOrInconsistency struct {
	Inconsistency string
	Err           string
}

func validateErrorTypes(doc *loads.Document, specFile string, categories map[string]*bool) []SpecErrorOrInconsistency {
	inconsistencies := []SpecErrorOrInconsistency{}
	isPreviewDirectory := strings.Split(specFile, "/")[5] == "preview"
	versionFromPath := strings.Split(specFile, "/")[6]
	versionFromSpec := doc.Spec().Info.Version
	if categories[ErrorTypeIncorrectSchemaVersion] != nil {
		if versionFromPath != versionFromSpec {
			inconsistencies = append(inconsistencies, SpecErrorOrInconsistency{
				Inconsistency: ErrorTypeIncorrectSchemaVersion,
				Err:           fmt.Sprintf("incorrect schema version. Path: %s, Spec: %s", versionFromPath, versionFromSpec),
			})
		}
	}
	if categories[ErrorPreviewSchemaWithoutPreviewVersion] != nil {
		if isPreviewDirectory && !strings.Contains(versionFromSpec, "preview") {
			inconsistencies = append(inconsistencies, SpecErrorOrInconsistency{
				Inconsistency: ErrorPreviewSchemaWithoutPreviewVersion,
				Err:           fmt.Sprintf("preview schema without preview version. Path: %s, Spec: %s", versionFromPath, versionFromSpec),
			})
		}
	}
	if categories[ErrorStableSchemaWithPreviewVersion] != nil {
		if !isPreviewDirectory && strings.Contains(versionFromSpec, "preview") {
			inconsistencies = append(inconsistencies, SpecErrorOrInconsistency{
				Inconsistency: ErrorStableSchemaWithPreviewVersion,
				Err:           fmt.Sprintf("stable schema with preview version. Path: %s, Spec: %s", versionFromPath, versionFromSpec),
			})
		}
	}
	if categories[InconsistencyListOperationUsingPost] != nil {
		for url, path := range doc.Spec().Paths.Paths {
			// get the trailing of the url
			ending := strings.Split(url, "/")[len(strings.Split(url, "/"))-1]
			if path.Post != nil && strings.Index(ending, "list") == 0 {
				inconsistencies = append(inconsistencies, SpecErrorOrInconsistency{
					Inconsistency: InconsistencyListOperationUsingPost,
					Err:           fmt.Sprintf("list operation using post. Path: %s", url),
				})
			}
		}
	}

	return inconsistencies
}

func getSpecFiles(sourceDir string) ([]string, error) {
	specFilePaths := []string{}
	specPath := path.Join(sourceDir, "specification")
	specFiles, err := ioutil.ReadDir(specPath)
	if err != nil {
		return nil, err
	}
	for _, specDir := range specFiles {
		if specDir.IsDir() {
			// get all the files in the directory
			resourceManagerDir, err := ioutil.ReadDir(path.Join(specPath, specDir.Name()))
			if err != nil {
				return nil, err
			}
			for _, resourceManager := range resourceManagerDir {
				if resourceManager.IsDir() && resourceManager.Name() == "resource-manager" {
					// get all directories in the resource-manager directory
					namespacesDir, err := ioutil.ReadDir(path.Join(specPath, specDir.Name(), resourceManager.Name()))
					if err != nil {
						return nil, err
					}
					for _, namespace := range namespacesDir {
						// get all the files in the namespace directory
						if namespace.IsDir() {
							namespacePath := path.Join(specPath, specDir.Name(), resourceManager.Name(), namespace.Name())
							stablePreviewDirs, err := ioutil.ReadDir(namespacePath)
							if err != nil {
								return nil, err
							}
							for _, stablePreview := range stablePreviewDirs {
								if stablePreview.IsDir() && stablePreview.Name() == "stable" || stablePreview.Name() == "preview" {
									stablePreviewPath := path.Join(namespacePath, stablePreview.Name())
									stableVersions, err := ioutil.ReadDir(stablePreviewPath)
									if err != nil {
										return nil, err
									}
									for _, stableVersion := range stableVersions {
										if stableVersion.IsDir() {
											stableVersionPath := path.Join(stablePreviewPath, stableVersion.Name())
											specFiles, err := ioutil.ReadDir(stableVersionPath)
											if err != nil {
												return nil, err
											}
											for _, specFile := range specFiles {
												if !specFile.IsDir() && path.Ext(specFile.Name()) == ".json" {
													specFilePaths = append(specFilePaths, path.Join(stableVersionPath, specFile.Name()))
												}
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}

	return specFilePaths, nil
}

func init() {
	rootCmd.AddCommand(allCmd)
	allCmd.Flags().StringP("output", "o", "validation-errors.json", "output file")
	allCmd.Flags().StringArray("categories", []string{
		ErrorSchemaValidationFailed,
		ErrorTypeIncorrectSchemaVersion,
		ErrorPreviewSchemaWithoutPreviewVersion,
		ErrorStableSchemaWithPreviewVersion,
		InconsistencyListOperationUsingPost,
	}, "categories of errors to validate")
	viper.BindPFlag("categories", allCmd.Flags().Lookup("categories"))
	viper.BindPFlag("output", allCmd.Flags().Lookup("output"))
}
