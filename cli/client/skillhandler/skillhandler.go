package skillhandler

import (
	"encoding/json"
	"fmt"
	"io"
	"mifer/pkg/errorer"
	"net/http"
	"strings"
)

type skillInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Context     string `json:"context"`
	Agent       string `json:"agent"`
}

type listResp struct {
	Skills []skillInfo `json:"skills"`
}

// List 查询技能列表，返回格式化后的状态信息
func (h *SkillHandler) List() (string, error) {
	req, err := http.NewRequest(http.MethodGet, h.url, nil)
	if err != nil {
		return "", errorer.NewS(errorer.ErrCreateRequestFailed, err)
	}

	resp, err := h.http.Do(req)
	if err != nil {
		return "", errorer.NewS(errorer.ErrRequestFailed, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", errorer.NewS(errorer.ErrReadResponseFailed, err)
	}

	var lr listResp
	if err := json.Unmarshal(respBody, &lr); err != nil {
		return "", errorer.NewS(errorer.ErrParseResponseFailed, err)
	}

	if len(lr.Skills) == 0 {
		return "没有已加载的技能", nil
	}

	var sb strings.Builder
	sb.WriteString("已加载的技能:")
	for _, s := range lr.Skills {
		mode := s.Context
		if mode == "" {
			mode = "inline"
		}
		info := fmt.Sprintf("\n  ● %-18s %s", s.Name, s.Description)
		if s.Context == "fork" {
			info += fmt.Sprintf("  (fork → %s)", s.Agent)
		} else {
			info += fmt.Sprintf("  (%s)", mode)
		}
		sb.WriteString(info)
	}
	return sb.String(), nil
}
