#  AZ Spec Validator

The goal of this tool it is to craw the azure spec repo in order to find the errors and inconsistencies of it and report them.

##  How to use

Clone the AzureSpecsRepo and run the tool with the following command:

```bash
go run main.go -s <path to azure spec repo>
```

This will generate a file called `validation-errors.json` with the errors and inconsistencies found.

###  Build

if you want to be able to run the tool from anywhere you can do with the following command:

```bash
go build
```

###  Prerequisites

- [Go](https://golang.org/doc/install)
- [Git](https://git-scm.com/downloads)
- [AzureSpecsRepo](https://github.com/Azure/azure-rest-api-specs)


## List of Errors and inconsistencies that are currently evaluated by the tool

###  Errors

- [ErrorSchemaValidationFailed] - [OAS v2](https://swagger.io/specification/v2/) schema validation failed
- [ErrorTypeIncorrectSchemaVersion] - Incorrect schema version The version of the schema inside the json document is not the same as the version of the folder where the json document is located.
- [ErrorPreviewSchemaWithoutPreviewVersion] - The schema is inside a preview folder but the version is not a preview version.
- [ErrorStableSchemaWithPreviewVersion] - The schema is inside a stable folder but the version is a preview version.

###  Inconsistencies

- [InconsistencyListOperationUsingPost] - The operation is a list operation but it is using a POST method.