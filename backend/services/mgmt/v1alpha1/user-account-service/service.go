package v1alpha1_useraccountservice

import (
	auth_client "github.com/Groupe-Hevea/neosync/backend/internal/auth/client"
	"github.com/Groupe-Hevea/neosync/backend/internal/userdata"
	"github.com/Groupe-Hevea/neosync/internal/authmgmt"
	"github.com/Groupe-Hevea/neosync/internal/billing"
	"github.com/Groupe-Hevea/neosync/internal/ee/license"
	"github.com/Groupe-Hevea/neosync/internal/ee/rbac"
	"github.com/Groupe-Hevea/neosync/internal/neosyncdb"
	"github.com/Groupe-Hevea/neosync/internal/temporal/clientmanager"
)

type Service struct {
	cfg                    *Config
	db                     *neosyncdb.NeosyncDb
	temporalConfigProvider clientmanager.ConfigProvider
	authclient             auth_client.Interface
	authadminclient        authmgmt.Interface
	billingclient          billing.Interface
	rbacClient             rbac.Interface
	licenseclient          license.EEInterface
}

type Config struct {
	IsAuthEnabled            bool
	IsNeosyncCloud           bool
	DefaultMaxAllowedRecords *int64
}

func New(
	cfg *Config,
	db *neosyncdb.NeosyncDb,
	temporalConfigProvider clientmanager.ConfigProvider,
	authclient auth_client.Interface,
	authadminclient authmgmt.Interface,
	billingclient billing.Interface,
	rbacClient rbac.Interface,
	licenseclient license.EEInterface,
) *Service {
	return &Service{
		cfg:                    cfg,
		db:                     db,
		temporalConfigProvider: temporalConfigProvider,
		authclient:             authclient,
		authadminclient:        authadminclient,
		billingclient:          billingclient,
		rbacClient:             rbacClient,
		licenseclient:          licenseclient,
	}
}

func (s *Service) UserDataClient() userdata.Interface {
	return userdata.NewClient(s, s.rbacClient, s.licenseclient)
}
