package types

// CallRecord represents a completed call for DynamoDB persistence
type CallRecord struct {
	DateKey      string  `json:"dateKey" dynamodbav:"DateKey"`           // YYYY-MM-DD (partition key)
	CallID       string  `json:"callId" dynamodbav:"CallID"`            // sort key
	VQ           VQName  `json:"vq" dynamodbav:"VQ"`
	Department   string  `json:"department" dynamodbav:"Department"`
	AgentID      string  `json:"agentId" dynamodbav:"AgentID"`
	EnqueueTime  string  `json:"enqueueTime" dynamodbav:"EnqueueTime"`  // RFC3339
	AssignTime   string  `json:"assignTime" dynamodbav:"AssignTime"`    // RFC3339
	CompleteTime string  `json:"completeTime" dynamodbav:"CompleteTime"` // RFC3339
	WaitTime     float64 `json:"waitTime" dynamodbav:"WaitTime"`        // seconds
	TalkTime     float64 `json:"talkTime" dynamodbav:"TalkTime"`        // seconds
	HoldTime     float64 `json:"holdTime" dynamodbav:"HoldTime"`        // seconds
	WrapTime     float64 `json:"wrapTime" dynamodbav:"WrapTime"`        // seconds
	HandleTime   float64 `json:"handleTime" dynamodbav:"HandleTime"`    // talk + hold + wrap
	Abandoned    bool    `json:"abandoned" dynamodbav:"Abandoned"`
	AnsweredInSL bool    `json:"answeredInSL" dynamodbav:"AnsweredInSL"`
}

// AgentDailyStats represents an agent's daily aggregated stats for DynamoDB
type AgentDailyStats struct {
	AgentID       string  `json:"agentId" dynamodbav:"AgentID"`         // partition key
	Date          string  `json:"date" dynamodbav:"Date"`               // YYYY-MM-DD (sort key)
	Department    string  `json:"department" dynamodbav:"Department"`
	TotalCalls    int     `json:"totalCalls" dynamodbav:"TotalCalls"`
	TotalTalkTime float64 `json:"totalTalkTime" dynamodbav:"TotalTalkTime"` // seconds
	TotalHoldTime float64 `json:"totalHoldTime" dynamodbav:"TotalHoldTime"` // seconds
	TotalWrapTime float64 `json:"totalWrapTime" dynamodbav:"TotalWrapTime"` // seconds
	TotalBreakTime float64 `json:"totalBreakTime" dynamodbav:"TotalBreakTime"` // seconds
	AvgHandleTime float64 `json:"avgHandleTime" dynamodbav:"AvgHandleTime"` // seconds
	Occupancy     float64 `json:"occupancy" dynamodbav:"Occupancy"`         // 0-100%
	LoginDuration float64 `json:"loginDuration" dynamodbav:"LoginDuration"` // seconds
}
