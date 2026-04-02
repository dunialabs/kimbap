package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestGenerateFromOpenAPIYAMLWithRefs(t *testing.T) {
	spec := `openapi: 3.0.3
info:
  title: Pet Store API
  version: 1.2.3
  description: Manage pets
servers:
  - url: https://api.petstore.example
security:
  - ApiKeyAuth: []
components:
  securitySchemes:
    ApiKeyAuth:
      type: apiKey
      in: header
      name: X-API-Key
  parameters:
    PetID:
      name: petId
      in: path
      required: true
      schema:
        type: string
    Limit:
      name: limit
      in: query
      schema:
        type: integer
        default: 10
  requestBodies:
    CreatePet:
      required: true
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/PetCreate'
  schemas:
    PetCreate:
      type: object
      required: [name]
      properties:
        name:
          type: string
        age:
          type: integer
paths:
  /pets/{petId}:
    parameters:
      - $ref: '#/components/parameters/PetID'
    get:
      operationId: getPet
      summary: Get pet
      parameters:
        - $ref: '#/components/parameters/Limit'
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema:
                type: object
    post:
      summary: Create pet
      requestBody:
        $ref: '#/components/requestBodies/CreatePet'
      responses:
        '201':
          description: created
          content:
            application/json:
              schema:
                type: object
`

	manifest, err := GenerateFromOpenAPI([]byte(spec))
	if err != nil {
		t.Fatalf("GenerateFromOpenAPI failed: %v", err)
	}

	if manifest.Name != "pet-store-api" {
		t.Fatalf("unexpected name: %s", manifest.Name)
	}
	if manifest.Auth.Type != "header" || manifest.Auth.HeaderName != "X-API-Key" {
		t.Fatalf("unexpected auth mapping: %+v", manifest.Auth)
	}

	getAction, ok := manifest.Actions["getpet"]
	if !ok {
		t.Fatalf("expected getpet action to exist")
	}
	if getAction.Request.PathParams["petId"] != "{petId}" {
		t.Fatalf("expected path param mapping in request")
	}
	if getAction.Request.Query["limit"] != "{limit}" {
		t.Fatalf("expected query param mapping in request")
	}

	postAction, ok := manifest.Actions["post-pets-petid"]
	if !ok {
		t.Fatalf("expected generated action key for POST operation")
	}
	if postAction.Risk.Level != "medium" {
		t.Fatalf("unexpected POST risk: %+v", postAction.Risk)
	}
	if postAction.Request.Body["name"] != "{name}" {
		t.Fatalf("expected request body field mapping for 'name'")
	}
}

func TestGenerateFromOpenAPIJSONMissingServerAndSecurity(t *testing.T) {
	spec := `{
  "openapi": "3.1.0",
  "info": {
    "title": "Inventory API",
    "version": "v2"
  },
  "paths": {
    "/items/{id}": {
      "delete": {
        "parameters": [
          {
            "name": "id",
            "in": "path",
            "required": true,
            "schema": {"type": "string"}
          }
        ],
        "responses": {
          "204": {"description": "deleted"}
        }
      }
    }
  }
}`

	_, err := GenerateFromOpenAPI([]byte(spec))
	if err == nil {
		t.Fatalf("expected error when OpenAPI servers are missing")
	}
	if !strings.Contains(err.Error(), "server URL") {
		t.Fatalf("expected missing server URL error, got %v", err)
	}
}

func TestGenerateFromOpenAPIURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`openapi: 3.0.0
info:
  title: URL Skill
  version: 1.0.0
servers:
  - url: https://example.org
paths:
  /health:
    get:
      operationId: health
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema:
                type: object
`))
	}))
	defer server.Close()

	manifest, err := GenerateFromOpenAPIURL(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("GenerateFromOpenAPIURL failed: %v", err)
	}

	if manifest.Name != "url-skill" {
		t.Fatalf("unexpected name from URL source: %s", manifest.Name)
	}
	if _, ok := manifest.Actions["health"]; !ok {
		t.Fatalf("expected health action from URL source")
	}
}

func TestGenerateFromOpenAPIURLResolvesRelativeServerURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`openapi: 3.0.0
info:
  title: Relative Server URL Skill
  version: 1.0.0
servers:
  - url: /api
paths:
  /health:
    get:
      operationId: health
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema:
                type: object
`))
	}))
	defer server.Close()

	manifest, err := GenerateFromOpenAPIURL(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("GenerateFromOpenAPIURL failed: %v", err)
	}

	if manifest.BaseURL != server.URL+"/api" {
		t.Fatalf("expected baseURL to resolve against fetched origin, got %q", manifest.BaseURL)
	}
}

func TestGenerateFromOpenAPIFileSupportsNestedExternalRefs(t *testing.T) {
	dir := t.TempDir()
	rootPath := writeOpenAPITestFile(t, dir, "openapi.yaml", `openapi: 3.0.3
info:
  title: Split Pet API
  version: 1.0.0
servers:
  - url: https://api.example.com
paths:
  /pets:
    post:
      operationId: createPet
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: ./schemas/pet-create.yaml#/PetCreate
      responses:
        '201':
          description: created
          content:
            application/json:
              schema:
                type: object
`)
	writeOpenAPITestFile(t, dir, "schemas/pet-create.yaml", `PetCreate:
  type: object
  required: [name, category]
  properties:
    name:
      type: string
    category:
      $ref: ./common/category.yaml#/Category
`)
	writeOpenAPITestFile(t, dir, "schemas/common/category.yaml", `Category:
  type: string
  enum: [dog, cat]
`)

	manifest, err := GenerateFromOpenAPIFile(rootPath)
	if err != nil {
		t.Fatalf("GenerateFromOpenAPIFile failed: %v", err)
	}

	action, ok := manifest.Actions["createpet"]
	if !ok {
		t.Fatalf("expected createpet action")
	}

	args := generatedActionArgsByName(action)
	if args["name"].Type != "string" || !args["name"].Required {
		t.Fatalf("expected required name:string arg, got %+v", args["name"])
	}
	if args["category"].Type != "string" || !args["category"].Required {
		t.Fatalf("expected required category:string arg, got %+v", args["category"])
	}
	if !slices.Equal(args["category"].Enum, []any{"dog", "cat"}) {
		t.Fatalf("expected category enum from nested external ref, got %+v", args["category"].Enum)
	}
	if action.Request.Body["category"] != "{category}" {
		t.Fatalf("expected request body category mapping, got %+v", action.Request.Body)
	}
}

func TestGenerateFromOpenAPIFileKeepsReferencedDocumentContextAfterSiblingOverrides(t *testing.T) {
	dir := t.TempDir()
	rootPath := writeOpenAPITestFile(t, dir, "openapi.yaml", `openapi: 3.0.3
info:
  title: Override Ref API
  version: 1.0.0
servers:
  - url: https://api.example.com
paths:
  /pets:
    post:
      operationId: createPet
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: ./schemas/pet-create.yaml#/PetCreate
              description: Pet payload override
      responses:
        '201':
          description: created
          content:
            application/json:
              schema:
                type: object
`)
	writeOpenAPITestFile(t, dir, "schemas/pet-create.yaml", `PetCreate:
  type: object
  required: [category]
  properties:
    category:
      $ref: ./common/category.yaml#/Category
`)
	writeOpenAPITestFile(t, dir, "schemas/common/category.yaml", `Category:
  type: string
  enum: [dog, cat]
`)

	manifest, err := GenerateFromOpenAPIFile(rootPath)
	if err != nil {
		t.Fatalf("GenerateFromOpenAPIFile failed: %v", err)
	}

	action, ok := manifest.Actions["createpet"]
	if !ok {
		t.Fatalf("expected createpet action")
	}
	args := generatedActionArgsByName(action)
	if !slices.Equal(args["category"].Enum, []any{"dog", "cat"}) {
		t.Fatalf("expected nested ref enum to survive sibling overrides, got %+v", args["category"].Enum)
	}
}

func TestGenerateFromOpenAPIFileSupportsExternalPathItemsAndParameters(t *testing.T) {
	dir := t.TempDir()
	rootPath := writeOpenAPITestFile(t, dir, "openapi.yaml", `openapi: 3.0.3
info:
  title: Split Paths API
  version: 1.0.0
servers:
  - url: https://api.example.com
paths:
  /pets/{petId}:
    $ref: ./paths/pet-by-id.yaml#/PetByID
`)
	writeOpenAPITestFile(t, dir, "paths/pet-by-id.yaml", `PetByID:
  parameters:
    - $ref: ../components/parameters.yaml#/PetID
  get:
    operationId: getPet
    parameters:
      - $ref: ../components/parameters.yaml#/TraceID
    responses:
      '200':
        description: ok
        content:
          application/json:
            schema:
              type: object
`)
	writeOpenAPITestFile(t, dir, "components/parameters.yaml", `PetID:
  name: petId
  in: path
  required: true
  schema:
    type: string
TraceID:
  name: X-Trace-Id
  in: header
  schema:
    type: string
`)

	manifest, err := GenerateFromOpenAPIFile(rootPath)
	if err != nil {
		t.Fatalf("GenerateFromOpenAPIFile failed: %v", err)
	}

	action, ok := manifest.Actions["getpet"]
	if !ok {
		t.Fatalf("expected getpet action")
	}
	if action.Request.PathParams["petId"] != "{petId}" {
		t.Fatalf("expected path param mapping, got %+v", action.Request.PathParams)
	}
	if action.Request.Headers["X-Trace-Id"] != "{X-Trace-Id}" {
		t.Fatalf("expected header param mapping, got %+v", action.Request.Headers)
	}
}

func TestGenerateFromOpenAPIFileRejectsRemoteExternalRefs(t *testing.T) {
	dir := t.TempDir()
	rootPath := writeOpenAPITestFile(t, dir, "openapi.yaml", `openapi: 3.0.3
info:
  title: Remote Ref API
  version: 1.0.0
servers:
  - url: https://api.example.com
paths:
  /pets:
    post:
      operationId: createPet
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: https://example.com/schemas/pet.yaml#/Pet
      responses:
        '201':
          description: created
          content:
            application/json:
              schema:
                type: object
`)

	_, err := GenerateFromOpenAPIFile(rootPath)
	if err == nil {
		t.Fatal("expected remote external ref to fail")
	}
	if !strings.Contains(err.Error(), "only local refs and relative file refs are supported") {
		t.Fatalf("expected explicit remote ref rejection, got %v", err)
	}
}

func TestGenerateFromOpenAPIFileRejectsNonPointerExternalRefFragments(t *testing.T) {
	dir := t.TempDir()
	rootPath := writeOpenAPITestFile(t, dir, "openapi.yaml", `openapi: 3.0.3
info:
  title: Invalid Fragment API
  version: 1.0.0
servers:
  - url: https://api.example.com
paths:
  /pets:
    post:
      operationId: createPet
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: ./schemas/pet.yaml#Pet
      responses:
        '201':
          description: created
          content:
            application/json:
              schema:
                type: object
`)
	writeOpenAPITestFile(t, dir, "schemas/pet.yaml", `Pet:
  type: object
`)

	_, err := GenerateFromOpenAPIFile(rootPath)
	if err == nil {
		t.Fatal("expected non-pointer fragment to fail")
	}
	if !strings.Contains(err.Error(), "only JSON Pointer fragments are supported") {
		t.Fatalf("expected explicit invalid fragment error, got %v", err)
	}
}

func TestGenerateFromOpenAPIRejectsExternalFileRefsWithoutFileContext(t *testing.T) {
	spec := `openapi: 3.0.3
info:
  title: External Ref API
  version: 1.0.0
servers:
  - url: https://api.example.com
paths:
  /pets:
    post:
      operationId: createPet
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: ./schemas/pet.yaml#/Pet
      responses:
        '201':
          description: created
          content:
            application/json:
              schema:
                type: object
`

	_, err := GenerateFromOpenAPI([]byte(spec))
	if err == nil {
		t.Fatal("expected byte-based generation to reject external file refs")
	}
	if !strings.Contains(err.Error(), "external file refs require OpenAPI file input") {
		t.Fatalf("expected explicit file-context error, got %v", err)
	}
}

func TestGenerateFromOpenAPIPreservesNumberTypes(t *testing.T) {
	spec := `openapi: 3.0.3
info:
  title: Pricing API
  version: 1.0.0
servers:
  - url: https://api.example.com
paths:
  /prices:
    post:
      operationId: createPrice
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [amount]
              properties:
                amount:
                  type: number
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema:
                type: object
`

	manifest, err := GenerateFromOpenAPI([]byte(spec))
	if err != nil {
		t.Fatalf("GenerateFromOpenAPI failed: %v", err)
	}

	action := manifest.Actions["createprice"]
	if len(action.Args) != 1 {
		t.Fatalf("expected one arg, got %d", len(action.Args))
	}
	if action.Args[0].Name != "amount" || action.Args[0].Type != "number" {
		t.Fatalf("expected amount:number, got %+v", action.Args[0])
	}
}

func TestGenerateFromOpenAPIWithOptionsNameOverride(t *testing.T) {
	manifest, err := GenerateFromOpenAPIWithOptions([]byte(openAPIFilterFixture), OpenAPIGenerateOptions{
		NameOverride: "Internal Admin API",
	})
	if err != nil {
		t.Fatalf("GenerateFromOpenAPIWithOptions failed: %v", err)
	}
	if manifest.Name != "internal-admin-api" {
		t.Fatalf("expected normalized name override, got %q", manifest.Name)
	}
}

func TestGenerateFromOpenAPIWithOptionsFiltersByTag(t *testing.T) {
	manifest, err := GenerateFromOpenAPIWithOptions([]byte(openAPIFilterFixture), OpenAPIGenerateOptions{
		Tags: []string{"ADMIN"},
	})
	if err != nil {
		t.Fatalf("GenerateFromOpenAPIWithOptions failed: %v", err)
	}

	if len(manifest.Actions) != 2 {
		t.Fatalf("expected 2 admin actions, got %+v", manifest.Actions)
	}
	if _, ok := manifest.Actions["listusers"]; !ok {
		t.Fatalf("expected unsuffixed listusers action after filtering, got %+v", manifest.Actions)
	}
	if _, ok := manifest.Actions["listaudit"]; !ok {
		t.Fatalf("expected listaudit action after filtering, got %+v", manifest.Actions)
	}
	for actionName := range manifest.Actions {
		if strings.HasPrefix(actionName, "listusers-") {
			t.Fatalf("expected filter to run before action key disambiguation, got %q", actionName)
		}
	}
}

func TestGenerateFromOpenAPIWithOptionsFiltersByPathPrefix(t *testing.T) {
	manifest, err := GenerateFromOpenAPIWithOptions([]byte(openAPIFilterFixture), OpenAPIGenerateOptions{
		PathPrefixes: []string{"admin"},
	})
	if err != nil {
		t.Fatalf("GenerateFromOpenAPIWithOptions failed: %v", err)
	}

	gotActions := sortedGeneratedActionKeys(manifest.Actions)
	wantActions := []string{"listaudit", "listusers"}
	if !slices.Equal(gotActions, wantActions) {
		t.Fatalf("path-prefix filter actions = %v, want %v", gotActions, wantActions)
	}
}

func TestGenerateFromOpenAPIWithOptionsSkipsBrokenExcludedPathPrefix(t *testing.T) {
	spec := `openapi: 3.0.3
info:
  title: Filter Prefix Safety API
  version: 1.0.0
servers:
  - url: https://api.example.com
paths:
  /public/broken:
    $ref: '#/components/pathItems/MissingPath'
  /admin/users:
    get:
      operationId: listUsers
      tags: [admin]
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema:
                type: object
`

	manifest, err := GenerateFromOpenAPIWithOptions([]byte(spec), OpenAPIGenerateOptions{
		PathPrefixes: []string{"/admin"},
	})
	if err != nil {
		t.Fatalf("expected excluded broken path prefix to be ignored, got %v", err)
	}
	if _, ok := manifest.Actions["listusers"]; !ok {
		t.Fatalf("expected admin action to remain available, got %+v", manifest.Actions)
	}
}

func TestGenerateFromOpenAPIWithOptionsFiltersUseANDSemantics(t *testing.T) {
	_, err := GenerateFromOpenAPIWithOptions([]byte(openAPIFilterFixture), OpenAPIGenerateOptions{
		Tags:         []string{"public"},
		PathPrefixes: []string{"/admin"},
	})
	if err == nil {
		t.Fatal("expected filter combination with no matches to fail")
	}
	if !strings.Contains(err.Error(), "no OpenAPI operations matched the requested filters") {
		t.Fatalf("expected no-match filter error, got %v", err)
	}
}

func sortedGeneratedActionKeys(actions map[string]ServiceAction) []string {
	keys := make([]string, 0, len(actions))
	for key := range actions {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

const openAPIFilterFixture = `openapi: 3.0.3
info:
  title: Filter Fixture API
  version: 1.0.0
servers:
  - url: https://api.example.com
paths:
  /public/users:
    get:
      operationId: listUsers
      tags: [public]
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema:
                type: object
  /admin/users:
    get:
      operationId: listUsers
      tags: [admin]
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema:
                type: object
  /admin/audit:
    get:
      operationId: listAudit
      tags: [admin]
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema:
                type: object
`

func TestGenerateFromOpenAPIAcceptsCaseInsensitiveJSONMediaType(t *testing.T) {
	spec := `openapi: 3.0.3
info:
  title: Media Type Case API
  version: 1.0.0
servers:
  - url: https://api.example.com
paths:
  /items:
    post:
      operationId: createItem
      requestBody:
        required: true
        content:
          Application/JSON:
            schema:
              type: object
              required: [name]
              properties:
                name:
                  type: string
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema:
                type: object
`

	manifest, err := GenerateFromOpenAPI([]byte(spec))
	if err != nil {
		t.Fatalf("GenerateFromOpenAPI failed: %v", err)
	}

	action, ok := manifest.Actions["createitem"]
	if !ok {
		t.Fatalf("expected createitem action")
	}

	foundName := false
	for _, arg := range action.Args {
		if arg.Name == "name" {
			foundName = true
			break
		}
	}
	if !foundName {
		t.Fatalf("expected request body arg 'name' for case-insensitive JSON media type, got %+v", action.Args)
	}
}

func TestGenerateFromOpenAPIOperationSecurityOverride(t *testing.T) {
	spec := `openapi: 3.0.3
info:
  title: Security API
  version: 1.0.0
servers:
  - url: https://api.example.com
security:
  - BearerAuth: []
components:
  securitySchemes:
    BearerAuth:
      type: http
      scheme: bearer
    ApiKeyAuth:
      type: apiKey
      in: query
      name: api_key
paths:
  /search:
    get:
      operationId: search
      security:
        - ApiKeyAuth: []
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema:
                type: object
`

	manifest, err := GenerateFromOpenAPI([]byte(spec))
	if err != nil {
		t.Fatalf("GenerateFromOpenAPI failed: %v", err)
	}

	if manifest.Auth.Type != "bearer" {
		t.Fatalf("expected service-level bearer auth, got %+v", manifest.Auth)
	}

	action := manifest.Actions["search"]
	if action.Auth == nil {
		t.Fatalf("expected operation-level auth override")
	}
	if action.Auth.Type != "query" || action.Auth.QueryParam != "api_key" {
		t.Fatalf("unexpected operation auth: %+v", action.Auth)
	}
}

func TestGenerateFromOpenAPIActionKeyDisambiguationIsDeterministic(t *testing.T) {
	spec := `openapi: 3.0.3
info:
  title: Collision API
  version: 1.0.0
servers:
  - url: https://api.example.com
paths:
  /alpha:
    get:
      operationId: duplicate
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema:
                type: object
  /beta:
    get:
      operationId: duplicate
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema:
                type: object
  /gamma:
    get:
      operationId: duplicate
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema:
                type: object
`

	manifest, err := GenerateFromOpenAPI([]byte(spec))
	if err != nil {
		t.Fatalf("GenerateFromOpenAPI failed: %v", err)
	}

	if _, ok := manifest.Actions["duplicate"]; !ok {
		t.Fatalf("expected base key to exist")
	}

	betaKey := "duplicate-" + shortStableHash("get /beta")[:6]
	if _, ok := manifest.Actions[betaKey]; !ok {
		t.Fatalf("expected deterministic beta key %q", betaKey)
	}

	gammaKey := "duplicate-" + shortStableHash("get /gamma")[:6]
	if _, ok := manifest.Actions[gammaKey]; !ok {
		t.Fatalf("expected deterministic gamma key %q", gammaKey)
	}

	for key := range manifest.Actions {
		if strings.HasSuffix(key, "-2") || strings.HasSuffix(key, "-3") {
			t.Fatalf("unexpected ordinal collision suffix in key %q", key)
		}
	}
}

func TestGenerateFromOpenAPICompositeSchemas(t *testing.T) {
	spec := `openapi: 3.0.3
info:
  title: Composite API
  version: 1.0.0
servers:
  - url: https://api.example.com
paths:
  /merge:
    post:
      operationId: mergePayload
      requestBody:
        required: true
        content:
          application/json:
            schema:
              allOf:
                - type: object
                  required: [name]
                  properties:
                    name:
                      type: string
                - type: object
                  required: [price]
                  properties:
                    price:
                      type: number
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema:
                type: object
  /choose:
    post:
      operationId: choosePayload
      requestBody:
        required: true
        content:
          application/json:
            schema:
              oneOf:
                - type: object
                  required: [mode]
                  properties:
                    mode:
                      type: string
                - type: object
                  required: [count]
                  properties:
                    count:
                      type: integer
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema:
                type: object
  /list:
    get:
      operationId: listItems
      parameters:
        - name: filter
          in: query
          schema:
            anyOf:
              - type: integer
              - type: string
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema:
                type: object
`

	manifest, err := GenerateFromOpenAPI([]byte(spec))
	if err != nil {
		t.Fatalf("GenerateFromOpenAPI failed: %v", err)
	}

	merge := manifest.Actions["mergepayload"]
	mergeArgs := map[string]ActionArg{}
	for _, arg := range merge.Args {
		mergeArgs[arg.Name] = arg
	}
	if !mergeArgs["name"].Required || !mergeArgs["price"].Required {
		t.Fatalf("expected allOf required fields to be required, got %+v", merge.Args)
	}
	if mergeArgs["price"].Type != "number" {
		t.Fatalf("expected allOf merged price:number, got %+v", mergeArgs["price"])
	}

	choose := manifest.Actions["choosepayload"]
	if len(choose.Args) != 1 {
		t.Fatalf("expected one arg from oneOf primary schema, got %d", len(choose.Args))
	}
	if choose.Args[0].Name != "mode" || !choose.Args[0].Required {
		t.Fatalf("expected oneOf primary arg mode to be required (from primary schema), got %+v", choose.Args[0])
	}

	list := manifest.Actions["listitems"]
	if len(list.Args) != 1 || list.Args[0].Name != "filter" || list.Args[0].Type != "integer" {
		t.Fatalf("expected anyOf first schema type for filter arg, got %+v", list.Args)
	}
}

func TestGenerateFromOpenAPIComplexOneOfFallsBackToOpaqueBody(t *testing.T) {
	spec := `openapi: 3.0.3
info:
  title: Complex OneOf API
  version: 1.0.0
servers:
  - url: https://api.example.com
paths:
  /events:
    post:
      operationId: createEvent
      requestBody:
        required: true
        content:
          application/json:
            schema:
              oneOf:
                - type: object
                  properties:
                    kind:
                      type: string
                    a:
                      type: string
                - type: object
                  properties:
                    kind:
                      type: string
                    b:
                      type: integer
                - type: object
                  properties:
                    kind:
                      type: string
                    c:
                      type: boolean
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema:
                type: object
`

	manifest, err := GenerateFromOpenAPI([]byte(spec))
	if err != nil {
		t.Fatalf("GenerateFromOpenAPI failed: %v", err)
	}

	action := manifest.Actions["createevent"]
	if len(action.Args) != 0 {
		t.Fatalf("expected no args for opaque body (raw JSON passthrough via --json), got %+v", action.Args)
	}
	if len(action.Request.Body) != 0 {
		t.Fatalf("expected empty body template for opaque schema, got %+v", action.Request.Body)
	}
}

func TestGenerateFromOpenAPIOneOfWithDiscriminatorFallsBackToOpaque(t *testing.T) {
	spec := `openapi: 3.0.3
info:
  title: Discriminator API
  version: 1.0.0
servers:
  - url: https://api.example.com
paths:
  /animals:
    post:
      operationId: createAnimal
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              discriminator:
                propertyName: petType
              oneOf:
                - type: object
                  properties:
                    petType:
                      type: string
                    bark:
                      type: boolean
                - type: object
                  properties:
                    petType:
                      type: string
                    hunts:
                      type: boolean
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema:
                type: object
`

	manifest, err := GenerateFromOpenAPI([]byte(spec))
	if err != nil {
		t.Fatalf("GenerateFromOpenAPI failed: %v", err)
	}

	action := manifest.Actions["createanimal"]
	if len(action.Args) != 0 {
		t.Fatalf("expected no args for opaque body with discriminator, got %+v", action.Args)
	}
	if len(action.Request.Body) != 0 {
		t.Fatalf("expected empty body template for opaque schema, got %+v", action.Request.Body)
	}
}

func TestGenerateFromOpenAPIAllOfTypeConflictUsesLastWriter(t *testing.T) {
	spec := `openapi: 3.0.3
info:
  title: Conflict API
  version: 1.0.0
servers:
  - url: https://api.example.com
paths:
  /conflict:
    post:
      operationId: conflictPayload
      requestBody:
        required: true
        content:
          application/json:
            schema:
              allOf:
                - type: object
                  required: [value]
                  properties:
                    value:
                      type: string
                - type: object
                  discriminator:
                    propertyName: mode
                  properties:
                    mode:
                      type: string
                    value:
                      type: integer
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema:
                type: object
`

	manifest, err := GenerateFromOpenAPI([]byte(spec))
	if err != nil {
		t.Fatalf("GenerateFromOpenAPI failed: %v", err)
	}

	action := manifest.Actions["conflictpayload"]
	argsByName := map[string]ActionArg{}
	for _, arg := range action.Args {
		argsByName[arg.Name] = arg
	}
	if argsByName["value"].Type != "integer" {
		t.Fatalf("expected last-writer type override to integer, got %+v", argsByName["value"])
	}
	if !strings.Contains(action.Description, `allOf conflict for property "value"`) {
		t.Fatalf("expected warning annotation in description, got %q", action.Description)
	}
}

func TestGenerateFromOpenAPISimpleOneOfStillFlattens(t *testing.T) {
	spec := `openapi: 3.0.3
info:
  title: Simple OneOf API
  version: 1.0.0
servers:
  - url: https://api.example.com
paths:
  /choose:
    post:
      operationId: chooseSimple
      requestBody:
        required: true
        content:
          application/json:
            schema:
              oneOf:
                - type: object
                  required: [mode]
                  properties:
                    mode:
                      type: string
                - type: object
                  required: [count]
                  properties:
                    count:
                      type: integer
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema:
                type: object
`

	manifest, err := GenerateFromOpenAPI([]byte(spec))
	if err != nil {
		t.Fatalf("GenerateFromOpenAPI failed: %v", err)
	}

	action := manifest.Actions["choosesimple"]
	if len(action.Args) != 1 {
		t.Fatalf("expected one arg from first oneOf variant, got %+v", action.Args)
	}
	if action.Args[0].Name != "mode" {
		t.Fatalf("expected flattened mode arg from primary schema, got %+v", action.Args[0])
	}
}

func TestE2EOpaqueBodyRawPassthrough(t *testing.T) {
	spec := `openapi: 3.0.3
info:
  title: Polymorphic API
  version: 1.0.0
servers:
  - url: https://api.example.com
paths:
  /events:
    post:
      operationId: createEvent
      requestBody:
        required: true
        content:
          application/json:
            schema:
              oneOf:
                - type: object
                  properties:
                    type: {type: string}
                    a: {type: string}
                - type: object
                  properties:
                    type: {type: string}
                    b: {type: integer}
                - type: object
                  properties:
                    type: {type: string}
                    c: {type: boolean}
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema:
                type: object
`
	manifest, err := GenerateFromOpenAPI([]byte(spec))
	if err != nil {
		t.Fatalf("GenerateFromOpenAPI: %v", err)
	}

	defs, err := ToActionDefinitions(manifest)
	if err != nil {
		t.Fatalf("ToActionDefinitions: %v", err)
	}

	if len(defs) != 1 {
		t.Fatalf("expected 1 action def, got %d", len(defs))
	}

	def := defs[0]

	if def.Adapter.RequestBody != "" {
		t.Fatalf("expected empty RequestBody template for opaque body, got %q", def.Adapter.RequestBody)
	}

	if def.InputSchema != nil && len(def.InputSchema.Properties) > 0 {
		t.Fatalf("expected no input properties for opaque body, got %+v", def.InputSchema.Properties)
	}

	if !def.InputSchema.AdditionalProperties {
		t.Fatal("expected additionalProperties=true so --json input passes through as raw body")
	}
}

func TestGenerateFromOpenAPIParityContract(t *testing.T) {
	spec := `openapi: 3.0.3
info:
  title: Parity Contract API
  version: 1.0.0
servers:
  - url: https://contract.example.com
security:
  - ApiKeyAuth: []
components:
  securitySchemes:
    ApiKeyAuth:
      type: apiKey
      in: query
      name: api_key
paths:
  /items/{id}:
    get:
      operationId: getItem
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
        - name: includeMeta
          in: query
          required: false
          schema:
            type: boolean
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema:
                type: object
  /items:
    post:
      operationId: createItem
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [name]
              properties:
                name:
                  type: string
                quantity:
                  type: number
      responses:
        '201':
          description: created
          content:
            application/json:
              schema:
                type: object
`

	manifest, err := GenerateFromOpenAPI([]byte(spec))
	if err != nil {
		t.Fatalf("GenerateFromOpenAPI failed: %v", err)
	}

	if manifest.BaseURL != "https://contract.example.com" {
		t.Fatalf("unexpected baseURL: %s", manifest.BaseURL)
	}
	if manifest.Auth.Type != "query" || manifest.Auth.QueryParam != "api_key" {
		t.Fatalf("unexpected auth mapping: %+v", manifest.Auth)
	}

	getAction, ok := manifest.Actions["getitem"]
	if !ok {
		t.Fatalf("expected getitem action")
	}
	if getAction.Method != "GET" || getAction.Path != "/items/{id}" {
		t.Fatalf("unexpected get action route: %+v", getAction)
	}
	if getAction.Request.PathParams["id"] != "{id}" {
		t.Fatalf("expected path param mapping id -> {id}, got %+v", getAction.Request.PathParams)
	}
	if getAction.Request.Query["includeMeta"] != "{includeMeta}" {
		t.Fatalf("expected query param mapping includeMeta -> {includeMeta}, got %+v", getAction.Request.Query)
	}

	createAction, ok := manifest.Actions["createitem"]
	if !ok {
		t.Fatalf("expected createitem action")
	}
	if createAction.Method != "POST" || createAction.Path != "/items" {
		t.Fatalf("unexpected create action route: %+v", createAction)
	}

	args := map[string]ActionArg{}
	for _, arg := range createAction.Args {
		args[arg.Name] = arg
	}
	if !args["name"].Required || args["name"].Type != "string" {
		t.Fatalf("expected required name:string arg, got %+v", args["name"])
	}
	if args["quantity"].Required || args["quantity"].Type != "number" {
		t.Fatalf("expected optional quantity:number arg, got %+v", args["quantity"])
	}
}

func TestGenerateFromOpenAPIUsesServicePrefixForNumericOrEmptyNames(t *testing.T) {
	tests := []struct {
		name     string
		title    string
		expected string
	}{
		{name: "empty title uses openapi-service", title: "", expected: "openapi-service"},
		{name: "numeric title gets service prefix", title: "123 payments", expected: "service-123-payments"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := `openapi: 3.0.3
info:
  title: ` + tt.title + `
  version: 1.0.0
servers:
  - url: https://api.example.com
paths:
  /health:
    get:
      operationId: health
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema:
                type: object
`

			manifest, err := GenerateFromOpenAPI([]byte(spec))
			if err != nil {
				t.Fatalf("GenerateFromOpenAPI failed: %v", err)
			}
			if manifest.Name != tt.expected {
				t.Fatalf("expected name %q, got %q", tt.expected, manifest.Name)
			}
		})
	}
}

func TestGenerateFromOpenAPIPreservesParentRequiredWithSimpleOneOf(t *testing.T) {
	spec := `openapi: 3.0.3
info:
  title: OneOf Required API
  version: 1.0.0
servers:
  - url: https://api.example.com
paths:
  /items:
    post:
      operationId: createItem
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [name, kind]
              properties:
                name:
                  type: string
              oneOf:
                - properties:
                    kind:
                      type: string
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema:
                type: object
`

	manifest, err := GenerateFromOpenAPI([]byte(spec))
	if err != nil {
		t.Fatalf("GenerateFromOpenAPI failed: %v", err)
	}

	action, ok := manifest.Actions["createitem"]
	if !ok {
		t.Fatalf("expected createitem action")
	}

	argMap := map[string]ActionArg{}
	for _, arg := range action.Args {
		argMap[arg.Name] = arg
	}

	if !argMap["name"].Required {
		t.Fatalf("expected parent-level required field 'name' to stay required after oneOf flattening")
	}
	if !argMap["kind"].Required {
		t.Fatalf("expected parent-level required field 'kind' to stay required after oneOf flattening")
	}
}

func writeOpenAPITestFile(t *testing.T, dir string, relativePath string, contents string) string {
	t.Helper()

	path := filepath.Join(dir, filepath.FromSlash(relativePath))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) failed: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(strings.TrimSpace(contents)+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) failed: %v", path, err)
	}
	return path
}

func generatedActionArgsByName(action ServiceAction) map[string]ActionArg {
	args := make(map[string]ActionArg, len(action.Args))
	for _, arg := range action.Args {
		args[arg.Name] = arg
	}
	return args
}
