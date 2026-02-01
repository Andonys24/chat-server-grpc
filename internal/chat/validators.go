package chat

import "regexp"

var nicknameRegex = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]{2,11}$`)

func IsValidNickname(nickname string) bool {
	return nicknameRegex.MatchString(nickname)
}
