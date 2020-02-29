package gateways

import (
	"common"
	"fmt"
	"github.com/gocql/gocql"
	"time"
)

type CassandraConfig struct {
	ClusterIP         string
	ClusterPort       int
	Keyspace          string
	ServicesTableName string
}

type cassandraRepository struct {
	session *gocql.Session
	config  *CassandraConfig
}

func NewCassandraRepository(cfg *CassandraConfig) (*cassandraRepository, error) {
	session, err := newSession(cfg)
	if err != nil {
		return nil, err
	}

	repo := cassandraRepository{
		session: session,
		config:  cfg,
	}

	return &repo, nil
}

func (repo *cassandraRepository) StoreRegistrantInfo(rInfo *common.RegistrantInfo) error {
	if rInfo == nil {
		return nil
	}

	fmt.Printf("Inserting row: %+v", *rInfo)

	return repo.session.Query("INSERT INTO %s.%s(service_name, control_address, service_balancer_address, service_local_address) values(?, ?, ?, ?)",
		repo.config.Keyspace, repo.config.ServicesTableName,
		rInfo.ServiceName,
		rInfo.ControlAddress,
		rInfo.ServiceBalancerAddress,
		rInfo.ServiceLocalAddress).Exec()
}

func (repo *cassandraRepository) RemoveRegistrantInfo(rInfo *common.RegistrantInfo) error {
	if rInfo == nil {
		return nil
	}

	fmt.Printf("Removing row: %+v", *rInfo)

	return repo.session.Query("DELETE from %s.%s WHERE service_name=? AND control_address=?",
		repo.config.Keyspace, repo.config.ServicesTableName,
		rInfo.ServiceName,
		rInfo.ControlAddress).Exec()
}

func (repo *cassandraRepository) GetAllRegistrantInfos() ([]*common.RegistrantInfo, error) {
	serviceName := ""
	controlAddress := ""
	serviceBalancerAddress := ""
	serviceLocalAddress := ""
	rInfos := []*common.RegistrantInfo{}

	iter := repo.session.Query("SELECT service_name, control_address, service_balancer_address, service_local_address FROM %s", repo.config.ServicesTableName).Iter()
	for iter.Scan(&serviceName, &controlAddress, &serviceBalancerAddress, &serviceLocalAddress) {
		rInfo := common.NewRegistrantInfo(controlAddress, serviceName, serviceBalancerAddress, serviceLocalAddress)
		rInfos = append(rInfos, &rInfo)
	}

	if err := iter.Close(); err != nil {
		return nil, err
	}

	return rInfos, nil
}

func newSession(cfg *CassandraConfig) (*gocql.Session, error) {
	fmt.Printf("Creating CQL session...\n")

	cluster := gocql.NewCluster(cfg.ClusterIP)
	cluster.Port = cfg.ClusterPort
	cluster.Keyspace = cfg.Keyspace
	cluster.ProtoVersion = 4
	cluster.ConnectTimeout = 10 * time.Second
	cluster.DisableInitialHostLookup = true
	cluster.Consistency = gocql.One

	return cluster.CreateSession()
}
