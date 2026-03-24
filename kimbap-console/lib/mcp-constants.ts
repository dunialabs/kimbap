// Skills configuration
export const SKILLS_MAX_FILE_SIZE = 10 * 1024 * 1024 // 10MB

export interface ServerMetadata {
  type: string
  name: string
  icon: string
  color: string
  bgColor: string
  description: string
}

export interface MCPFunction {
  id: string
  name: string
  description: string
  category: string
}

export const SERVER_METADATA: Record<string, ServerMetadata> = {
  github: {
    type: 'github',
    name: 'GitHub',
    icon: 'Github',
    color: 'bg-gray-900',
    bgColor: 'bg-gray-100 dark:bg-gray-800',
    description: 'Repository management and collaboration'
  },
  notion: {
    type: 'notion',
    name: 'Notion',
    icon: 'FileText',
    color: 'bg-black',
    bgColor: 'bg-gray-100 dark:bg-gray-800',
    description: 'Workspace and knowledge management'
  },
  figma: {
    type: 'figma',
    name: 'Figma',
    icon: 'Palette',
    color: 'bg-purple-600',
    bgColor: 'bg-purple-100 dark:bg-purple-900/30',
    description: 'Design collaboration platform'
  },
  postgresql: {
    type: 'postgresql',
    name: 'PostgreSQL',
    icon: 'Database',
    color: 'bg-blue-700',
    bgColor: 'bg-blue-100 dark:bg-blue-900/30',
    description: 'Relational database management'
  },
  googledrive: {
    type: 'googledrive',
    name: 'Google Drive',
    icon: 'FolderOpen',
    color: 'bg-yellow-600',
    bgColor: 'bg-yellow-100 dark:bg-yellow-900/30',
    description: 'Cloud storage and collaboration'
  },
  'sequential-thinking': {
    type: 'sequential-thinking',
    name: 'Sequential Thinking',
    icon: 'Brain',
    color: 'bg-indigo-600',
    bgColor: 'bg-indigo-100 dark:bg-indigo-900/30',
    description: 'Dynamic and reflective problem solving'
  },
  wcgw: {
    type: 'wcgw',
    name: 'WCGW',
    icon: 'Terminal',
    color: 'bg-red-600',
    bgColor: 'bg-red-100 dark:bg-red-900/30',
    description: 'Command execution with safety controls'
  },
  slack: {
    type: 'slack',
    name: 'Slack',
    icon: 'MessageSquare',
    color: 'bg-purple-600',
    bgColor: 'bg-purple-100 dark:bg-purple-900/30',
    description: 'Team communication and collaboration'
  },
  aws: {
    type: 'aws',
    name: 'AWS',
    icon: 'Cloud',
    color: 'bg-orange-500',
    bgColor: 'bg-orange-100 dark:bg-orange-900/30',
    description: 'Amazon Web Services cloud platform'
  },
  mysql: {
    type: 'mysql',
    name: 'MySQL',
    icon: 'Database',
    color: 'bg-blue-600',
    bgColor: 'bg-blue-100 dark:bg-blue-900/30',
    description: 'MySQL database management'
  },
  redis: {
    type: 'redis',
    name: 'Redis',
    icon: 'Database',
    color: 'bg-red-600',
    bgColor: 'bg-red-100 dark:bg-red-900/30',
    description: 'In-memory data structure store'
  },
  stripe: {
    type: 'stripe',
    name: 'Stripe',
    icon: 'CreditCard',
    color: 'bg-blue-600',
    bgColor: 'bg-blue-100 dark:bg-blue-900/30',
    description: 'Payment processing platform'
  },
  linear: {
    type: 'linear',
    name: 'Linear',
    icon: 'Terminal',
    color: 'bg-blue-600',
    bgColor: 'bg-blue-100 dark:bg-blue-900/30',
    description: 'Issue tracking and project management'
  },
  openai: {
    type: 'openai',
    name: 'OpenAI',
    icon: 'Bot',
    color: 'bg-green-600',
    bgColor: 'bg-green-100 dark:bg-green-900/30',
    description: 'AI language models and tools'
  },
  elasticsearch: {
    type: 'elasticsearch',
    name: 'Elasticsearch',
    icon: 'Search',
    color: 'bg-yellow-600',
    bgColor: 'bg-yellow-100 dark:bg-yellow-900/30',
    description: 'Search and analytics engine'
  },
  salesforce: {
    type: 'salesforce',
    name: 'Salesforce',
    icon: 'Cloud',
    color: 'bg-blue-600',
    bgColor: 'bg-blue-100 dark:bg-blue-900/30',
    description: 'Customer relationship management'
  },
  mongodb: {
    type: 'mongodb',
    name: 'MongoDB',
    icon: 'Database',
    color: 'bg-green-600',
    bgColor: 'bg-green-100 dark:bg-green-900/30',
    description: 'NoSQL document database'
  },
  'brave-search': {
    type: 'brave-search',
    name: 'Brave Search',
    icon: 'Search',
    color: 'bg-orange-600',
    bgColor: 'bg-orange-100 dark:bg-orange-900/30',
    description: 'Privacy-focused web search'
  }
}

export type ServerType = keyof typeof SERVER_METADATA

export const PRESET_TOOLS = [
  {
    name: 'GitHub',
    description: 'Repository management and collaboration with fine-grained permissions',
    type: 'development' as const,
    serverType: 'github' as ServerType,
    auth: 'oauth' as const,
  },
  {
    name: 'Notion',
    description: 'Workspace and knowledge management with page and database permissions',
    type: 'productivity' as const,
    serverType: 'notion' as ServerType,
    auth: 'oauth' as const,
  },
  {
    name: 'Figma',
    description: 'Design collaboration platform with project access controls',
    type: 'design' as const,
    serverType: 'figma' as ServerType,
    auth: 'api_key' as const,
  },
  {
    name: 'PostgreSQL',
    description: 'Relational database with advanced querying and data management',
    type: 'database' as const,
    serverType: 'postgresql' as ServerType,
    auth: 'database_connection' as const,
  },
  {
    name: 'Google Drive',
    description: 'Cloud storage with file sharing and collaborative editing',
    type: 'storage' as const,
    serverType: 'googledrive' as ServerType,
    auth: 'oauth' as const,
  },
  {
    name: 'Sequential Thinking',
    description: 'Dynamic and reflective problem solving with chain-of-thought',
    type: 'ai' as const,
    serverType: 'sequential-thinking' as ServerType,
    auth: 'api_key' as const,
  },
  {
    name: 'WCGW',
    description: 'Command execution environment with safety controls and monitoring',
    type: 'development' as const,
    serverType: 'wcgw' as ServerType,
    auth: 'none' as const,
  },
  {
    name: 'Slack',
    description: 'Team communication with channel management and message automation',
    type: 'communication' as const,
    serverType: 'slack' as ServerType,
    auth: 'oauth' as const,
  },
  {
    name: 'AWS',
    description: 'Amazon Web Services with resource management and monitoring',
    type: 'cloud' as const,
    serverType: 'aws' as ServerType,
    auth: 'api_key' as const,
  },
  {
    name: 'MySQL',
    description: 'MySQL database with query execution and schema management',
    type: 'database' as const,
    serverType: 'mysql' as ServerType,
    auth: 'username_password' as const,
  },
  {
    name: 'Redis',
    description: 'In-memory data store with caching and pub/sub capabilities',
    type: 'database' as const,
    serverType: 'redis' as ServerType,
    auth: 'api_key' as const,
  },
  {
    name: 'Stripe',
    description: 'Payment processing with transaction management and analytics',
    type: 'finance' as const,
    serverType: 'stripe' as ServerType,
    auth: 'api_key' as const,
  },
  {
    name: 'Linear',
    description: 'Issue tracking and project management with workflow automation',
    type: 'productivity' as const,
    serverType: 'linear' as ServerType,
    auth: 'api_key' as const,
  },
  {
    name: 'OpenAI',
    description: 'AI language models for chat, completion, and embedding generation',
    type: 'ai' as const,
    serverType: 'openai' as ServerType,
    auth: 'api_key' as const,
  },
  {
    name: 'Elasticsearch',
    description: 'Search and analytics engine with full-text search capabilities',
    type: 'database' as const,
    serverType: 'elasticsearch' as ServerType,
    auth: 'api_key' as const,
  },
  {
    name: 'Salesforce',
    description: 'Customer relationship management with lead and opportunity tracking',
    type: 'business' as const,
    serverType: 'salesforce' as ServerType,
    auth: 'oauth' as const,
  },
  {
    name: 'MongoDB',
    description: 'NoSQL document database with flexible schema and aggregation',
    type: 'database' as const,
    serverType: 'mongodb' as ServerType,
    auth: 'database_connection' as const,
  },
  {
    name: 'Brave Search',
    description: 'Privacy-focused web search with ad-free results and ranking',
    type: 'search' as const,
    serverType: 'brave-search' as ServerType,
    auth: 'api_key' as const,
  }
]

export const FUNCTIONS_MAP: Record<string, MCPFunction[]> = {
  github: [
    { id: 'createRepository', name: 'Create Repository', description: 'Create a new GitHub repository', category: 'Repository Management' },
    { id: 'readFile', name: 'Read File', description: 'Read file contents from repository', category: 'File Operations' },
    { id: 'writeFile', name: 'Write File', description: 'Write file contents to repository', category: 'File Operations' },
    { id: 'createIssue', name: 'Create Issue', description: 'Create a new issue in repository', category: 'Issue Management' },
    { id: 'listIssues', name: 'List Issues', description: 'List issues in repository', category: 'Issue Management' },
    { id: 'createPullRequest', name: 'Create Pull Request', description: 'Create a new pull request', category: 'Pull Requests' },
  ],
  notion: [
    { id: 'createPage', name: 'Create Page', description: 'Create a new Notion page', category: 'Page Management' },
    { id: 'updatePage', name: 'Update Page', description: 'Update existing Notion page', category: 'Page Management' },
    { id: 'queryDatabase', name: 'Query Database', description: 'Query Notion database', category: 'Database Operations' },
    { id: 'createDatabase', name: 'Create Database', description: 'Create a new database', category: 'Database Operations' },
  ],
  postgresql: [
    { id: 'executeQuery', name: 'Execute Query', description: 'Execute SQL query', category: 'Query Operations' },
    { id: 'createTable', name: 'Create Table', description: 'Create new database table', category: 'Schema Management' },
    { id: 'insertData', name: 'Insert Data', description: 'Insert data into table', category: 'Data Operations' },
    { id: 'updateData', name: 'Update Data', description: 'Update existing data', category: 'Data Operations' },
  ],
  slack: [
    { id: 'sendMessage', name: 'Send Message', description: 'Send message to channel', category: 'Messaging' },
    { id: 'listChannels', name: 'List Channels', description: 'List available channels', category: 'Channel Management' },
    { id: 'createChannel', name: 'Create Channel', description: 'Create new channel', category: 'Channel Management' },
  ],
  openai: [
    { id: 'createChatCompletion', name: 'Chat Completion', description: 'Generate chat completion', category: 'Text Generation' },
    { id: 'createEmbedding', name: 'Create Embedding', description: 'Generate text embeddings', category: 'Embeddings' },
    { id: 'listModels', name: 'List Models', description: 'List available models', category: 'Model Management' },
  ],
  stripe: [
    { id: 'createPaymentIntent', name: 'Create Payment Intent', description: 'Create payment intent', category: 'Payments' },
    { id: 'listCustomers', name: 'List Customers', description: 'List customers', category: 'Customer Management' },
    { id: 'createCustomer', name: 'Create Customer', description: 'Create new customer', category: 'Customer Management' },
  ]
}