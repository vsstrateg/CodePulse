package domain

type Test struct {
	ID             int    `json:"id"`
	TaskID         int    `json:"task_id"`
	Input          string `json:"input_data"`
	ExpectedOutput string `json:"expected_output"`
	IsSample       bool   `json:"is_sample"`
}
