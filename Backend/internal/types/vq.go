package types

// VQName represents a virtual queue identifier
type VQName string

// Sales VQs
const (
	VQSalesInbound    VQName = "sales_inbound"
	VQSalesOutbound   VQName = "sales_outbound"
	VQSalesCallback   VQName = "sales_callback"
	VQSalesChat       VQName = "sales_chat"
)

// Support VQs
const (
	VQSupportGeneral  VQName = "support_general"
	VQSupportBilling  VQName = "support_billing"
	VQSupportCallback VQName = "support_callback"
	VQSupportChat     VQName = "support_chat"
)

// Technical VQs
const (
	VQTechL1          VQName = "tech_l1"
	VQTechL2          VQName = "tech_l2"
	VQTechCallback    VQName = "tech_callback"
	VQTechChat        VQName = "tech_chat"
)

// Retention VQs
const (
	VQRetentionSave   VQName = "retention_save"
	VQRetentionCancel VQName = "retention_cancel"
	VQRetentionCallback VQName = "retention_callback"
	VQRetentionChat   VQName = "retention_chat"
)

// VQDepartmentMapping maps each VQ to its department
var VQDepartmentMapping = map[VQName]Department{
	VQSalesInbound:      DeptSales,
	VQSalesOutbound:     DeptSales,
	VQSalesCallback:     DeptSales,
	VQSalesChat:         DeptSales,
	VQSupportGeneral:    DeptSupport,
	VQSupportBilling:    DeptSupport,
	VQSupportCallback:   DeptSupport,
	VQSupportChat:       DeptSupport,
	VQTechL1:            DeptTechnical,
	VQTechL2:            DeptTechnical,
	VQTechCallback:      DeptTechnical,
	VQTechChat:          DeptTechnical,
	VQRetentionSave:     DeptRetention,
	VQRetentionCancel:   DeptRetention,
	VQRetentionCallback: DeptRetention,
	VQRetentionChat:     DeptRetention,
}

// DepartmentVQs maps each department to its VQs
var DepartmentVQs = map[Department][]VQName{
	DeptSales:     {VQSalesInbound, VQSalesOutbound, VQSalesCallback, VQSalesChat},
	DeptSupport:   {VQSupportGeneral, VQSupportBilling, VQSupportCallback, VQSupportChat},
	DeptTechnical: {VQTechL1, VQTechL2, VQTechCallback, VQTechChat},
	DeptRetention: {VQRetentionSave, VQRetentionCancel, VQRetentionCallback, VQRetentionChat},
}

// AllVQs returns all virtual queue names
var AllVQs = []VQName{
	VQSalesInbound, VQSalesOutbound, VQSalesCallback, VQSalesChat,
	VQSupportGeneral, VQSupportBilling, VQSupportCallback, VQSupportChat,
	VQTechL1, VQTechL2, VQTechCallback, VQTechChat,
	VQRetentionSave, VQRetentionCancel, VQRetentionCallback, VQRetentionChat,
}
