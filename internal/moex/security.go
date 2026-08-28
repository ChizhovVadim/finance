package moex

import (
	"finance/model"
	"fmt"
	"strconv"
	"strings"
)

func GetSecurityInfo(securityName string) (model.Security, error) {
	// квартальные фьючерсы
	if strings.HasPrefix(securityName, "Si") {
		securityCode, err := encodeSecurity(securityName)
		if err != nil {
			return model.Security{}, err
		}
		return model.Security{
			Name:           securityName,
			ClassCode:      FuturesClassCode,
			Code:           securityCode,
			PricePrecision: 0,
			PriceStep:      1,
			PriceStepCost:  1,
			Lever:          1,
		}, nil
	}
	if strings.HasPrefix(securityName, "CNY") {
		securityCode, err := encodeSecurity(securityName) //CR
		if err != nil {
			return model.Security{}, err
		}
		return model.Security{
			Name:           securityName,
			ClassCode:      FuturesClassCode,
			Code:           securityCode,
			PricePrecision: 3,
			PriceStep:      0.001,
			PriceStepCost:  1,
			Lever:          1000,
		}, nil
	}
	return model.Security{}, fmt.Errorf("secInfo not found %v", securityName)
}

// Sample: "Si-3.17" -> "SiH7"
// http://moex.com/s205
func encodeSecurity(securityName string) (string, error) {
	// вечные фьючерсы
	if strings.HasSuffix(securityName, "F") {
		return securityName, nil
	}

	const MonthCodes = "FGHJKMNQUVXZ"
	var parts = strings.SplitN(securityName, "-", 2)
	var name = parts[0]
	parts = strings.SplitN(parts[1], ".", 2)
	month, err := strconv.Atoi(parts[0])
	if err != nil {
		return "", err
	}
	year, err := strconv.Atoi(parts[1])
	if err != nil {
		return "", err
	}

	// курс китайский юань – российский рубль
	if name == "CNY" {
		name = "CR"
	}

	return fmt.Sprintf("%v%v%v", name, string(MonthCodes[month-1]), year%10), nil
}
