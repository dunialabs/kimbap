package services

import (
	"net/http"
	"net/http/httptest"
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

	manifest, err := GenerateFromOpenAPIURL(server.URL)
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

	manifest, err := GenerateFromOpenAPIURL(server.URL)
	if err != nil {
		t.Fatalf("GenerateFromOpenAPIURL failed: %v", err)
	}

	if manifest.BaseURL != server.URL+"/api" {
		t.Fatalf("expected baseURL to resolve against fetched origin, got %q", manifest.BaseURL)
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
