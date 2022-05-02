package models

import (
	"multi-chain-storage/database"

	"github.com/filswan/go-swan-lib/logs"
)

type Network struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	RpcUrl      string `json:"rpc_url"`
	Description string `json:"description"`
	CreateAt    int64  `json:"create_at"`
	UpdateAt    int64  `json:"update_at"`
}

func GetNetworkByName(name string) (*Network, error) {
	var networks []*Network
	err := database.GetDB().Where("name=?", name).Find(&networks).Error
	if err != nil {
		logs.GetLogger().Error(err)
		return nil, err
	}

	if len(networks) > 0 {
		return networks[0], nil
	}

	return nil, nil
}
