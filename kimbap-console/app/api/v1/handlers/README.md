# Protocol Handlers (Legacy — Frozen)

> **DEPRECATION NOTICE:** The `cmdId`-based protocol system in this directory is **frozen**.
> No new protocol handlers should be added here. All new API features MUST be implemented
> as RESTful endpoints under `/api/external/*`.
>
> Existing handlers will be maintained for backward compatibility and will be incrementally
> migrated to REST endpoints over time.

## Migration Guide

### For new features

Instead of creating a new `protocol-XXXXX.ts` handler, create a REST endpoint:

```
app/api/external/
  └── your-resource/
      └── route.ts          # GET, POST, PUT, DELETE handlers
```

Example:

```typescript
// app/api/external/tokens/route.ts
import { NextRequest, NextResponse } from 'next/server';

export async function GET(request: NextRequest) {
  // List tokens
  return NextResponse.json({ data: tokens });
}

export async function POST(request: NextRequest) {
  // Create token
  return NextResponse.json({ data: newToken }, { status: 201 });
}
```

### For migrating existing handlers

1. Create the equivalent REST endpoint under `/api/external/`
2. Mark the old protocol handler with a `@deprecated` JSDoc comment
3. Update the frontend to call the new REST endpoint
4. Keep the old handler for backward compatibility (do not delete)

---

## Legacy Architecture (reference only)

- All API requests go to `/api/v1` endpoint
- The router extracts `cmdId` from `body.common.cmdId`
- Each protocol is handled by a separate file named `protocol-XXXXX.ts`
- Response formatting is handled centrally by the router

## Adding a New Protocol (DEPRECATED — do not follow this pattern)

```typescript
// Example: protocol-10002.ts
interface Request10002 {
  common: {
    cmdId: number;
  };
  params: {
    // Your parameters here
  };
}

interface Response10002Data {
  // Your response data structure
}

export async function handleProtocol10002(body: Request10002): Promise<Response10002Data> {
  // Validate parameters
  if (!body.params?.someField) {
    throw new Error('someField is required');
  }

  // Your business logic here
  
  // Return data (no need to wrap in common/data structure)
  return {
    // Your response data
  };
}
```

2. Register the handler in `index.ts`:

```typescript
import { handleProtocol10002 } from './protocol-10002';

export const protocolHandlers: Record<number, ProtocolHandler> = {
  10001: handleProtocol10001,
  10002: handleProtocol10002, // Add your new handler here
};
```

## Benefits

- **Centralized routing**: All requests go through `/api/v1`
- **Consistent responses**: Response formatting is handled by the router
- **Easy to maintain**: Each protocol has its own file
- **Type safety**: Each handler can define its own request/response types
- **Error handling**: Errors are automatically caught and formatted

## Example API Call

```bash
curl -X POST http://localhost:3000/api/v1 \
  -H "Content-Type: application/json" \
  -d '{
    "common": {
      "cmdId": 10001
    },
    "params": {
      "masterPwd": "your-password"
    }
  }'
```

## Protocol Documentation

### Protocol 10001 - Master Password Authentication
**Purpose**: Creates the initial proxy server and owner user with master password authentication.

**Request Parameters**:
- `masterPwd` (string, required): Master password for authentication

**Response Data**:
- `accessToken` (string): Generated authentication token
- `proxyId` (number): Created proxy server ID
- `proxyName` (string): Name of the proxy server
- `status` (number): Server status (1=running, 2=stopped)
- `role` (number): User role (1=owner)

**Implementation Notes**:
- Generates a unique access token
- Encrypts the token using the master password
- Creates a proxy record with default name "My MCP Server"
- Creates an owner user with full permissions
- TODO: Actual MCP server startup logic pending implementation

### Protocol 10002 - Get Proxy Info
**Purpose**: Retrieves information about the existing proxy server.

**Request Parameters**: None

**Response Data**:
- `proxyId` (number): Proxy server ID
- `proxyName` (string): Name of the proxy server
- `status` (number): Server running status (1=running, 2=stopped)
- `createdAt` (number): Unix timestamp of creation

**Implementation Notes**:
- Returns the first proxy record from the database
- Server status is currently hardcoded to running

### Protocol 10003 - Proxy Management
**Purpose**: Manages proxy server operations including editing, starting/stopping, and backup/restore.

**Request Parameters**:
- `handleType` (number, required): Operation type
  - 1: Edit base info
  - 2: Start server
  - 3: Stop server
  - 4: Backup to cloud
  - 5: Backup to local
  - 6: Restore from cloud
  - 7: Restore from local file
- `proxyId` (number, required): Target proxy ID
- `proxyName` (string, optional): New name for edit operation

**Response Data**:
- `success` (boolean): Operation success status
- `message` (string, optional): Status message

**Implementation Notes**:
- Only handleType=1 (edit base info) is currently implemented
- Other operations return success with TODO messages

### Protocol 10004 - Fetch Tool Templates
**Purpose**: Retrieves available tool templates from the cloud API.

**Request Parameters**: None

**Response Data**:
- `templates` (array): Array of tool templates containing:
  - `toolTmplId` (string): Template identifier
  - `toolType` (string): Type of tool
  - `name` (string): Tool name
  - `description` (string): Tool description
  - `tags` (array): Tool tags
  - `authtags` (array): Authentication tags
  - `credentials` (array): Required credentials configuration

**Implementation Notes**:
- Makes HTTP POST request to cloud API endpoint
- Handles JSON parsing for nested fields
- Configured with 15-second timeout
- Uses axios for proxy support

### Protocol 10005 - Tool Management
**Purpose**: Comprehensive tool management including add, edit, enable/disable, and delete operations.

**Request Parameters**:
- `handleType` (number, required): Operation type
  - 1: Add tool
  - 2: Edit tool
  - 3: Enable server
  - 4: Disable server
  - 5: Delete server
- `proxyId` (number, optional): Proxy ID for ownership verification
- `toolId` (string, optional): Tool/server ID for operations 2-5
- `toolTmplId` (string, optional): Template ID when adding tools
- `masterPwd` (string, optional): Master password for encryption
- `authConf` (array, optional): Authentication configuration
- `functions` (array, optional): Tool functions to enable/disable
- `resources` (array, optional): Tool resources to enable/disable

**Response Data**: Varies by operation type
- Add tool: Returns `serverId`
- Edit/Enable/Disable: Returns `success` and `message`
- Delete: Returns `success` and `message`

**Implementation Notes**:
- Implements encryption/decryption of launch configurations
- Validates proxy ownership before operations
- Manages server records in database
- TODO: Actual MCP server lifecycle management pending

### Protocol 10006 - List Tools
**Purpose**: Retrieves list of configured tools/servers with filtering options.

**Request Parameters**:
- `handleType` (number, required): Filter type
  - 1: Get all servers
  - 2: Get enabled servers only
  - 3-4: Not implemented
- `proxyId` (number, optional): Filter by proxy ID

**Response Data**:
- `toolList` (array): Array of tools containing:
  - `toolTmplId` (string): Template ID
  - `toolType` (string): Tool type
  - `name` (string): Tool/server name
  - `credentials` (object): Credential configuration
  - `toolFuncs` (array): Enabled functions
  - `toolResources` (array): Enabled resources
  - `lastUsed` (number): Last usage timestamp
  - `enabled` (boolean): Server enabled status
  - `toolId` (string): Server ID

**Implementation Notes**:
- Queries servers from database with optional filtering
- Enriches data by matching with cloud tool templates
- Parses capabilities JSON to extract functions and resources
- Handles missing template data gracefully