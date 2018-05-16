package mysqlpool

import (
	. "business_center/def"
	"fmt"
)

func QueryAssetProperty(queryMap map[string]interface{}) ([]AssetProperty, bool) {
	sqls := "select asset_name,full_name,is_token,parent_name,logo,deposit_min,withdrawal_rate," +
		"withdrawal_value,withdrawal_reserve_rate,withdrawal_alert_rate,withdrawal_stategy,confirmation_num," +
		"decaimal,gas_factor,debt,park_amount from asset_property where true"

	assetProperty := make([]AssetProperty, 0)
	params := make([]interface{}, 0)

	if len(queryMap) > 0 {
		sqls += andConditions(queryMap, &params)
		sqls += andPagination(queryMap, &params)
	}

	db := Get()
	rows, err := db.Query(sqls, params...)
	defer rows.Close()
	if err != nil {
		fmt.Println(err.Error())
		return assetProperty, len(assetProperty) > 0
	}

	var data AssetProperty
	for rows.Next() {
		err := rows.Scan(&data.AssetName, &data.FullName, &data.IsToken, &data.ParentName, &data.Logo,
			&data.DepositMin, &data.WithdrawalRate, &data.WithdrawalValue, &data.WithdrawalReserveRate,
			&data.WithdrawalAlertRate, &data.WithdrawalStategy, &data.ConfirmationNum, &data.Decimal,
			&data.GasFactor, &data.Debt, &data.ParkAmount)
		if err == nil {
			assetProperty = append(assetProperty, data)
		}
	}
	return assetProperty, len(assetProperty) > 0
}

func QueryAssetPropertyCount(queryMap map[string]interface{}) int {
	sqls := "select count(*) from asset_property" +
		" where true"

	count := 0
	params := make([]interface{}, 0)

	if len(queryMap) > 0 {
		sqls += andConditions(queryMap, &params)
	}

	db := Get()
	db.QueryRow(sqls, params...).Scan(&count)
	return count
}

func QueryAssetPropertyByName(assetName string) (AssetProperty, bool) {
	queryMap := make(map[string]interface{})
	queryMap["asset_name"] = assetName
	if assetProperty, ok := QueryAssetProperty(queryMap); ok {
		return assetProperty[0], true
	}
	return AssetProperty{}, false
}
