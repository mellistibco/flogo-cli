package dep

import (
	"strings"
	"go/token"
	"go/ast"
	"bytes"
	"go/printer"
	"os/exec"
	"os"
	"fmt"
	"path"
	"go/parser"
	"strconv"
	"io/ioutil"
	"errors"
	"encoding/json"

	"github.com/TIBCOSoftware/flogo-cli/config"
	"github.com/TIBCOSoftware/flogo-cli/util"
)

type DepManager struct {
	AppDir string
}


type ConstraintDef struct {
	ProjectRoot string
	Version     string
}

// Init initializes the dependency manager
func (b *DepManager) Init(rootDir string) error {
	exists := fgutil.ExecutableExists("dep")
	if !exists {
		return errors.New("dep not installed")
	}

	cmd := exec.Command("dep", "init")
	cmd.Dir = b.AppDir
	newEnv := os.Environ()
	newEnv = append(newEnv, fmt.Sprintf("GOPATH=%s", rootDir))
	cmd.Env = newEnv

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return err
	}

	// TODO remove this prune cmd once it gets absorved into dep ensure https://github.com/golang/dep/issues/944
	cmd = exec.Command("dep", "prune")
	cmd.Dir = b.AppDir
	cmd.Env = newEnv

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}


// IsInitialized Returns true if a dep environment has been initialized
func (b *DepManager) IsInitialized() bool {

	_, err :=os.Stat(path.Join(b.AppDir, "Gopkg.toml"))
	if err != nil{
		return false
	}
	_, err =os.Stat(path.Join(b.AppDir, "Gopkg.lock"))
	if err != nil{
		return false
	}

	return true
}

// InstallDependency installs the given dependency
func (b *DepManager) InstallDependency(rootDir, appDir, depPath , depVersion string) error {
	exists := fgutil.ExecutableExists("dep")
	if !exists {
		return errors.New("dep not installed")
	}
	fmt.Println("Validating existing dependencies, this might take a few seconds...")

	// Load imports file
	importsPath := path.Join(appDir, config.FileImportsGo)
	// Validate that it exists
	_, err := os.Stat(importsPath)

	if err != nil {
		return fmt.Errorf("Error installing dependency, import file '%s' doesn't exists", importsPath)
	}

	fset := token.NewFileSet()

	importsFileAst, err := parser.ParseFile(fset, importsPath, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("Error parsing import file '%s', %s", importsPath, err)
	}

	//Validate that the install does not exist in imports.go file
	for _, imp := range importsFileAst.Imports {
		if imp.Path.Value == strconv.Quote(depPath) {
			return fmt.Errorf("Error installing dependency, import '%s' already exists", depPath)
		}
	}

	existingConstraint, err := GetExistingConstraint(rootDir, appDir, depPath)
	if err != nil {
		return err
	}

	if existingConstraint != nil {
		if len(depVersion) > 0 {
			fmt.Printf("Existing root package version found '%s', to update it please change Gopkg.toml manually\n", existingConstraint.Version)
		}
	} else {
		// Contraint does not exist add it
		fmt.Printf("Adding new dependency '%s' version '%s' \n", depPath, depVersion)
		cmd := exec.Command("dep", "ensure", "-add", fmt.Sprintf("%s@%s", depPath, depVersion))
		cmd.Dir = appDir
		newEnv := os.Environ()
		newEnv = append(newEnv, fmt.Sprintf("GOPATH=%s", rootDir))
		cmd.Env = newEnv

		// Only show errors
		//cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		err := cmd.Run()
		if err != nil {
			return fmt.Errorf("Error adding dependency '%s', '%s'", depPath, err.Error())
		}
	}

	// Add the import
	for i := 0; i < len(importsFileAst.Decls); i++ {
		d := importsFileAst.Decls[i]

		switch d.(type) {
		case *ast.FuncDecl:
		// No action
		case *ast.GenDecl:
			dd := d.(*ast.GenDecl)

			// IMPORT Declarations
			if dd.Tok == token.IMPORT {
				// Add the new import
				newSpec := &ast.ImportSpec{Name: &ast.Ident{Name: "_"}, Path: &ast.BasicLit{Value: strconv.Quote(depPath)}}
				dd.Specs = append(dd.Specs, newSpec)
				break
			}
		}
	}

	ast.SortImports(fset, importsFileAst)

	out, err := GenerateFile(fset, importsFileAst)
	if err != nil {
		return fmt.Errorf("Error creating import file '%s', %s", importsPath, err)
	}

	err = ioutil.WriteFile(importsPath, out, os.ModePerm)
	if err != nil {
		return fmt.Errorf("Error creating import file '%s', %s", importsPath, err)
	}

	// Sync up
	fmt.Printf("Synching up Gopkg.toml and imports \n")
	cmd := exec.Command("dep", "ensure")
	cmd.Dir = appDir
	newEnv := os.Environ()
	newEnv = append(newEnv, fmt.Sprintf("GOPATH=%s", rootDir))
	cmd.Env = newEnv

	// Only show errors
	//cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("Error Synching up Gopkg.toml and imports '%s', '%s'", depPath, err.Error())
	}

	fmt.Printf("'%s' installed successfully \n", depPath)

	return nil
}


// UninstallDependency deletes the given dependency
func (b *DepManager) UninstallDependency(rootDir, appDir , depPath string) error {
	exists := fgutil.ExecutableExists("dep")
	if !exists {
		return errors.New("dep not installed")
	}

	// Load imports file
	importsPath := path.Join(appDir, config.FileImportsGo)
	// Validate that it exists
	_, err := os.Stat(importsPath)

	if err != nil {
		return fmt.Errorf("Error installing dependency, import file '%s' doesn't exists", importsPath)
	}

	fset := token.NewFileSet()

	importsFileAst, err := parser.ParseFile(fset, importsPath, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("Error parsing import file '%s', %s", importsPath, err)
	}

	exists = false

	//Validate that the install exists in imports.go file
	for _, imp := range importsFileAst.Imports {
		if imp.Path.Value == strconv.Quote(depPath) {
			exists = true
			break
		}
	}

	if !exists{
		fmt.Printf("No import '%s' found in import file \n", depPath)
		// Just sync up and return
		// Sync up
		fmt.Printf("Synching up Gopkg.toml and imports \n")
		cmd := exec.Command("dep", "ensure")
		cmd.Dir = appDir
		newEnv := os.Environ()
		newEnv = append(newEnv, fmt.Sprintf("GOPATH=%s", rootDir))
		cmd.Env = newEnv

		// Only show errors
		//cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		err = cmd.Run()
		if err != nil {
			return fmt.Errorf("Error Synching up Gopkg.toml and imports '%s', '%s'", depPath, err.Error())
		}

		fmt.Printf("'%s' uninstalled successfully \n", depPath)
		return nil
	}

	fmt.Printf("Deleting import from imports file \n")
	// Delete the import
	for i := 0; i < len(importsFileAst.Decls); i++ {
		d := importsFileAst.Decls[i]

		switch d.(type) {
		case *ast.FuncDecl:
		// No action
		case *ast.GenDecl:
			dd := d.(*ast.GenDecl)

			// IMPORT Declarations
			if dd.Tok == token.IMPORT {
				var newSpecs []ast.Spec
				for _, spec := range dd.Specs {
					importSpec, ok := spec.(*ast.ImportSpec)
					if !ok{
						newSpecs = append(newSpecs, spec)
						continue
					}
					// Check Path
					if importPath := importSpec.Path; importPath.Value != strconv.Quote(depPath) {
						// Add import
						newSpecs = append(newSpecs, spec)
						continue
					}
				}
				// Update specs
				dd.Specs = newSpecs
				break
			}
		}
	}

	ast.SortImports(fset, importsFileAst)

	out, err := GenerateFile(fset, importsFileAst)
	if err != nil {
		return fmt.Errorf("Error creating import file '%s', %s", importsPath, err)
	}

	err = ioutil.WriteFile(importsPath, out, os.ModePerm)
	if err != nil {
		return fmt.Errorf("Error creating import file '%s', %s", importsPath, err)
	}

	// Sync up
	fmt.Printf("Synching up Gopkg.toml and imports \n")
	cmd := exec.Command("dep", "ensure")
	cmd.Dir = appDir
	newEnv := os.Environ()
	newEnv = append(newEnv, fmt.Sprintf("GOPATH=%s", rootDir))
	cmd.Env = newEnv

	// Only show errors
	//cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("Error Synching up Gopkg.toml and imports '%s', '%s'", depPath, err.Error())
	}

	fmt.Printf("'%s' uninstalled successfully \n", depPath)
	return nil
}

// GetExistingConstraint returns the constraint definition if it already exists
func GetExistingConstraint(rootDir, appDir, depPath string) (*ConstraintDef, error) {
	// Validate that the install project does not exist in Gopkg.toml
	cmd := exec.Command("dep", "status", "-json")
	cmd.Dir = appDir
	newEnv := os.Environ()
	newEnv = append(newEnv, fmt.Sprintf("GOPATH=%s", rootDir))
	cmd.Env = newEnv

	status, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("Error checking project dependency status '%s'", err)
	}

	var statusMap []map[string]interface{}

	err = json.Unmarshal(status, &statusMap)
	if err != nil {
		return nil, fmt.Errorf("Error marshalling project dependency status '%s'", err)
	}

	var existingConstraint map[string]interface{}

	for _, constraint := range statusMap {
		// Get project root
		projectRoot, ok := constraint["ProjectRoot"]
		if !ok {
			continue
		}
		pr := projectRoot.(string)
		if strings.HasPrefix(depPath, pr) {
			// Constraint already exists
			existingConstraint = constraint
			break
		}
	}

	var constraint *ConstraintDef

	if existingConstraint != nil {
		constraint = &ConstraintDef{ProjectRoot: existingConstraint["ProjectRoot"].(string), Version: existingConstraint["Version"].(string)}
	}

	return constraint, nil
}

func GenerateFile(fset *token.FileSet, file *ast.File) ([]byte, error) {
	var output []byte
	buffer := bytes.NewBuffer(output)
	if err := printer.Fprint(buffer, fset, file); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}
