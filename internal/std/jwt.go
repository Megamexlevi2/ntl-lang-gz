package std

import (
	"fmt"
	"lunex/internal/runtime"
	shared "lunex/internal/std/shared"
	"time"

	gojwt "github.com/golang-jwt/jwt/v5"
)

func JWTModule() *runtime.Value {
	sign := runtime.FuncVal(&runtime.Function{
		Name: "sign",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.Null, fmt.Errorf("sign(payload, secret, options?)")
			}
			if args[0].Tag != runtime.TypeObject {
				return runtime.Null, fmt.Errorf("payload must be an object")
			}
			secret := args[0].ToString()
			if len(args) >= 2 {
				secret = args[1].ToString()
			}
			algorithm := "HS256"
			expiresIn := int64(3600)
			var issuer, audience, subject string

			if len(args) > 2 && args[2].Tag == runtime.TypeObject {
				opts := args[2].ObjVal
				if v, ok := opts["algorithm"]; ok {
					algorithm = v.ToString()
				}
				if v, ok := opts["expiresIn"]; ok && v.Tag == runtime.TypeNumber {
					expiresIn = int64(v.NumVal)
				}
				if v, ok := opts["issuer"]; ok {
					issuer = v.ToString()
				}
				if v, ok := opts["audience"]; ok {
					audience = v.ToString()
				}
				if v, ok := opts["subject"]; ok {
					subject = v.ToString()
				}
			}

			claims := gojwt.MapClaims{}
			for k, v := range args[0].ObjVal {
				switch v.Tag {
				case runtime.TypeString:
					claims[k] = v.StrVal
				case runtime.TypeNumber:
					claims[k] = v.NumVal
				case runtime.TypeBool:
					claims[k] = v.BoolVal
				case runtime.TypeNull, runtime.TypeUndefined:
					claims[k] = nil
				}
			}
			now := time.Now()
			claims["iat"] = now.Unix()
			claims["exp"] = now.Unix() + expiresIn
			if issuer != "" {
				claims["iss"] = issuer
			}
			if audience != "" {
				claims["aud"] = audience
			}
			if subject != "" {
				claims["sub"] = subject
			}

			var method gojwt.SigningMethod
			switch algorithm {
			case "HS384":
				method = gojwt.SigningMethodHS384
			case "HS512":
				method = gojwt.SigningMethodHS512
			default:
				method = gojwt.SigningMethodHS256
			}

			token := gojwt.NewWithClaims(method, claims)
			signed, err := token.SignedString([]byte(secret))
			if err != nil {
				return runtime.Null, err
			}
			return runtime.StringVal(signed), nil
		},
	})

	verify := runtime.FuncVal(&runtime.Function{
		Name: "verify",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.Null, fmt.Errorf("verify(token, secret)")
			}
			tokenStr := args[0].ToString()
			secret := args[1].ToString()

			token, err := gojwt.Parse(tokenStr, func(t *gojwt.Token) (interface{}, error) {
				if _, ok := t.Method.(*gojwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
				}
				return []byte(secret), nil
			})
			if err != nil {
				return runtime.ObjectVal(map[string]*runtime.Value{
					"valid": runtime.False,
					"error": runtime.StringVal(err.Error()),
				}), nil
			}
			claims, ok := token.Claims.(gojwt.MapClaims)
			if !ok || !token.Valid {
				return runtime.ObjectVal(map[string]*runtime.Value{
					"valid": runtime.False,
					"error": runtime.StringVal("invalid token"),
				}), nil
			}
			payload := make(map[string]*runtime.Value)
			for k, v := range claims {
				payload[k] = shared.JsonToValue(v)
			}
			return runtime.ObjectVal(map[string]*runtime.Value{
				"valid":   runtime.True,
				"payload": runtime.ObjectVal(payload),
			}), nil
		},
	})

	decode := runtime.FuncVal(&runtime.Function{
		Name: "decode",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.Null, fmt.Errorf("decode(token)")
			}
			token, _, err := new(gojwt.Parser).ParseUnverified(args[0].ToString(), gojwt.MapClaims{})
			if err != nil {
				return runtime.Null, err
			}
			claims, ok := token.Claims.(gojwt.MapClaims)
			if !ok {
				return runtime.Null, fmt.Errorf("cannot parse claims")
			}
			payload := make(map[string]*runtime.Value)
			for k, v := range claims {
				payload[k] = shared.JsonToValue(v)
			}
			header := make(map[string]*runtime.Value)
			for k, v := range token.Header {
				header[k] = shared.JsonToValue(v)
			}
			return runtime.ObjectVal(map[string]*runtime.Value{
				"header":  runtime.ObjectVal(header),
				"payload": runtime.ObjectVal(payload),
			}), nil
		},
	})

	isExpired := runtime.FuncVal(&runtime.Function{
		Name: "isExpired",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.True, nil
			}
			token, _, err := new(gojwt.Parser).ParseUnverified(args[0].ToString(), gojwt.MapClaims{})
			if err != nil {
				return runtime.True, nil
			}
			claims, ok := token.Claims.(gojwt.MapClaims)
			if !ok {
				return runtime.True, nil
			}
			exp, err := claims.GetExpirationTime()
			if err != nil || exp == nil {
				return runtime.False, nil
			}
			return runtime.BoolVal(time.Now().After(exp.Time)), nil
		},
	})

	refresh := runtime.FuncVal(&runtime.Function{
		Name: "refresh",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.Null, fmt.Errorf("refresh(token, secret, expiresIn?)")
			}
			tokenStr := args[0].ToString()
			secret := args[1].ToString()
			expiresIn := int64(3600)
			if len(args) > 2 && args[2].Tag == runtime.TypeNumber {
				expiresIn = int64(args[2].NumVal)
			}
			token, err := gojwt.Parse(tokenStr, func(t *gojwt.Token) (interface{}, error) {
				return []byte(secret), nil
			})
			if err != nil {
				return runtime.Null, err
			}
			claims, ok := token.Claims.(gojwt.MapClaims)
			if !ok {
				return runtime.Null, fmt.Errorf("invalid token")
			}
			now := time.Now()
			claims["iat"] = now.Unix()
			claims["exp"] = now.Unix() + expiresIn
			newToken := gojwt.NewWithClaims(gojwt.SigningMethodHS256, claims)
			signed, err := newToken.SignedString([]byte(secret))
			if err != nil {
				return runtime.Null, err
			}
			return runtime.StringVal(signed), nil
		},
	})

	return runtime.ObjectVal(map[string]*runtime.Value{
		"sign":      sign,
		"verify":    verify,
		"decode":    decode,
		"isExpired": isExpired,
		"refresh":   refresh,
	})
}
