package server

var ErrorMap = map[int]string{
	0:      "ok",
	-41201: "Json解析失败",
	-41202: "该nonce不存在",
	-41203: "没有此交易",
	-41204: "此hash没有交易",
	-41205: "数据错误",
	-41206: "没有块数据",
	-41207: "交易失败",
	-41208: "获取块高失败",
}

var (
	Success          = 0
	ErrJSON          = -41201
	ErrNoNonce       = -41202
	ErrNoTransaction = -41203
	ErrNoTxByHash    = -41204
	ErrData          = -41205
	ErrNoBlock       = -41206
	ErrtTx           = -41207
	ErrNoBlockHeight = -41208
)
