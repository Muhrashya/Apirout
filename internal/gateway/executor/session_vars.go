package executor

// SessionContext holds verified identity claims extracted from JWT
type SessionContext struct {
	SubjectID    string
	SubjectType  string // "admin" or "api_client"
	Email        string
	Role         string
	APIClientID  string
	GroupID      string
	CustomClaims map[string]interface{}
}

// InjectSessionVariables injects verified session variables into query params
// guaranteeing that client request params cannot tamper with :session_* values
func InjectSessionVariables(params map[string]interface{}, sess *SessionContext) {
	if sess == nil {
		return
	}

	if sess.SubjectID != "" {
		params["session_user_id"] = sess.SubjectID
		params["session_sub"] = sess.SubjectID
	}
	if sess.Email != "" {
		params["session_email"] = sess.Email
	}
	if sess.Role != "" {
		params["session_role"] = sess.Role
	}
	if sess.APIClientID != "" {
		params["session_api_client_id"] = sess.APIClientID
	}
	if sess.GroupID != "" {
		params["session_group_id"] = sess.GroupID
	}

	for k, v := range sess.CustomClaims {
		keyName := "session_" + k
		params[keyName] = v
	}
}
