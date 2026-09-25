package config

// Version injected at build time
var Version = "Snapshot"

// GraphQL logging flag
var GraphQLLogging bool = false

// Configuration file path given by --config, empty when not set
var ConfigFilePath string
