package v1alpha1_apikeyservice

import (
	"github.com/Groupe-Hevea/neosync/backend/internal/userdata"
	"github.com/Groupe-Hevea/neosync/internal/neosyncdb"
)

type Service struct {
	cfg            *Config
	db             *neosyncdb.NeosyncDb
	userdataclient userdata.Interface
}

type Config struct {
	IsAuthEnabled bool
}

func New(
	cfg *Config,
	db *neosyncdb.NeosyncDb,
	userdataclient userdata.Interface,
) *Service {
	return &Service{cfg: cfg, db: db, userdataclient: userdataclient}
}
