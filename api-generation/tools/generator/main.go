package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/getkin/kin-openapi/openapi3"
)

func main() {
	if len(os.Args) < 3 {
		log.Fatal("Usage: generator <openapi-file> <base-output-dir>")
	}

	schemaFile := os.Args[1]
	baseOutputDir := os.Args[2]

	// Load OpenAPI spec
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromFile(schemaFile)
	if err != nil {
		log.Fatalf("Failed to load OpenAPI spec: %v", err)
	}

	// Parse OpenAPI to intermediate model
	parser := NewOpenAPIParser(doc)
	model, err := parser.Parse()
	if err != nil {
		log.Fatalf("Failed to parse OpenAPI spec: %v", err)
	}

	// Generate actor-specific packages using the intermediate model
	generator := &Generator{}
	err = generator.GenerateActorPackages(model, baseOutputDir)
	if err != nil {
		log.Fatalf("Failed to generate actor packages: %v", err)
	}
}

// Generator handles code generation from the intermediate model
type Generator struct{}

// GenerateActorPackages generates actor-specific packages from the intermediate model
func (g *Generator) GenerateActorPackages(model *GenerationModel, baseOutputDir string) error {
	if len(model.Actors) == 0 {
		return fmt.Errorf("no actors found in the model")
	}

	// First, generate shared types package if there are shared types
	if len(model.SharedTypes) > 0 || len(model.SharedTypeAliases) > 0 {
		err := g.generateSharedTypes(model, baseOutputDir)
		if err != nil {
			return fmt.Errorf("failed to generate shared types: %v", err)
		}
	}

	// Generate package for each actor type
	for _, actor := range model.Actors {
		// Create actor-specific package name and directory
		packageName := strings.ToLower(actor.ActorType)
		if !strings.HasSuffix(packageName, "actor") {
			packageName += "actor"
		}

		outputDir := filepath.Join(baseOutputDir, packageName)

		// Create output directory
		err := os.MkdirAll(outputDir, 0755)
		if err != nil {
			return fmt.Errorf("failed to create output directory %s: %v", outputDir, err)
		}

		// Get actor-specific types (if any)
		actorSpecificTypes := model.ActorSpecificTypes[actor.ActorType]
		if actorSpecificTypes == nil {
			actorSpecificTypes = []TypeDef{}
		}

		// Create actor model for this specific actor
		actorModel := ActorModel{
			ActorType:      actor.ActorType,
			PackageName:    packageName,
			Types:          actorSpecificTypes, // Only actor-specific types
			TypeAliases:    []TypeAlias{},      // Actor-specific aliases (none for now)
			ActorInterface: actor,
		}

		// Generate types for this actor
		err = g.generateActorTypes(&actorModel, outputDir)
		if err != nil {
			return fmt.Errorf("failed to generate types for %s: %v", actor.ActorType, err)
		}

		// Generate interface for this actor
		err = g.generateActorInterface(&actorModel, outputDir)
		if err != nil {
			return fmt.Errorf("failed to generate interface for %s: %v", actor.ActorType, err)
		}

		// NOTE: Factory generation skipped for now as it should reference actual implementation types
		// The factories should be in the implementation packages, not the generated packages

		fmt.Printf("Generated actor package: %s\n", outputDir)
		fmt.Printf("  %s/types.go\n", outputDir)
		fmt.Printf("  %s/api.go\n", outputDir)
		// fmt.Printf("  %s/factory.go\n", outputDir) // Skipped for now
	}

	return nil
}

// generateSharedTypes generates the shared types package
func (g *Generator) generateSharedTypes(model *GenerationModel, baseOutputDir string) error {
	// Create shared types directory
	sharedTypesDir := filepath.Join(baseOutputDir, "types")
	err := os.MkdirAll(sharedTypesDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create shared types directory %s: %v", sharedTypesDir, err)
	}

	// Load template from file
	templatePath := getTemplatePath("shared_types.tmpl")
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return fmt.Errorf("failed to parse shared types template: %v", err)
	}

	// Generate shared types file
	data := SharedTypesTemplateData{
		PackageName:   "types",
		SharedTypes:   model.SharedTypes,
		SharedAliases: model.SharedTypeAliases,
	}

	typesFile, err := os.Create(fmt.Sprintf("%s/types.go", sharedTypesDir))
	if err != nil {
		return fmt.Errorf("failed to create shared types file: %v", err)
	}
	defer typesFile.Close()

	err = tmpl.Execute(typesFile, data)
	if err != nil {
		return fmt.Errorf("failed to execute shared types template: %v", err)
	}

	fmt.Printf("Generated shared types package: %s/types.go\n", sharedTypesDir)
	return nil
}

func (g *Generator) generateActorTypes(actorModel *ActorModel, outputDir string) error {
	// Load template from file
	templatePath := getTemplatePath("actor_types.tmpl")
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return fmt.Errorf("failed to parse actor types template: %v", err)
	}

	// Process types to replace shared type references with qualified names
	processedTypes := make([]TypeDef, len(actorModel.Types))
	copy(processedTypes, actorModel.Types)
	
	// Check if any field types reference shared types (for import decision)
	hasSharedTypeReferences := false
	// Note: In a more complete implementation, we would:
	// 1. Check each field type to see if it references a shared type
	// 2. Replace those references with "types.TypeName"
	// 3. Set hasSharedTypeReferences = true if any references found

	// Generate types file
	data := struct {
		PackageName string
		Types       []TypeDef
		TypeAliases []TypeAlias
		SharedTypes bool // Indicates if shared types package needs to be imported
	}{
		PackageName: actorModel.PackageName,
		Types:       processedTypes,
		TypeAliases: actorModel.TypeAliases,
		SharedTypes: hasSharedTypeReferences, // Only import if actually needed
	}

	typesFile, err := os.Create(fmt.Sprintf("%s/types.go", outputDir))
	if err != nil {
		return fmt.Errorf("failed to create types file: %v", err)
	}
	defer typesFile.Close()

	err = tmpl.Execute(typesFile, data)
	if err != nil {
		return fmt.Errorf("failed to execute actor types template: %v", err)
	}

	return nil
}

func (g *Generator) generateActorInterface(actorModel *ActorModel, outputDir string) error {
	// Load template from file
	templatePath := getTemplatePath("interface.tmpl")
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return fmt.Errorf("failed to parse interface template: %v", err)
	}

	// Generate interface file for this actor
	data := SingleActorTemplateData{
		PackageName: actorModel.PackageName,
		Actor:       actorModel.ActorInterface,
	}

	// Use api.go as filename instead of generated.go for better clarity
	interfaceFile, err := os.Create(filepath.Join(outputDir, "api.go"))
	if err != nil {
		return fmt.Errorf("failed to create interface file: %v", err)
	}
	defer interfaceFile.Close()

	err = tmpl.Execute(interfaceFile, data)
	if err != nil {
		return fmt.Errorf("failed to execute interface template: %v", err)
	}

	return nil
}

func (g *Generator) generateActorFactory(actorModel *ActorModel, outputDir string) error {
	// Load template from file
	templatePath := getTemplatePath("factory.tmpl")
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return fmt.Errorf("failed to parse factory template: %v", err)
	}

	// Generate factory file for this actor
	data := SingleActorTemplateData{
		PackageName: actorModel.PackageName,
		Actor:       actorModel.ActorInterface,
	}

	factoryFile, err := os.Create(filepath.Join(outputDir, "factory.go"))
	if err != nil {
		return fmt.Errorf("failed to create factory file: %v", err)
	}
	defer factoryFile.Close()

	err = tmpl.Execute(factoryFile, data)
	if err != nil {
		return fmt.Errorf("failed to execute factory template: %v", err)
	}

	return nil
}

// Utility functions

func getTemplatePath(templateName string) string {
	// Get the directory where this binary is located
	execPath, err := os.Executable()
	if err != nil {
		log.Fatalf("Failed to get executable path: %v", err)
	}
	
	// Look for templates directory relative to the executable
	execDir := filepath.Dir(execPath)
	templatePath := filepath.Join(execDir, "..", "templates", templateName)
	
	// If not found, try relative to current working directory (for development)
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		wd, _ := os.Getwd()
		templatePath = filepath.Join(wd, "templates", templateName)
		
		// If still not found, try relative to the generator source directory
		if _, err := os.Stat(templatePath); os.IsNotExist(err) {
			// Try to find the templates directory in the project structure
			// Walk up from the executable to find api-generation/tools/generator/templates
			currentDir := execDir
			for i := 0; i < 10; i++ { // Limit search depth
				testPath := filepath.Join(currentDir, "generator", "templates", templateName)
				if _, err := os.Stat(testPath); err == nil {
					templatePath = testPath
					break
				}
				testPath = filepath.Join(currentDir, "tools", "generator", "templates", templateName)
				if _, err := os.Stat(testPath); err == nil {
					templatePath = testPath
					break
				}
				testPath = filepath.Join(currentDir, "api-generation", "tools", "generator", "templates", templateName)
				if _, err := os.Stat(testPath); err == nil {
					templatePath = testPath
					break
				}
				currentDir = filepath.Dir(currentDir)
				if currentDir == "/" || currentDir == filepath.Dir(currentDir) {
					break
				}
			}
		}
	}
	
	return templatePath
}