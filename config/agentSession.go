package config

import "os"

// GetAgentSession is the agent session of this run, and the link to it: from
// --agent-session and --agent-session-link, else from YONTRACK_AGENT_SESSION
// and YONTRACK_AGENT_SESSION_LINK. Each is empty when not set.
//
// They are not in the configuration file: a session lasts one run.
func GetAgentSession() (session string, link string) {
	session = AgentSession
	if session == "" {
		session = os.Getenv("YONTRACK_AGENT_SESSION")
	}
	link = AgentSessionLink
	if link == "" {
		link = os.Getenv("YONTRACK_AGENT_SESSION_LINK")
	}
	return session, link
}
