// Package ghauth проверяет GitHub Personal Access Token и знает, где его
// создать — это адрес, который экран входа показывает пользователю сразу.
package ghauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"tgit/internal/i18n"
)

// TokenCreateURL сразу открывает форму создания classic-токена на GitHub
// с нужным описанием и запрошенными правами (repo — доступ к репозиториям,
// read:user — чтобы показать имя пользователя после входа).
const TokenCreateURL = "https://github.com/settings/tokens/new?description=tgit-cli&scopes=repo,read:user"

// User — минимальные данные о владельце токена.
type User struct {
	Login string `json:"login"`
	Name  string `json:"name"`
}

var httpClient = &http.Client{Timeout: 10 * time.Second}

// ValidateToken обращается к GitHub API и возвращает пользователя, если токен рабочий.
func ValidateToken(ctx context.Context, token string) (*User, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf(i18n.T.NetworkErrFmt, err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusUnauthorized:
		return nil, fmt.Errorf("%s", i18n.T.TokenRejectedMsg)
	default:
		return nil, fmt.Errorf(i18n.T.UnexpectedStatusFmt, resp.StatusCode)
	}

	var u User
	if err := json.NewDecoder(resp.Body).Decode(&u); err != nil {
		return nil, fmt.Errorf(i18n.T.ParseFailedFmt, err)
	}
	return &u, nil
}
