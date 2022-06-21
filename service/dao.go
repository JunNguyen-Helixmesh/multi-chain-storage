package service

import (
	"multi-chain-storage/common/constants"
	"multi-chain-storage/models"

	"github.com/filswan/go-swan-lib/logs"
)

func GetDeals2PreSign(signerWalletAddress string) ([]*models.Deal2PreSign, error) {
	signerWallet, err := models.GetWalletByAddress(signerWalletAddress, constants.WALLET_TYPE_META_MASK)
	if err != nil {
		logs.GetLogger().Error(err)
		return nil, err
	}

	deals2PreSign, err := models.GetDeals2PreSign(signerWallet.ID)
	if err != nil {
		logs.GetLogger().Error(err)
		return nil, err
	}

	return deals2PreSign, nil
}

func GetDeals2Sign(signerWalletAddress string) ([]*models.Deal2Sign, error) {
	signerWallet, err := models.GetWalletByAddress(signerWalletAddress, constants.WALLET_TYPE_META_MASK)
	if err != nil {
		logs.GetLogger().Error(err)
		return nil, err
	}

	deals2Sign, err := models.GetDeals2Sign(signerWallet.ID)
	if err != nil {
		logs.GetLogger().Error(err)
		return nil, err
	}

	for _, deal2Sign := range deals2Sign {
		daoSignatures, err := models.GetDaoSignaturesByOfflineDealId(deal2Sign.OfflineDealId)
		if err != nil {
			logs.GetLogger().Error(err)
			return nil, err
		}

		for i := 0; i < deal2Sign.BatchCount; i++ {
			signed := false
			for _, daodaoSignature := range daoSignatures {
				if daodaoSignature.BatchNo == i {
					signed = true
				}
			}

			if !signed {
				deal2Sign.BatchNo = append(deal2Sign.BatchNo, i)
			}
		}
	}

	return deals2Sign, nil
}

func RegisterDao(daoWalletAddress string) error {
	daoWallet, err := models.GetWalletByAddress(daoWalletAddress, constants.WALLET_TYPE_META_MASK)
	if err != nil {
		logs.GetLogger().Error(err)
		return err
	}

	if daoWallet.IsDao == nil || !*daoWallet.IsDao {
		err = models.SetWalletAsDao(daoWallet.ID)
		if err != nil {
			logs.GetLogger().Error(err)
			return err
		}
	}

	return nil
}
