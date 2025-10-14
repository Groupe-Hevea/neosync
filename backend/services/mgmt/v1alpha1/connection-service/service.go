package v1alpha1_connectionservice

import (
	"github.com/Groupe-Hevea/neosync/backend/internal/userdata"
	"github.com/Groupe-Hevea/neosync/backend/pkg/mongoconnect"
	"github.com/Groupe-Hevea/neosync/backend/pkg/sqlconnect"
	sql_manager "github.com/Groupe-Hevea/neosync/backend/pkg/sqlmanager"
	awsmanager "github.com/Groupe-Hevea/neosync/internal/aws"
	"github.com/Groupe-Hevea/neosync/internal/neosyncdb"
)

type Service struct {
	cfg            *Config
	db             *neosyncdb.NeosyncDb
	userclient     userdata.Interface
	sqlConnector   sqlconnect.SqlConnector
	sqlmanager     sql_manager.SqlManagerClient
	mongoconnector mongoconnect.Interface
	awsManager     awsmanager.NeosyncAwsManagerClient
}

type Config struct {
	IsNeosyncCloud bool
}

func New(
	cfg *Config,
	db *neosyncdb.NeosyncDb,
	userclient userdata.Interface,
	mongoconnector mongoconnect.Interface,
	awsManager awsmanager.NeosyncAwsManagerClient,
	sqlmanager sql_manager.SqlManagerClient,
	sqlconnector sqlconnect.SqlConnector,
) *Service {
	return &Service{
		cfg:            cfg,
		db:             db,
		userclient:     userclient,
		sqlmanager:     sqlmanager,
		mongoconnector: mongoconnector,
		awsManager:     awsManager,
		sqlConnector:   sqlconnector,
	}
}
