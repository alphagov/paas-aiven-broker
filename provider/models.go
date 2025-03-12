package provider

import (
	"encoding/json"

	"github.com/pivotal-cf/brokerapi/v12/domain"
)

type ProvisionData struct {
	InstanceID    string
	Details       domain.ProvisionDetails
	Service       domain.Service
	Plan          domain.ServicePlan
	RawParameters json.RawMessage
}

type DeprovisionData struct {
	InstanceID string
	Details    domain.DeprovisionDetails
	Service    domain.Service
	Plan       domain.ServicePlan
}

type BindData struct {
	InstanceID string
	BindingID  string
	Details    domain.BindDetails
}

type UnbindData struct {
	InstanceID string
	BindingID  string
	Details    domain.UnbindDetails
}

type UpdateData struct {
	InstanceID    string
	Details       domain.UpdateDetails
	Service       domain.Service
	Plan          domain.ServicePlan
	RawParameters json.RawMessage
}

type LastOperationData struct {
	InstanceID    string
	OperationData string
}

type ProvisionParameters struct {
	UserIpFilter                  string  `json:"ip_filter"`
	RestoreFromLatestBackupOf     *string `json:"restore_from_latest_backup_of"`
	RestoreFromLatestBackupBefore *string `json:"restore_from_latest_backup_before"`
}

type UpdateParameters struct {
	UserIpFilter string `json:"ip_filter"`
}

func (pp *ProvisionParameters) Validate() error {
	return nil
}
