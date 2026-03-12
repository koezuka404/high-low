package usecase

// AppError はゲーム API 用の構造化エラー。Controller で code に応じた HTTP ステータスに変換する。
type AppError struct {
	Code    string
	Message string
}

func (e *AppError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

// ゲームユースケース用エラー（仕様 7.1 等の code に対応）
var (
	errVersionConflict     = &AppError{Code: "version_conflict", Message: "version conflict"}
	errGameAlreadyStarted  = &AppError{Code: "game_already_started", Message: "game already started"}
	errGameNotStarted      = &AppError{Code: "game_not_started", Message: "game not started"}
	errGameNotFinished     = &AppError{Code: "game_not_finished", Message: "game not finished"}
	errCheatNotAllowed     = &AppError{Code: "cheat_not_allowed", Message: "cheat not allowed"}
	errCheatAlreadyUsed    = &AppError{Code: "cheat_already_used", Message: "cheat already used"}
	errCheatNotAvailable   = &AppError{Code: "cheat_not_available", Message: "cheat not available"}
	errInvalidInput        = &AppError{Code: "invalid_input", Message: "invalid input"}
	errInvalidMode         = &AppError{Code: "invalid_mode", Message: "invalid mode"}         // 7.5: mode が PLAYER/DEALER 以外
	errNoSelectableCard    = &AppError{Code: "invalid_game_state", Message: "no selectable card"}
	errForbidden           = &AppError{Code: "forbidden", Message: "forbidden"}
	errSessionNotFound     = &AppError{Code: "session_not_found", Message: "session not found"} // 7.3 / 7.4: セッション不存在 → 404
)
