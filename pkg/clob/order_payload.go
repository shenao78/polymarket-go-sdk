package clob

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/GoPolymarket/polymarket-go-sdk/pkg/clob/clobtypes"
	"github.com/GoPolymarket/polymarket-go-sdk/pkg/types"
)

func buildOrderPayload(order *clobtypes.SignedOrder) (string, error) {
	if order == nil {
		return "", fmt.Errorf("order is required")
	}
	orderType := normalizeOrderType(order.OrderType, clobtypes.OrderTypeGTC)
	orderPart, err := orderWithSignature(order)
	if err != nil {
		return "", err
	}

	res := "{" +
		"\"order\":" + orderPart + "," +
		"\"orderType\":\"" + string(orderType) + "\"," +
		"\"owner\":\"" + order.Owner + "\""

	if order.PostOnly != nil {
		res += ",\"postOnly\":" + strconv.FormatBool(*order.PostOnly)
	}
	if order.DeferExec != nil {
		res += ",\"deferExec\":" + strconv.FormatBool(*order.DeferExec)
	}

	res += "}"
	return res, nil
}

func buildOrdersPayload(orders *clobtypes.SignedOrders) (string, error) {
	if orders == nil {
		return "", fmt.Errorf("orders are required")
	}

	res := "["
	numOrders := len(orders.Orders)

	for i := range orders.Orders {
		payload, err := buildOrderPayload(&orders.Orders[i])
		if err != nil {
			return "", err
		}

		res += payload
		if i < numOrders-1 {
			res += ","
		}
	}

	res += "]"
	return res, nil
}

func orderWithSignature(order *clobtypes.SignedOrder) (string, error) {
	if order == nil {
		return "", fmt.Errorf("order is required")
	}
	if order.Signature == "" {
		return "", fmt.Errorf("signature is required")
	}
	if order.Owner == "" {
		return "", fmt.Errorf("owner is required")
	}

	sigType := 0
	if order.Order.SignatureType != nil {
		sigType = *order.Order.SignatureType
	}

	side := strings.ToUpper(order.Order.Side)
	if side != "BUY" && side != "SELL" {
		return "", fmt.Errorf("invalid order side %q", order.Order.Side)
	}

	salt, err := saltToJSON(order.Order.Salt)
	if err != nil {
		return "", err
	}

	saltStr := strconv.FormatUint(salt.(uint64), 10)
	return "{" +
		"\"expiration\":\"" + u256String(order.Order.Expiration) + "\"," +
		"\"feeRateBps\":\"" + decimalString(order.Order.FeeRateBps) + "\"," +
		"\"maker\":\"" + order.Order.Maker.Hex() + "\"," +
		"\"makerAmount\":\"" + decimalString(order.Order.MakerAmount) + "\"," +
		"\"nonce\":\"" + u256String(order.Order.Nonce) + "\"," +
		"\"salt\":" + saltStr + "," +
		"\"side\":\"" + side + "\"," +
		"\"signature\":\"" + order.Signature + "\"," +
		"\"signatureType\":" + strconv.Itoa(sigType) + "," +
		"\"signer\":\"" + order.Order.Signer.Hex() + "\"," +
		"\"taker\":\"" + order.Order.Taker.Hex() + "\"," +
		"\"takerAmount\":\"" + decimalString(order.Order.TakerAmount) + "\"," +
		"\"tokenId\":\"" + u256String(order.Order.TokenID) + "\"" +
		"}", nil
}

func u256String(value types.U256) string {
	if value.Int == nil {
		return "0"
	}
	return value.Int.String()
}

func decimalString(value types.Decimal) string {
	return value.String()
}

func saltToJSON(value types.U256) (interface{}, error) {
	if value.Int == nil {
		return uint64(0), nil
	}
	if value.Int.Sign() < 0 {
		return nil, fmt.Errorf("salt must be non-negative")
	}
	if value.Int.BitLen() > 53 {
		return nil, fmt.Errorf("salt is too large (max 53 bits)")
	}
	return value.Int.Uint64(), nil
}

func normalizeOrderType(orderType clobtypes.OrderType, fallback clobtypes.OrderType) clobtypes.OrderType {
	trimmed := strings.TrimSpace(string(orderType))
	if trimmed == "" {
		return fallback
	}
	upper := strings.ToUpper(trimmed)
	return clobtypes.OrderType(upper)
}
