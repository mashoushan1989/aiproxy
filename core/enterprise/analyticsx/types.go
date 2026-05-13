//go:build enterprise

package analyticsx

type DepartmentSummary struct {
	DepartmentID   string  `json:"department_id"`
	DepartmentName string  `json:"department_name"`
	Level1DeptID   string  `json:"level1_dept_id"`
	Level2DeptID   string  `json:"level2_dept_id"`
	MemberCount    int     `json:"member_count"`
	ActiveUsers    int     `json:"active_users"`
	RequestCount   int64   `json:"request_count"`
	UsedAmount     float64 `json:"used_amount"`
	TotalTokens    int64   `json:"total_tokens"`
	InputTokens    int64   `json:"input_tokens"`
	OutputTokens   int64   `json:"output_tokens"`
	SuccessRate    float64 `json:"success_rate"`
	AvgCost        float64 `json:"avg_cost"`
	AvgCostPerUser float64 `json:"avg_cost_per_user"`
	UniqueModels   int     `json:"unique_models"`
}

type UserRankingEntry struct {
	Rank                int     `json:"rank"`
	GroupID             string  `json:"group_id"`
	UserName            string  `json:"user_name"`
	DepartmentID        string  `json:"department_id"`
	DepartmentName      string  `json:"department_name"`
	UsedAmount          float64 `json:"used_amount"`
	RequestCount        int64   `json:"request_count"`
	TotalTokens         int64   `json:"total_tokens"`
	InputTokens         int64   `json:"input_tokens"`
	OutputTokens        int64   `json:"output_tokens"`
	CachedTokens        int64   `json:"cached_tokens"`
	CacheCreationTokens int64   `json:"cache_creation_tokens"`
	SuccessRate         float64 `json:"success_rate"`
	UniqueModels        int     `json:"unique_models"`
}

type ModelDistributionEntry struct {
	Model        string  `json:"model"`
	RequestCount int64   `json:"request_count"`
	TotalTokens  int64   `json:"total_tokens"`
	InputTokens  int64   `json:"input_tokens"`
	OutputTokens int64   `json:"output_tokens"`
	UsedAmount   float64 `json:"used_amount"`
	UniqueUsers  int     `json:"unique_users"`
	Percentage   float64 `json:"percentage"`
}
