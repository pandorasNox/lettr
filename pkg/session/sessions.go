package session

import "slices"

type Sessions struct {
	sessions []session
}

func NewSessions() Sessions {
	return Sessions{}
}

func (ss Sessions) String() string {
	out := ""
	for _, s := range ss.sessions {
		out = out + s.id + " " + s.expiresAt.String() + "\n"
	}

	return out
}

func (ss *Sessions) UpdateOrSet(sess session) {
	index := slices.IndexFunc((ss.sessions), func(s session) bool {
		return s.id == sess.id
	})
	if index == -1 {
		ss.sessions = append(ss.sessions, sess)
		return
	}

	(ss.sessions)[index] = sess
}