import { load } from 'js-yaml';
import { convertOpenApiToRestApiFormat } from '../lib/rest-api-utils';

type AssertContext = {
  failures: string[];
};

function assert(condition: boolean, message: string, ctx: AssertContext): void {
  if (!condition) {
    ctx.failures.push(message);
  }
}

const paritySpecYAML = `openapi: 3.0.3
info:
  title: Parity Contract API
  version: 1.0.0
servers:
  - url: https://contract.example.com
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
`;

function main(): void {
  const spec = load(paritySpecYAML);
  const converted = convertOpenApiToRestApiFormat(spec);
  const ctx: AssertContext = { failures: [] };

  assert(converted.baseUrl === 'https://contract.example.com', `expected baseUrl=https://contract.example.com, got ${converted.baseUrl}`, ctx);
  assert(converted.auth?.type === 'query_param', `expected auth.type=query_param, got ${converted.auth?.type}`, ctx);
  assert(converted.auth?.param === 'api_key', `expected auth.param=api_key, got ${converted.auth?.param}`, ctx);

  const getTool = converted.tools.find((tool) => tool.name === 'getItem');
  assert(!!getTool, 'expected getItem tool', ctx);
  if (getTool) {
    assert(getTool.method === 'GET', `expected getItem.method=GET, got ${getTool.method}`, ctx);
    assert(getTool.endpoint === '/items/{id}', `expected getItem.endpoint=/items/{id}, got ${getTool.endpoint}`, ctx);
    const idParam = getTool.parameters.find((p) => p.name === 'id' && p.location === 'path');
    assert(!!idParam, 'expected getItem path param id', ctx);
    if (idParam) {
      assert(idParam.type === 'string', `expected id type=string, got ${idParam.type}`, ctx);
      assert(idParam.required === true, `expected id required=true, got ${idParam.required}`, ctx);
    }
    const includeMeta = getTool.parameters.find((p) => p.name === 'includeMeta' && p.location === 'query');
    assert(!!includeMeta, 'expected getItem query param includeMeta', ctx);
    if (includeMeta) {
      assert(includeMeta.type === 'boolean', `expected includeMeta type=boolean, got ${includeMeta.type}`, ctx);
      assert(includeMeta.required === false, `expected includeMeta required=false, got ${includeMeta.required}`, ctx);
    }
  }

  const createTool = converted.tools.find((tool) => tool.name === 'createItem');
  assert(!!createTool, 'expected createItem tool', ctx);
  if (createTool) {
    assert(createTool.method === 'POST', `expected createItem.method=POST, got ${createTool.method}`, ctx);
    assert(createTool.endpoint === '/items', `expected createItem.endpoint=/items, got ${createTool.endpoint}`, ctx);
    const nameParam = createTool.parameters.find((p) => p.name === 'name' && p.location === 'body');
    assert(!!nameParam, 'expected createItem body param name', ctx);
    if (nameParam) {
      assert(nameParam.type === 'string', `expected name type=string, got ${nameParam.type}`, ctx);
      assert(nameParam.required === true, `expected name required=true, got ${nameParam.required}`, ctx);
    }
    const quantityParam = createTool.parameters.find((p) => p.name === 'quantity' && p.location === 'body');
    assert(!!quantityParam, 'expected createItem body param quantity', ctx);
    if (quantityParam) {
      assert(quantityParam.type === 'number', `expected quantity type=number, got ${quantityParam.type}`, ctx);
      assert(quantityParam.required === false, `expected quantity required=false, got ${quantityParam.required}`, ctx);
    }
  }

  if (ctx.failures.length > 0) {
    throw new Error(`OpenAPI parity contract failed:\n- ${ctx.failures.join('\n- ')}`);
  }

  console.log('OpenAPI parity contract check passed');
}

main();
