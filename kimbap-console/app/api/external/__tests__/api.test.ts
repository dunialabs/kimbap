/**
 * External API Test Suite
 *
 * Usage:
 *   npx tsx app/api/external/__tests__/api.test.ts [endpoint]
 *
 * Examples:
 *   npx tsx app/api/external/__tests__/api.test.ts              # Run all tests
 *   npx tsx app/api/external/__tests__/api.test.ts auth/init    # Run specific endpoint tests
 *   npx tsx app/api/external/__tests__/api.test.ts token-all     # Run all token-related endpoint tests
 *
 * Make sure the dev server is running: npm run dev
 */

const BASE_URL = process.env.API_BASE_URL || 'http://localhost:3000';
const OWNER_TOKEN = '3d10f57081c4a761b2b7f55072bffcd2c91abba003445085c9d3311805b2e0f66b88a84847fc2fccfb1d023691e2a4d445b7bacbd9c8f39bc686fdbdff74579b';
const TARGET_ENDPOINT = process.argv[2];
const tokenFlowContext = {
  createdTokenIds: [] as string[],
  enabled: false,
};

interface TestCase {
  name: string;
  run: () => Promise<void>;
}

interface TestSuite {
  endpoint: string;
  tests: TestCase[];
}

const suites: TestSuite[] = [];

function suite(endpoint: string, tests: TestCase[]) {
  suites.push({ endpoint, tests });
}

function logEndpointResult(endpoint: string, status: number, data: any, phase?: string) {
  const phaseText = phase ? ` ${phase}` : '';
  console.log('');
  console.log(`[${endpoint}]${phaseText}`);
  console.log('      Status:', status, '| Response:', JSON.stringify(data));
}

function rememberCreatedTokenIdsFromResponse(data: any) {
  if (!tokenFlowContext.enabled) return;
  const ids = data?.data?.tokens?.map((t: any) => t?.tokenId).filter((id: any) => typeof id === 'string');
  if (Array.isArray(ids) && ids.length > 0) {
    tokenFlowContext.createdTokenIds.push(...ids);
    console.log('    Token Flow Context: remembered tokenIds =', JSON.stringify(tokenFlowContext.createdTokenIds));
  }
}

function getTokenIdForMutation(preferConsume = false): string | null {
  if (tokenFlowContext.createdTokenIds.length === 0) return null;
  return preferConsume ? (tokenFlowContext.createdTokenIds.shift() || null) : tokenFlowContext.createdTokenIds[0];
}

// ============================================================
// POST /api/external/auth/init
// ============================================================

suite('auth/init', [
  {
    name: 'missing masterPwd should return E1001',
    run: async () => {
      const res = await fetch(`${BASE_URL}/api/external/auth/init`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({}),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'empty masterPwd should return E1001',
    run: async () => {
      const res = await fetch(`${BASE_URL}/api/external/auth/init`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ masterPwd: '' }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'valid masterPwd should return 201 or E3007',
    run: async () => {
      const res = await fetch(`${BASE_URL}/api/external/auth/init`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ masterPwd: 'test-master-password-123' }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
]);

// ============================================================
// POST /api/external/proxy
// ============================================================

suite('proxy', [
  {
    name: 'should return 200 with proxy info or E3001 if not initialized',
    run: async () => {
      const res = await fetch(`${BASE_URL}/api/external/proxy`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({}),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
]);

// ============================================================
// POST /api/external/proxy/update
// ============================================================

suite('proxy/update', [
  {
    name: 'missing Authorization should return E2001',
    run: async () => {
      const res = await fetch(`${BASE_URL}/api/external/proxy/update`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ proxyId: 1, proxyName: 'New Name' }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'missing proxyId should return E1001',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/proxy/update`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({ proxyName: 'New Name' }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'invalid proxyId should return E1003',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/proxy/update`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({ proxyId: 9999, proxyName: 'New Name' }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'valid request should return 200',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/proxy/update`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({ proxyId: 1, proxyName: 'WCL MCP Server' }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
]);

// ============================================================
// POST /api/external/tokens
// ============================================================

suite('tokens', [
  {
    name: 'missing Authorization should return E2001',
    run: async () => {
      const res = await fetch(`${BASE_URL}/api/external/tokens`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({}),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'valid request should return 200 with tokens list',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/tokens`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({}),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
]);

// ============================================================
// POST /api/external/tokens/get
// ============================================================

suite('tokens/get', [
  {
    name: 'missing Authorization should return E2001',
    run: async () => {
      const res = await fetch(`${BASE_URL}/api/external/tokens/get`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ tokenId: 'test-token-id' }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'missing tokenId should return E1001',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/tokens/get`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({}),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'non-existent tokenId should return E3003',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/tokens/get`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({ tokenId: 'non-existent-token-id' }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'valid tokenId should return 200 with token details',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/tokens/get`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({ tokenId: '203f319b51bea91aa4ece9129f00007d' }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
]);

// ============================================================
// POST /api/external/scopes
// ============================================================

suite('scopes', [
  {
    name: 'missing Authorization should return E2001',
    run: async () => {
      const res = await fetch(`${BASE_URL}/api/external/scopes`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({}),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'valid request should return 200 with scopes list',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/scopes`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({}),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
]);

// ============================================================
// POST /api/external/tokens/create
// ============================================================

suite('tokens/create', [
  {
    name: 'missing Authorization should return E2001',
    run: async () => {
      const res = await fetch(`${BASE_URL}/api/external/tokens/create`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ tokens: [{ name: 'Test', role: 3 }] }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'missing tokens array should return E1001',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/tokens/create`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({}),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'empty tokens array should return E1001',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/tokens/create`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({ tokens: [] }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'role=1 (owner) should return E1003',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/tokens/create`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({ tokens: [{ name: 'Test', role: 1 }] }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'valid batch tokens should return 201',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const flowNamespace = `token-all-ns-${Date.now()}`;
      const flowTag = `token-all-tag-${Math.random().toString(16).slice(2, 8)}`;
      const res = await fetch(`${BASE_URL}/api/external/tokens/create`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({
          tokens: [
            {
              "name": "Developer Token 01 admin",
              "role": 2,                    // 2=admin, 3=member (cannot create owner)
              "notes": "admin",
              "expiresAt": 1864967156,      // Unix timestamp, 0=never
              "rateLimit": 1000,
              "namespace": flowNamespace,
              "tags": [flowTag, "admin"],
              "permissions": [
                {
                  "toolId": "2e927a929a85408bbb61b235e31c0745",
                  "functions": [
                    {
                      "funcName": "githubListRepositories",
                      "enabled": false
                    }
                  ],
                  "resources": [
                  ]
                }
              ]
            },
            {
              "name": "Developer Token 02 member",
              "role": 3,                    // 2=admin, 3=member (cannot create owner)
              "notes": "member",
              "expiresAt": 1864967156,      // Unix timestamp, 0=never
              "rateLimit": 1000,
              "namespace": flowNamespace,
              "tags": [flowTag, "member"],
              "permissions": [
                {
                  "toolId": "2e927a929a85408bbb61b235e31c0745",
                  "functions": [
                    {
                      "funcName": "githubCreateRepository",
                      "enabled": false
                    }
                  ],
                  "resources": [
                  ]
                }
              ]
            }
          ],
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
      rememberCreatedTokenIdsFromResponse(data);

      // Verify changes after create: list + get
      const listRes = await fetch(`${BASE_URL}/api/external/tokens`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({ namespace: flowNamespace, tags: [flowTag] }),
      });
      const listData = await listRes.json();
      logEndpointResult('tokens', listRes.status, listData, '(after create)');

      const createdTokenId = getTokenIdForMutation(false);
      if (createdTokenId) {
        const getRes = await fetch(`${BASE_URL}/api/external/tokens/get`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            Authorization: `Bearer ${OWNER_TOKEN}`,
          },
          body: JSON.stringify({ tokenId: createdTokenId }),
        });
        const getData = await getRes.json();
        logEndpointResult('tokens/get', getRes.status, getData, '(after create)');
      }
    },
  },
]);

// ============================================================
// POST /api/external/tokens/update
// ============================================================

suite('tokens/update', [
  {
    name: 'missing Authorization should return E2001',
    run: async () => {
      const res = await fetch(`${BASE_URL}/api/external/tokens/update`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ tokenId: 'test-token-id', name: 'New Name' }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'missing tokenId should return E1001',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/tokens/update`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({ name: 'New Name' }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));

    },
  },
  {
    name: 'non-existent tokenId should return E3003',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/tokens/update`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({ tokenId: 'non-existent-token-id', name: 'New Name' }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));

    },
  },
  {
    name: 'valid update (name and notes) should return 200',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      let targetTokenId = getTokenIdForMutation(false);

      if (!targetTokenId) {
        const tokensRes = await fetch(`${BASE_URL}/api/external/tokens`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            Authorization: `Bearer ${OWNER_TOKEN}`,
          },
          body: JSON.stringify({}),
        });
        const tokensData = await tokensRes.json();
        const nonOwnerToken = tokensData?.data?.tokens?.find((t: any) => t.role !== 1);
        targetTokenId = nonOwnerToken?.tokenId || null;
      }

      if (!targetTokenId) {
        console.log('    Skipped: No token available for update');
        return;
      }

      const res = await fetch(`${BASE_URL}/api/external/tokens/update`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({
          tokenId: targetTokenId,
          name: 'Developer Token 01 admin update',
          notes: 'Updated notes via API test',
          permissions: [
            {
              "toolId": "2e927a929a85408bbb61b235e31c0745",
              "functions": [
                {
                  "funcName": "githubCreateRepository",
                  "enabled": false
                }
              ],
              "resources": [
              ]
            }
          ]
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));

      // Verify changes after update: list + get
      const listRes = await fetch(`${BASE_URL}/api/external/tokens`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({}),
      });
      const listData = await listRes.json();
      logEndpointResult('tokens', listRes.status, listData, '(after update)');

      const getRes = await fetch(`${BASE_URL}/api/external/tokens/get`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({ tokenId: targetTokenId }),
      });
      const getData = await getRes.json();
      logEndpointResult('tokens/get', getRes.status, getData, '(after update)');
    },
  },
  {
    name: 'valid update with namespace should return 200',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      // First get a valid tokenId
      const tokensRes = await fetch(`${BASE_URL}/api/external/tokens`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({}),
      });
      const tokensData = await tokensRes.json();
      if (!tokensData.success || !tokensData.data?.tokens?.length) {
        console.log('    Skipped: No tokens found');
        return;
      }
      const nonOwnerToken = tokensData.data.tokens.find((t: any) => t.role !== 1);
      if (!nonOwnerToken) {
        console.log('    Skipped: No non-owner token found');
        return;
      }

      const res = await fetch(`${BASE_URL}/api/external/tokens/update`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({
          tokenId: nonOwnerToken.tokenId,
          namespace: 'test-namespace',
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'valid update with tags should return 200',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      // First get a valid tokenId
      const tokensRes = await fetch(`${BASE_URL}/api/external/tokens`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({}),
      });
      const tokensData = await tokensRes.json();
      if (!tokensData.success || !tokensData.data?.tokens?.length) {
        console.log('    Skipped: No tokens found');
        return;
      }
      const nonOwnerToken = tokensData.data.tokens.find((t: any) => t.role !== 1);
      if (!nonOwnerToken) {
        console.log('    Skipped: No non-owner token found');
        return;
      }

      const res = await fetch(`${BASE_URL}/api/external/tokens/update`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({
          tokenId: nonOwnerToken.tokenId,
          tags: ['tag1', 'tag2'],
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'valid update with namespace and tags should return 200',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      // First get a valid tokenId
      const tokensRes = await fetch(`${BASE_URL}/api/external/tokens`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({}),
      });
      const tokensData = await tokensRes.json();
      if (!tokensData.success || !tokensData.data?.tokens?.length) {
        console.log('    Skipped: No tokens found');
        return;
      }
      const nonOwnerToken = tokensData.data.tokens.find((t: any) => t.role !== 1);
      if (!nonOwnerToken) {
        console.log('    Skipped: No non-owner token found');
        return;
      }

      const res = await fetch(`${BASE_URL}/api/external/tokens/update`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({
          tokenId: nonOwnerToken.tokenId,
          namespace: 'updated-namespace',
          tags: ['updated-tag1', 'updated-tag2'],
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
]);

// ============================================================
// POST /api/external/tokens/delete
// ============================================================

suite('tokens/delete', [
  {
    name: 'missing Authorization should return E2001',
    run: async () => {
      const res = await fetch(`${BASE_URL}/api/external/tokens/delete`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ tokenId: 'test-token-id' }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'missing tokenId should return E1001',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/tokens/delete`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({}),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'non-existent tokenId should return E3003',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/tokens/delete`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({ tokenId: 'non-existent-token-id' }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'valid tokenId should return 200',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      let tokenIdForDelete = getTokenIdForMutation(true);

      if (!tokenIdForDelete) {
        const tokensRes = await fetch(`${BASE_URL}/api/external/tokens`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            Authorization: `Bearer ${OWNER_TOKEN}`,
          },
          body: JSON.stringify({}),
        });
        const tokensData = await tokensRes.json();
        const nonOwnerToken = tokensData?.data?.tokens?.find((t: any) => t.role !== 1);
        tokenIdForDelete = nonOwnerToken?.tokenId || null;
      }

      if (!tokenIdForDelete) {
        console.log('    Skipped: No token available for delete');
        return;
      }

      const res = await fetch(`${BASE_URL}/api/external/tokens/delete`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({ tokenId: tokenIdForDelete }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));

      // Verify changes after delete: list + get
      const listRes = await fetch(`${BASE_URL}/api/external/tokens`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({}),
      });
      const listData = await listRes.json();
      logEndpointResult('tokens', listRes.status, listData, '(after delete)');

      const getRes = await fetch(`${BASE_URL}/api/external/tokens/get`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({ tokenId: tokenIdForDelete }),
      });
      const getData = await getRes.json();
      logEndpointResult('tokens/get', getRes.status, getData, '(after delete)');
    },
  },
]);

// ============================================================
// POST /api/external/tokens/batch-update
// ============================================================

suite('tokens/batch-update', [
  {
    name: 'missing Authorization should return E2001',
    run: async () => {
      const res = await fetch(`${BASE_URL}/api/external/tokens/batch-update`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ tokenIds: ['test-token-id'], namespace: 'test' }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'missing tokenIds should return E1001',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/tokens/batch-update`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({ namespace: 'test' }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'empty tokenIds array should return E1001',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/tokens/batch-update`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({ tokenIds: [], namespace: 'test' }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'invalid permissionsMode should return E1003',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/tokens/batch-update`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({
          tokenIds: ['test-token-id'],
          permissions: [],
          permissionsMode: 'invalid',
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'invalid tagsMode should return E1003',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/tokens/batch-update`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({
          tokenIds: ['test-token-id'],
          tags: ['tag1'],
          tagsMode: 'invalid',
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'tagsMode add without tags should return E1001',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/tokens/batch-update`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({
          tokenIds: ['test-token-id'],
          tagsMode: 'add',
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'tagsMode clear without tags should return 200',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      // First get valid tokenIds
      const tokensRes = await fetch(`${BASE_URL}/api/external/tokens`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({}),
      });
      const tokensData = await tokensRes.json();
      if (!tokensData.success || !tokensData.data?.tokens?.length) {
        console.log('    Skipped: No tokens found');
        return;
      }
      const nonOwnerTokens = tokensData.data.tokens.filter((t: any) => t.role !== 1);
      if (nonOwnerTokens.length === 0) {
        console.log('    Skipped: No non-owner tokens found');
        return;
      }

      const res = await fetch(`${BASE_URL}/api/external/tokens/batch-update`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({
          tokenIds: [nonOwnerTokens[0].tokenId],
          tagsMode: 'clear',
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'missing required fields (no permissions, namespace, or tags) should return E1001',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/tokens/batch-update`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({
          tokenIds: ['test-token-id'],
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'valid batch update with namespace should return 200',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      // First get valid tokenIds
      const tokensRes = await fetch(`${BASE_URL}/api/external/tokens`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({}),
      });
      const tokensData = await tokensRes.json();
      if (!tokensData.success || !tokensData.data?.tokens?.length) {
        console.log('    Skipped: No tokens found');
        return;
      }
      const nonOwnerTokens = tokensData.data.tokens.filter((t: any) => t.role !== 1);
      if (nonOwnerTokens.length === 0) {
        console.log('    Skipped: No non-owner tokens found');
        return;
      }

      const res = await fetch(`${BASE_URL}/api/external/tokens/batch-update`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({
          tokenIds: nonOwnerTokens.slice(0, 2).map((t: any) => t.tokenId),
          namespace: 'batch-test-namespace',
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'valid batch update with tags (replace mode) should return 200',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      // First get valid tokenIds
      const tokensRes = await fetch(`${BASE_URL}/api/external/tokens`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({}),
      });
      const tokensData = await tokensRes.json();
      if (!tokensData.success || !tokensData.data?.tokens?.length) {
        console.log('    Skipped: No tokens found');
        return;
      }
      const nonOwnerTokens = tokensData.data.tokens.filter((t: any) => t.role !== 1);
      if (nonOwnerTokens.length === 0) {
        console.log('    Skipped: No non-owner tokens found');
        return;
      }

      const res = await fetch(`${BASE_URL}/api/external/tokens/batch-update`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({
          tokenIds: nonOwnerTokens.slice(0, 2).map((t: any) => t.tokenId),
          tags: ['batch-tag1', 'batch-tag2'],
          tagsMode: 'replace',
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'valid batch update with tags (add mode) should return 200',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      // First get valid tokenIds
      const tokensRes = await fetch(`${BASE_URL}/api/external/tokens`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({}),
      });
      const tokensData = await tokensRes.json();
      if (!tokensData.success || !tokensData.data?.tokens?.length) {
        console.log('    Skipped: No tokens found');
        return;
      }
      const nonOwnerTokens = tokensData.data.tokens.filter((t: any) => t.role !== 1);
      if (nonOwnerTokens.length === 0) {
        console.log('    Skipped: No non-owner tokens found');
        return;
      }

      const res = await fetch(`${BASE_URL}/api/external/tokens/batch-update`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({
          tokenIds: nonOwnerTokens.slice(0, 2).map((t: any) => t.tokenId),
          tags: ['added-tag'],
          tagsMode: 'add',
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'valid batch update with tags (remove mode) should return 200',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      // First get valid tokenIds
      const tokensRes = await fetch(`${BASE_URL}/api/external/tokens`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({}),
      });
      const tokensData = await tokensRes.json();
      if (!tokensData.success || !tokensData.data?.tokens?.length) {
        console.log('    Skipped: No tokens found');
        return;
      }
      const nonOwnerTokens = tokensData.data.tokens.filter((t: any) => t.role !== 1);
      if (nonOwnerTokens.length === 0) {
        console.log('    Skipped: No non-owner tokens found');
        return;
      }

      const res = await fetch(`${BASE_URL}/api/external/tokens/batch-update`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({
          tokenIds: nonOwnerTokens.slice(0, 2).map((t: any) => t.tokenId),
          tags: ['added-tag'],
          tagsMode: 'remove',
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'valid batch update with permissions (replace mode) should return 200',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      // First get valid tokenIds
      const tokensRes = await fetch(`${BASE_URL}/api/external/tokens`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({}),
      });
      const tokensData = await tokensRes.json();
      if (!tokensData.success || !tokensData.data?.tokens?.length) {
        console.log('    Skipped: No tokens found');
        return;
      }
      const nonOwnerTokens = tokensData.data.tokens.filter((t: any) => t.role !== 1);
      if (nonOwnerTokens.length === 0) {
        console.log('    Skipped: No non-owner tokens found');
        return;
      }

      const res = await fetch(`${BASE_URL}/api/external/tokens/batch-update`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({
          tokenIds: nonOwnerTokens.slice(0, 2).map((t: any) => t.tokenId),
          permissions: [
            {
              toolId: '2e927a929a85408bbb61b235e31c0745',
              functions: [
                { funcName: 'testFunction', enabled: true }
              ],
            }
          ],
          permissionsMode: 'replace',
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'valid batch update with permissions (merge mode) should return 200',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      // First get valid tokenIds
      const tokensRes = await fetch(`${BASE_URL}/api/external/tokens`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({}),
      });
      const tokensData = await tokensRes.json();
      if (!tokensData.success || !tokensData.data?.tokens?.length) {
        console.log('    Skipped: No tokens found');
        return;
      }
      const nonOwnerTokens = tokensData.data.tokens.filter((t: any) => t.role !== 1);
      if (nonOwnerTokens.length === 0) {
        console.log('    Skipped: No non-owner tokens found');
        return;
      }

      const res = await fetch(`${BASE_URL}/api/external/tokens/batch-update`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({
          tokenIds: nonOwnerTokens.slice(0, 2).map((t: any) => t.tokenId),
          permissions: [
            {
              toolId: '2e927a929a85408bbb61b235e31c0745',
              functions: [
                { funcName: 'mergedFunction', enabled: true }
              ],
            }
          ],
          permissionsMode: 'merge',
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'batch update with non-existent tokenId should return partial success',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      // First get valid tokenIds
      const tokensRes = await fetch(`${BASE_URL}/api/external/tokens`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({}),
      });
      const tokensData = await tokensRes.json();
      if (!tokensData.success || !tokensData.data?.tokens?.length) {
        console.log('    Skipped: No tokens found');
        return;
      }
      const nonOwnerTokens = tokensData.data.tokens.filter((t: any) => t.role !== 1);
      if (nonOwnerTokens.length === 0) {
        console.log('    Skipped: No non-owner tokens found');
        return;
      }

      const res = await fetch(`${BASE_URL}/api/external/tokens/batch-update`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({
          tokenIds: [nonOwnerTokens[0].tokenId, 'non-existent-token-id'],
          namespace: 'partial-success-test',
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'batch update with namespace and tags should return 200',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      // First get valid tokenIds
      const tokensRes = await fetch(`${BASE_URL}/api/external/tokens`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({}),
      });
      const tokensData = await tokensRes.json();
      if (!tokensData.success || !tokensData.data?.tokens?.length) {
        console.log('    Skipped: No tokens found');
        return;
      }
      const nonOwnerTokens = tokensData.data.tokens.filter((t: any) => t.role !== 1);
      if (nonOwnerTokens.length === 0) {
        console.log('    Skipped: No non-owner tokens found');
        return;
      }

      const res = await fetch(`${BASE_URL}/api/external/tokens/batch-update`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({
          tokenIds: nonOwnerTokens.slice(0, 2).map((t: any) => t.tokenId),
          namespace: 'combined-namespace',
          tags: ['combined-tag1', 'combined-tag2'],
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
]);

// ============================================================
// POST /api/external/tools
// ============================================================

suite('tools', [
  {
    name: 'missing Authorization should return E2001',
    run: async () => {
      const res = await fetch(`${BASE_URL}/api/external/tools`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({}),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'valid request should return 200 with tools list',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/tools`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({}),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'filter enabled=true should return 200',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/tools`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({ enabled: true }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
]);

// ============================================================
// POST /api/external/tools/delete
// ============================================================

suite('tools/delete', [
  {
    name: 'missing Authorization should return E2001',
    run: async () => {
      const res = await fetch(`${BASE_URL}/api/external/tools/delete`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ toolId: 'test-tool-id' }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'missing toolId should return E1001',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/tools/delete`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({}),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'non-existent toolId should return E3002',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/tools/delete`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({ toolId: 'non-existent-tool-id' }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'valid toolId should return 200',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      // Note: This test assumes a valid tool exists. Replace with actual toolId.
      const res = await fetch(`${BASE_URL}/api/external/tools/delete`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({ toolId: 'test-tool-id' }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
]);

// ============================================================
// POST /api/external/tools/enable
// ============================================================

suite('tools/enable', [
  {
    name: 'missing Authorization should return E2001',
    run: async () => {
      const res = await fetch(`${BASE_URL}/api/external/tools/enable`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ toolId: 'test-tool-id' }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'missing toolId should return E1001',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/tools/enable`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({}),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'non-existent toolId should return E3002',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/tools/enable`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({ toolId: 'non-existent-tool-id' }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'valid toolId should return 200',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      // Note: This test assumes a valid tool exists. Replace with actual toolId.
      const res = await fetch(`${BASE_URL}/api/external/tools/enable`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({ toolId: 'test-tool-id' }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
]);

// ============================================================
// POST /api/external/tools/disable
// ============================================================

suite('tools/disable', [
  {
    name: 'missing Authorization should return E2001',
    run: async () => {
      const res = await fetch(`${BASE_URL}/api/external/tools/disable`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ toolId: 'test-tool-id' }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'missing toolId should return E1001',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/tools/disable`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({}),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'non-existent toolId should return E3002',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/tools/disable`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({ toolId: 'non-existent-tool-id' }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'valid toolId should return 200',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      // Note: This test assumes a valid tool exists. Replace with actual toolId.
      const res = await fetch(`${BASE_URL}/api/external/tools/disable`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({ toolId: 'test-tool-id' }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
]);

// ============================================================
// POST /api/external/tools/start
// ============================================================

suite('tools/start', [
  {
    name: 'missing Authorization should return E2001',
    run: async () => {
      const res = await fetch(`${BASE_URL}/api/external/tools/start`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ toolId: 'test-tool-id' }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'missing toolId should return E1001',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/tools/start`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({}),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'non-existent toolId should return E3002',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/tools/start`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({ toolId: 'non-existent-tool-id' }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'valid toolId should return 200',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      // Note: This test assumes a valid tool exists. Replace with actual toolId.
      const res = await fetch(`${BASE_URL}/api/external/tools/start`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({ toolId: 'test-tool-id' }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
]);

// ============================================================
// POST /api/external/tools/connect-all
// ============================================================

suite('tools/connect-all', [
  {
    name: 'missing Authorization should return E2001',
    run: async () => {
      const res = await fetch(`${BASE_URL}/api/external/tools/connect-all`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({}),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'valid request should return 200 with results',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/tools/connect-all`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({}),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
]);

// ============================================================
// POST /api/external/templates
// ============================================================

suite('templates', [
  {
    name: 'missing Authorization should return E2001',
    run: async () => {
      const res = await fetch(`${BASE_URL}/api/external/templates`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({}),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'valid request should return 200 with templates list',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/templates`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({}),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
]);

// ============================================================
// POST /api/external/tools/get
// ============================================================

suite('tools/get', [
  {
    name: 'missing Authorization should return E2001',
    run: async () => {
      const res = await fetch(`${BASE_URL}/api/external/tools/get`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ toolId: 'test-tool-id' }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'missing toolId should return E1001',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/tools/get`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({}),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'non-existent toolId should return E3002',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/tools/get`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({ toolId: 'non-existent-tool-id' }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'valid toolId should return 200 with tool details',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/tools/get`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({ toolId: '2e927a929a85408bbb61b235e31c0745' }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
]);

// ============================================================
// POST /api/external/tools/basic-mcp
// ============================================================

suite('tools/basic-mcp', [
  {
    name: 'missing Authorization should return E2001',
    run: async () => {
      const res = await fetch(`${BASE_URL}/api/external/tools/basic-mcp`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          toolTmplId: 'github-mcp',
          mcpJsonConf: { command: 'npx', args: ['-y', '@anthropic/github-mcp'] },
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'missing toolTmplId should return E1001',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/tools/basic-mcp`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({
          mcpJsonConf: { command: 'npx', args: ['-y', '@anthropic/github-mcp'] },
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'missing mcpJsonConf should return E1001',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/tools/basic-mcp`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({
          toolTmplId: 'github-mcp',
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'non-existent toolTmplId should return E3004',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/tools/basic-mcp`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({
          toolTmplId: 'non-existent-template',
          mcpJsonConf: { command: 'npx', args: ['-y', '@example/mcp'] },
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'valid request should return 201 with toolId and isStarted',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/tools/basic-mcp`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({
          toolTmplId: 'github-mcp',
          mcpJsonConf: {
            command: 'npx',
            args: ['-y', '@anthropic/github-mcp'],
            env: {
              GITHUB_TOKEN: 'ghp_test_token_12345',
            },
          },
          lazyStartEnabled: false,
          publicAccess: false,
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
]);

// ============================================================
// POST /api/external/tools/basic-mcp/update
// ============================================================

suite('tools/basic-mcp/update', [
  {
    name: 'missing Authorization should return E2001',
    run: async () => {
      const res = await fetch(`${BASE_URL}/api/external/tools/basic-mcp/update`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          toolId: 'test-tool-id',
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'missing toolId should return E1001',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/tools/basic-mcp/update`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({}),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'non-existent toolId should return E3002',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/tools/basic-mcp/update`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({
          toolId: 'nonexistent-tool-id-12345',
          lazyStartEnabled: true,
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'valid update (lazyStartEnabled and publicAccess) should return 200',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      // Note: This test assumes a valid basic-mcp tool exists. Replace with actual toolId.
      const res = await fetch(`${BASE_URL}/api/external/tools/basic-mcp/update`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({
          toolId: 'test-basic-mcp-tool-id',
          lazyStartEnabled: true,
          publicAccess: false,
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'update with functions and resources should return 200',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      // Note: This test assumes a valid basic-mcp tool exists. Replace with actual toolId.
      const res = await fetch(`${BASE_URL}/api/external/tools/basic-mcp/update`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({
          toolId: 'test-basic-mcp-tool-id',
          functions: [
            {
              funcName: 'create_issue',
              enabled: true,
              dangerLevel: 1,
              description: 'Create a GitHub issue',
            },
          ],
          resources: [
            {
              uri: 'repo://owner/repo',
              enabled: true,
            },
          ],
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'update with mcpJsonConf should return 200',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      // Note: This test assumes a valid basic-mcp tool exists with authType=ApiKey and allowUserInput=false.
      const res = await fetch(`${BASE_URL}/api/external/tools/basic-mcp/update`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({
          toolId: 'test-basic-mcp-tool-id',
          mcpJsonConf: {
            command: 'npx',
            args: ['-y', '@anthropic/github-mcp'],
            env: {
              GITHUB_TOKEN: 'ghp_updated_token_12345',
            },
          },
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
]);

// ============================================================
// POST /api/external/tools/custom-mcp
// ============================================================

suite('tools/custom-mcp', [
  {
    name: 'missing Authorization should return E2001',
    run: async () => {
      const res = await fetch(`${BASE_URL}/api/external/tools/custom-mcp`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          customRemoteConfig: {
            url: 'https://my-mcp-server.com/sse',
          },
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'missing customRemoteConfig should return E1001',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/tools/custom-mcp`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({}),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'missing customRemoteConfig.url should return E1001',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/tools/custom-mcp`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({
          customRemoteConfig: {},
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'invalid url format should return E1005',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/tools/custom-mcp`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({
          customRemoteConfig: {
            url: 'not-a-valid-url',
          },
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'valid request should return 201 with toolId and isStarted',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/tools/custom-mcp`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({
          customRemoteConfig: {
            url: 'https://my-mcp-server.example.com/sse',
            headers: {
              'Authorization': 'Bearer test-token-12345',
            },
          },
          lazyStartEnabled: true,
          publicAccess: false,
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
]);

// ============================================================
// POST /api/external/tools/custom-mcp/update
// ============================================================

suite('tools/custom-mcp/update', [
  {
    name: 'missing Authorization should return E2001',
    run: async () => {
      const res = await fetch(`${BASE_URL}/api/external/tools/custom-mcp/update`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          toolId: 'test-tool-id',
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'missing toolId should return E1001',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/tools/custom-mcp/update`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({}),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'non-existent toolId should return E3002',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/tools/custom-mcp/update`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({
          toolId: 'nonexistent-tool-id-12345',
          lazyStartEnabled: true,
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'update with empty customRemoteConfig.url should return E1003',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      // Note: This test assumes a valid custom-mcp tool exists. Replace with actual toolId.
      const res = await fetch(`${BASE_URL}/api/external/tools/custom-mcp/update`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({
          toolId: 'test-custom-mcp-tool-id',
          customRemoteConfig: {
            url: '',
          },
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'valid update (lazyStartEnabled) should return 200',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      // Note: This test assumes a valid custom-mcp tool exists. Replace with actual toolId.
      const res = await fetch(`${BASE_URL}/api/external/tools/custom-mcp/update`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({
          toolId: 'test-custom-mcp-tool-id',
          lazyStartEnabled: true,
          publicAccess: false,
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
]);

// ============================================================
// POST /api/external/tools/rest-api
// ============================================================

suite('tools/rest-api', [
  {
    name: 'missing Authorization should return E2001',
    run: async () => {
      const res = await fetch(`${BASE_URL}/api/external/tools/rest-api`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          restApiConfig: {
            baseUrl: 'https://api.example.com',
            tools: [{ name: 'getUsers', method: 'GET', path: '/users' }],
          },
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'missing restApiConfig should return E1001',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/tools/rest-api`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({}),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'missing restApiConfig.baseUrl should return E1003',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/tools/rest-api`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({
          restApiConfig: {
            tools: [{ name: 'getUsers', method: 'GET', path: '/users' }],
          },
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'empty restApiConfig.tools should return E1003',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/tools/rest-api`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({
          restApiConfig: {
            baseUrl: 'https://api.example.com',
            tools: [],
          },
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'valid request should return 201 with toolId and isStarted',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/tools/rest-api`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({
          restApiConfig: {
            name: 'Test REST API',
            baseUrl: 'https://api.example.com',
            auth: {
              type: 'bearer',
              token: 'test-token-12345',
            },
            tools: [
              {
                name: 'getUsers',
                description: 'Get all users',
                method: 'GET',
                path: '/users',
              },
              {
                name: 'createUser',
                description: 'Create a new user',
                method: 'POST',
                path: '/users',
                body: {
                  type: 'json',
                  schema: {
                    name: 'string',
                    email: 'string',
                  },
                },
              },
            ],
          },
          lazyStartEnabled: true,
          publicAccess: false,
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
]);

// ============================================================
// POST /api/external/tools/rest-api/update
// ============================================================

suite('tools/rest-api/update', [
  {
    name: 'missing Authorization should return E2001',
    run: async () => {
      const res = await fetch(`${BASE_URL}/api/external/tools/rest-api/update`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          toolId: 'test-tool-id',
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'missing toolId should return E1001',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/tools/rest-api/update`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({}),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'non-existent toolId should return E3002',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/tools/rest-api/update`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({
          toolId: 'nonexistent-tool-id-12345',
          lazyStartEnabled: true,
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'update with empty tools array should return E1003',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      // Note: This test assumes a valid rest-api tool exists. Replace with actual toolId.
      const res = await fetch(`${BASE_URL}/api/external/tools/rest-api/update`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({
          toolId: 'test-rest-api-tool-id',
          restApiConfig: {
            baseUrl: 'https://api.example.com',
            tools: [],
          },
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'valid update (lazyStartEnabled) should return 200',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      // Note: This test assumes a valid rest-api tool exists. Replace with actual toolId.
      const res = await fetch(`${BASE_URL}/api/external/tools/rest-api/update`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({
          toolId: 'test-rest-api-tool-id',
          lazyStartEnabled: true,
          publicAccess: false,
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'valid update with restApiConfig should return 200',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      // Note: This test assumes a valid rest-api tool exists. Replace with actual toolId.
      const res = await fetch(`${BASE_URL}/api/external/tools/rest-api/update`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({
          toolId: 'test-rest-api-tool-id',
          restApiConfig: {
            name: 'Updated REST API',
            baseUrl: 'https://api.example.com',
            auth: {
              type: 'bearer',
              token: 'new-token-67890',
            },
            tools: [
              {
                name: 'getUsers',
                description: 'Get all users',
                method: 'GET',
                path: '/users',
              },
            ],
          },
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
]);

// ============================================================
// POST /api/external/tools/skills
// ============================================================

suite('tools/skills', [
  {
    name: 'missing Authorization should return E2001',
    run: async () => {
      const res = await fetch(`${BASE_URL}/api/external/tools/skills`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          serverName: 'My Skills Server',
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'empty body should return 201 with default serverName',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/tools/skills`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({}),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'valid request with serverName should return 201',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/tools/skills`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({
          serverName: 'My Custom Skills Server',
          lazyStartEnabled: true,
          publicAccess: false,
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
]);

// ============================================================
// POST /api/external/tools/skills/update
// ============================================================

suite('tools/skills/update', [
  {
    name: 'missing Authorization should return E2001',
    run: async () => {
      const res = await fetch(`${BASE_URL}/api/external/tools/skills/update`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          toolId: 'test-tool-id',
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'missing toolId should return E1001',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/tools/skills/update`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({}),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'non-existent toolId should return E3002',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/tools/skills/update`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({
          toolId: 'nonexistent-tool-id-12345',
          lazyStartEnabled: true,
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'valid update (lazyStartEnabled and publicAccess) should return 200',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      // Note: This test assumes a valid skills tool exists. Replace with actual toolId.
      const res = await fetch(`${BASE_URL}/api/external/tools/skills/update`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({
          toolId: 'test-skills-tool-id',
          lazyStartEnabled: true,
          publicAccess: false,
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'valid update with functions and resources should return 200',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      // Note: This test assumes a valid skills tool exists. Replace with actual toolId.
      const res = await fetch(`${BASE_URL}/api/external/tools/skills/update`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({
          toolId: 'test-skills-tool-id',
          functions: [
            {
              funcName: 'my_skill',
              enabled: true,
              dangerLevel: 0,
              description: 'My custom skill',
            },
          ],
          resources: [
            {
              uri: 'skill://my-resource',
              enabled: true,
            },
          ],
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
]);

// ============================================================
// POST /api/external/ip-whitelist
// ============================================================

suite('ip-whitelist', [
  {
    name: 'missing Authorization should return E2001',
    run: async () => {
      const res = await fetch(`${BASE_URL}/api/external/ip-whitelist`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({}),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'valid request should return 200 with allowAll and ipList',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/ip-whitelist`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({}),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
]);

// ============================================================
// POST /api/external/ip-whitelist/allow-all
// ============================================================

suite('ip-whitelist/allow-all', [
  {
    name: 'missing Authorization should return E2001',
    run: async () => {
      const res = await fetch(`${BASE_URL}/api/external/ip-whitelist/allow-all`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({}),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'valid request should return 200 with success message',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/ip-whitelist/allow-all`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({}),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
]);

// ============================================================
// POST /api/external/ip-whitelist/restrict
// ============================================================

suite('ip-whitelist/restrict', [
  {
    name: 'missing Authorization should return E2001',
    run: async () => {
      const res = await fetch(`${BASE_URL}/api/external/ip-whitelist/restrict`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({}),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'valid request should return 200 with success message',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/ip-whitelist/restrict`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({}),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
]);

// ============================================================
// POST /api/external/ip-whitelist/add
// ============================================================

suite('ip-whitelist/add', [
  {
    name: 'missing Authorization should return E2001',
    run: async () => {
      const res = await fetch(`${BASE_URL}/api/external/ip-whitelist/add`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          ipList: ['192.168.1.0/24'],
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'missing ipList should return E1001',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/ip-whitelist/add`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({}),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'empty ipList should return E1001',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/ip-whitelist/add`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({
          ipList: [],
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'valid request should return 201 with added count',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/ip-whitelist/add`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({
          ipList: ['192.168.1.0/24', '10.0.0.1'],
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
]);

// ============================================================
// POST /api/external/ip-whitelist/delete
// ============================================================

suite('ip-whitelist/delete', [
  {
    name: 'missing Authorization should return E2001',
    run: async () => {
      const res = await fetch(`${BASE_URL}/api/external/ip-whitelist/delete`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          ipList: ['192.168.1.0/24'],
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'missing ipList should return E1001',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/ip-whitelist/delete`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({}),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'empty ipList should return E1001',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/ip-whitelist/delete`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({
          ipList: [],
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'valid request should return 200 with deleted count',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/ip-whitelist/delete`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({
          ipList: ['192.168.1.0/24', '10.0.0.1'],
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
]);

// ============================================================
// POST /api/external/kimbap-core/connect
// ============================================================

suite('kimbap-core/connect', [
  {
    name: 'missing Authorization should return E2001',
    run: async () => {
      const res = await fetch(`${BASE_URL}/api/external/kimbap-core/connect`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          host: 'kimbap-core.example.com',
          port: 443,
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'missing host should return E1001',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/kimbap-core/connect`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({
          port: 443,
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'invalid port should return E1003',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/kimbap-core/connect`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({
          host: 'kimbap-core.example.com',
          port: 99999,
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'unreachable host should return E4014',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/kimbap-core/connect`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({
          host: 'nonexistent.invalid.host',
          port: 443,
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'valid KIMBAP Core should return 200 with isValid=1',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      // Note: Replace with actual KIMBAP Core host for testing
      const res = await fetch(`${BASE_URL}/api/external/kimbap-core/connect`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({
          host: 'https://localhost',
          port: 3002,
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
]);

// ============================================================
// POST /api/external/user-capabilities
// ============================================================

suite('user-capabilities', [
  {
    name: 'missing Authorization should return E2001',
    run: async () => {
      const res = await fetch(`${BASE_URL}/api/external/user-capabilities`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ tokenId: 'test-token-id' }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'missing tokenId should return E1001',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/user-capabilities`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({}),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'empty tokenId should return E1001',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/user-capabilities`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({ tokenId: '' }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'non-existent tokenId should return E3003',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/user-capabilities`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({ tokenId: 'non-existent-token-id' }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'valid tokenId should return 200 with capabilities',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      // First get a valid tokenId from the tokens list
      const tokensRes = await fetch(`${BASE_URL}/api/external/tokens`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({}),
      });
      const tokensData = await tokensRes.json();
      if (!tokensData.success || !tokensData.data?.tokens?.length) {
        console.log('    Skipped: No tokens found');
        return;
      }
      const validTokenId = tokensData.data.tokens[0].tokenId;
      const res = await fetch(`${BASE_URL}/api/external/user-capabilities`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({ tokenId: validTokenId }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
]);

// ============================================================
// POST /api/external/user-capabilities/set
// ============================================================

suite('user-capabilities/set', [
  {
    name: 'missing Authorization should return E2001',
    run: async () => {
      const res = await fetch(`${BASE_URL}/api/external/user-capabilities/set`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ tokenId: 'test-token-id', capabilities: [] }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'missing tokenId should return E1001',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/user-capabilities/set`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({ capabilities: [] }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'missing capabilities should return E1001',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/user-capabilities/set`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({ tokenId: 'test-token-id' }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'non-existent tokenId should return E3003',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/user-capabilities/set`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({ tokenId: 'non-existent-token-id', capabilities: [] }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'valid request with empty capabilities should return 200',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      // First get a valid tokenId from the tokens list
      const tokensRes = await fetch(`${BASE_URL}/api/external/tokens`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({}),
      });
      const tokensData = await tokensRes.json();
      if (!tokensData.success || !tokensData.data?.tokens?.length) {
        console.log('    Skipped: No tokens found');
        return;
      }
      const validTokenId = tokensData.data.tokens[0].tokenId;
      const res = await fetch(`${BASE_URL}/api/external/user-capabilities/set`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({ tokenId: validTokenId, capabilities: [] }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'valid request with capabilities should return 200',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      // First get a valid tokenId from the tokens list
      const tokensRes = await fetch(`${BASE_URL}/api/external/tokens`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({}),
      });
      const tokensData = await tokensRes.json();
      if (!tokensData.success || !tokensData.data?.tokens?.length) {
        console.log('    Skipped: No tokens found');
        return;
      }
      const validTokenId = tokensData.data.tokens[0].tokenId;

      // Get current capabilities first
      const capRes = await fetch(`${BASE_URL}/api/external/user-capabilities`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({ tokenId: validTokenId }),
      });
      const capData = await capRes.json();
      if (!capData.success || !capData.data?.capabilities?.length) {
        console.log('    Skipped: No capabilities found for user');
        return;
      }

      // Use first capability to test setting
      const firstCap = capData.data.capabilities[0];
      const res = await fetch(`${BASE_URL}/api/external/user-capabilities/set`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({
          tokenId: validTokenId,
          capabilities: [
            {
              toolId: firstCap.toolId,
              enabled: firstCap.enabled,
              functions: firstCap.functions?.slice(0, 1) || [],
            },
          ],
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
]);

// ============================================================
// POST /api/external/user-servers/configure
// ============================================================

suite('user-servers/configure', [
  {
    name: 'missing Authorization should return E2001',
    run: async () => {
      const res = await fetch(`${BASE_URL}/api/external/user-servers/configure`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          tokenId: 'test-token-id',
          toolId: 'test-tool-id',
          authConf: [{ key: 'API_KEY', value: 'test', dataType: 1 }],
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'missing tokenId should return E1001',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/user-servers/configure`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({
          toolId: 'test-tool-id',
          authConf: [{ key: 'API_KEY', value: 'test', dataType: 1 }],
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'missing toolId should return E1001',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/user-servers/configure`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({
          tokenId: 'test-token-id',
          authConf: [{ key: 'API_KEY', value: 'test', dataType: 1 }],
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'missing auth parameter should return E1001',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/user-servers/configure`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({
          tokenId: 'test-token-id',
          toolId: 'test-tool-id',
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'multiple auth parameters should return E1003',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/user-servers/configure`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({
          tokenId: 'test-token-id',
          toolId: 'test-tool-id',
          authConf: [{ key: 'API_KEY', value: 'test', dataType: 1 }],
          remoteAuth: { params: { key: 'value' } },
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'invalid authConf (empty array) should return E1003',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/user-servers/configure`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({
          tokenId: 'test-token-id',
          toolId: 'test-tool-id',
          authConf: [],
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'invalid remoteAuth (empty params and headers) should return E1003',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/user-servers/configure`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({
          tokenId: 'test-token-id',
          toolId: 'test-tool-id',
          remoteAuth: { params: {}, headers: {} },
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'invalid restfulApiAuth (missing type) should return E1003',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/user-servers/configure`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({
          tokenId: 'test-token-id',
          toolId: 'test-tool-id',
          restfulApiAuth: { value: 'test-token' },
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'invalid restfulApiAuth bearer (missing value) should return E1003',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/user-servers/configure`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({
          tokenId: 'test-token-id',
          toolId: 'test-tool-id',
          restfulApiAuth: { type: 'bearer' },
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'invalid restfulApiAuth basic (missing password) should return E1003',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/user-servers/configure`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({
          tokenId: 'test-token-id',
          toolId: 'test-tool-id',
          restfulApiAuth: { type: 'basic', username: 'user' },
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'non-existent tokenId should return E3003',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/user-servers/configure`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({
          tokenId: 'non-existent-token-id',
          toolId: 'test-tool-id',
          authConf: [{ key: 'API_KEY', value: 'test', dataType: 1 }],
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'valid request with authConf (Template server) should return 200 or Core error',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      // First get a valid tokenId from the tokens list
      const tokensRes = await fetch(`${BASE_URL}/api/external/tokens`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({}),
      });
      const tokensData = await tokensRes.json();
      if (!tokensData.success || !tokensData.data?.tokens?.length) {
        console.log('    Skipped: No tokens found');
        return;
      }
      const validTokenId = tokensData.data.tokens[0].tokenId;

      // Get tools to find a valid toolId
      const toolsRes = await fetch(`${BASE_URL}/api/external/tools`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({}),
      });
      const toolsData = await toolsRes.json();
      const validToolId = toolsData.data?.tools?.[0]?.toolId || 'test-tool-id';

      const res = await fetch(`${BASE_URL}/api/external/user-servers/configure`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({
          tokenId: validTokenId,
          toolId: validToolId,
          authConf: [{ key: 'YOUR_API_KEY', value: 'test-api-key', dataType: 1 }],
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'valid request with remoteAuth (CustomRemote server) should return 200 or Core error',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      // First get a valid tokenId from the tokens list
      const tokensRes = await fetch(`${BASE_URL}/api/external/tokens`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({}),
      });
      const tokensData = await tokensRes.json();
      if (!tokensData.success || !tokensData.data?.tokens?.length) {
        console.log('    Skipped: No tokens found');
        return;
      }
      const validTokenId = tokensData.data.tokens[0].tokenId;

      const res = await fetch(`${BASE_URL}/api/external/user-servers/configure`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({
          tokenId: validTokenId,
          toolId: 'custom-remote-tool-id',
          remoteAuth: {
            params: { api_key: 'test-key' },
            headers: { Authorization: 'Bearer test-token' },
          },
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'valid request with restfulApiAuth bearer should return 200 or Core error',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      // First get a valid tokenId from the tokens list
      const tokensRes = await fetch(`${BASE_URL}/api/external/tokens`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({}),
      });
      const tokensData = await tokensRes.json();
      if (!tokensData.success || !tokensData.data?.tokens?.length) {
        console.log('    Skipped: No tokens found');
        return;
      }
      const validTokenId = tokensData.data.tokens[0].tokenId;

      const res = await fetch(`${BASE_URL}/api/external/user-servers/configure`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({
          tokenId: validTokenId,
          toolId: 'rest-api-tool-id',
          restfulApiAuth: {
            type: 'bearer',
            value: 'test-bearer-token',
          },
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
]);

// ============================================================
// POST /api/external/user-servers/unconfigure
// ============================================================

suite('user-servers/unconfigure', [
  {
    name: 'missing Authorization should return E2001',
    run: async () => {
      const res = await fetch(`${BASE_URL}/api/external/user-servers/unconfigure`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          tokenId: 'test-token-id',
          toolId: 'test-tool-id',
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'missing tokenId should return E1001',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/user-servers/unconfigure`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({
          toolId: 'test-tool-id',
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'missing toolId should return E1001',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/user-servers/unconfigure`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({
          tokenId: 'test-token-id',
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'non-existent tokenId should return E3003',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/user-servers/unconfigure`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({
          tokenId: 'non-existent-token-id',
          toolId: 'test-tool-id',
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'valid request should return 200 (idempotent)',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      // First get a valid tokenId from the tokens list
      const tokensRes = await fetch(`${BASE_URL}/api/external/tokens`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({}),
      });
      const tokensData = await tokensRes.json();
      if (!tokensData.success || !tokensData.data?.tokens?.length) {
        console.log('    Skipped: No tokens found');
        return;
      }
      const validTokenId = tokensData.data.tokens[0].tokenId;

      const res = await fetch(`${BASE_URL}/api/external/user-servers/unconfigure`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({
          tokenId: validTokenId,
          toolId: 'any-tool-id',
        }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
]);

// ============================================================
// POST /api/external/sessions
// ============================================================

suite('sessions', [
  {
    name: 'missing Authorization should return E2001',
    run: async () => {
      const res = await fetch(`${BASE_URL}/api/external/sessions`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({}),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'valid request without tokenId should return all sessions',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/sessions`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({}),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'non-existent tokenId should return E3003',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      const res = await fetch(`${BASE_URL}/api/external/sessions`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({ tokenId: 'non-existent-token-id' }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
  {
    name: 'valid tokenId should return sessions for that user',
    run: async () => {
      if (!OWNER_TOKEN) { console.log('    Skipped: Set OWNER_TOKEN'); return; }
      // First get a valid tokenId from the tokens list
      const tokensRes = await fetch(`${BASE_URL}/api/external/tokens`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({}),
      });
      const tokensData = await tokensRes.json();
      if (!tokensData.success || !tokensData.data?.tokens?.length) {
        console.log('    Skipped: No tokens found');
        return;
      }
      const validTokenId = tokensData.data.tokens[0].tokenId;

      const res = await fetch(`${BASE_URL}/api/external/sessions`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${OWNER_TOKEN}`,
        },
        body: JSON.stringify({ tokenId: validTokenId }),
      });
      const data = await res.json();
      console.log('    Status:', res.status, '| Response:', JSON.stringify(data));
    },
  },
]);

// ============================================================
// Run Tests
// ============================================================

async function runTests() {
  console.log('========================================');
  console.log('External API Test Suite');
  console.log(`Base URL: ${BASE_URL}`);
  console.log(`Owner Token: ${OWNER_TOKEN ? OWNER_TOKEN.slice(0, 8) + '...' : '(empty)'}`);
  if (TARGET_ENDPOINT) {
    console.log(`Target: ${TARGET_ENDPOINT}`);
  }
  console.log('========================================\n');

  const tokenRelatedEndpoints = new Set([
    'tokens',
    'tokens/get',
    'tokens/create',
    'tokens/update',
    'tokens/delete',
    'tokens/batch-update',
  ]);

  const isTokenGroupTarget = TARGET_ENDPOINT === 'token-all' || TARGET_ENDPOINT === 'tokens/all';
  tokenFlowContext.enabled = isTokenGroupTarget;
  tokenFlowContext.createdTokenIds = [];

  const targetSuites = TARGET_ENDPOINT
    ? (isTokenGroupTarget
      ? suites.filter((s) => tokenRelatedEndpoints.has(s.endpoint))
      : suites.filter((s) => s.endpoint === TARGET_ENDPOINT))
    : suites;

  if (targetSuites.length === 0) {
    console.log(`No tests found for endpoint: ${TARGET_ENDPOINT}`);
    console.log('Group targets: token-all, tokens/all');
    console.log('Available endpoints:', suites.map((s) => s.endpoint).join(', '));
    process.exit(1);
  }

  try {
    for (const s of targetSuites) {
      console.log(`[${s.endpoint}]`);
      for (const t of s.tests) {
        console.log(`  - ${t.name}`);
        try {
          await t.run();
        } catch (error) {
          console.log('    Error:', error);
        }
      }
      console.log('');
    }
  } finally {
    if (isTokenGroupTarget && OWNER_TOKEN && tokenFlowContext.createdTokenIds.length > 0) {
      console.log('[token-all cleanup]');
      while (tokenFlowContext.createdTokenIds.length > 0) {
        const tokenId = tokenFlowContext.createdTokenIds.shift() as string;
        try {
          const res = await fetch(`${BASE_URL}/api/external/tokens/delete`, {
            method: 'POST',
            headers: {
              'Content-Type': 'application/json',
              Authorization: `Bearer ${OWNER_TOKEN}`,
            },
            body: JSON.stringify({ tokenId }),
          });
          const data = await res.json();
          console.log(`  - delete ${tokenId} -> Status: ${res.status} | Response:`, JSON.stringify(data));
        } catch (error) {
          console.log(`  - delete ${tokenId} -> Error:`, error);
        }
      }
      console.log('');
    }
  }

  console.log('Done.');
}

runTests();
