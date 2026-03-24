package skills

import (
	"net/http"
	"net/http/httptest"
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
	if postAction.Risk.Level != "medium" || !postAction.Risk.Mutating {
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

	manifest, err := GenerateFromOpenAPI([]byte(spec))
	if err != nil {
		t.Fatalf("GenerateFromOpenAPI failed: %v", err)
	}

	if manifest.BaseURL != "https://example.com" {
		t.Fatalf("expected fallback base URL, got %s", manifest.BaseURL)
	}
	if manifest.Auth.Type != "none" {
		t.Fatalf("expected auth.type=none, got %+v", manifest.Auth)
	}
	if manifest.Version != "2.0.0" {
		t.Fatalf("expected normalized version 2.0.0, got %s", manifest.Version)
	}

	deleteAction, ok := manifest.Actions["delete-items-id"]
	if !ok {
		t.Fatalf("expected generated DELETE action")
	}
	if deleteAction.Risk.Level != "high" || !deleteAction.Risk.Mutating {
		t.Fatalf("unexpected DELETE risk: %+v", deleteAction.Risk)
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
