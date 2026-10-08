package config

// Version injected at build time
var Version = "Snapshot"

// GraphQL logging flag
var GraphQLLogging bool = false

// Configuration file path given by --config, empty when not set
var ConfigFilePath string

// Agent session given by --agent-session, empty when not set
var AgentSession string

// Link to the agent session given by --agent-session-link, empty when not set
var AgentSessionLink string
