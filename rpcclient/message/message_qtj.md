介绍

    该文档规范对KTO的rpc接口客户端的操作，对相关参数、响应定义。所有字段均以文档为准，操作失败时接口返回的响应为nil。
    测试环境的rpc地址为: 106.13.188.227:8545
# 0. SignOrd
**对订单数据签名**
- 接口定义

```rpc
      rpc SignOrd(req_sign_ord) returns(resp_sign_ord){}
```
- order字段
  
    序号|字段|类型|描述
    :-:|:--|--|:--
    1|Id|string|趣淘鲸订单id
    2|Address|string|趣淘鲸地址
    3|Price|uint64|价格
    4|Hash|string|hash值
    5|Signature|string|签名
    6|Ciphertext|string|证书
    7|Tradename|string|商品名称
    8|Region|string|区域

- 请求参数 req_sign_ord
    序号|字段|类型|描述
    :-:|:--|--|:--
    1|priv|string|趣淘鲸私钥
    2|Order|order|趣淘鲸订单数据

- 返回参数 resp_sign_ord
    序号|字段|类型|描述
    :-:|:--|--|:--
    1|hash|string|趣淘鲸订单数据的hash
    2|signature|string|趣淘鲸订单的签名

# 1. SendTransaction
**向kto发送交易**
- 接口定义

```rpc
    rpc SendTransaction(req_transaction) returns (res_transaction) {}
```
- order字段
  
    序号|字段|类型|描述
    :-:|:--|--|:--
    1|Id|string|趣淘鲸订单id
    2|Address|string|趣淘鲸地址
    3|Price|uint64|价格
    4|Hash|string|hash值
    5|Signature|string|签名
    6|Ciphertext|strig|证书

- 请求参数 req_transaction
  
    序号|字段|类型|描述
    :-:|:--|:--|:--
    1|From|string|交易发送方地址
    2|To|string|交易接收方地址
    3|Amount|uint64|交易金额
    4|Nonce|uint64|递增的随机数
    5|Priv|string|私钥
    6|Order|order|趣淘鲸订单

   

- 响应参数 res_transaction

    序号|字段|字段|描述
    :--:|:--|:--|:---
    1|Hash|string|交易的哈希值

# 2.GetAddressNonceAt
**通过地址获得该地址的需要的nonce**

- 接口定义

```rpc
    rpc GetAddressNonceAt(req_nonce) returns (respose_nonce) {}
```

- 请求参数 req_nonce

    序号|字段|类型|描述
    :-:|:--|:-|:--
    1|address|string|地址值

- 相应参数 respose_nonce

    序号|字段|类型|描述
    :-:|:--|:-|:--
    1|nonce|uint64|下一笔交易需要的nonce


# 3.GetTxByHash
**通过哈希值获取交易信息**
- 接口定义

```rpc
    rpc GetTxByHash(req_tx_by_hash) returns (Tx) {}
```

- 请求参数　req_tx_by_hash

    序号|字段|类型|描述
    :-:|:--|:-|:--
    1|hash|string|交易的哈希值

- 响应参数 　Tx

    序号|字段|类型|描述
    :-:|:--|:-|:--
    1|Nonce|uint64|该交易的nonce
    2|Amount|uint64|交易金额
    3|From|string|交易发起方地址
    4|To|string|交易接收方地址
    5|Hash|string|该交易的hash
    6|Signature|string|交易的签名
    7|Time|int64|时间戳，单位为秒

# 4.GetTxsByAddr
**通过地址获取该地址的所有交易**
- 接口定义

```rpc
    rpc GetTxsByAddr(req_tx) returns (respose_txs) {}
```

- 请求参数　req_tx

    序号|字段|类型|描述
    :-:|:--|:-|:--
    1|address|string|地址

- 响应参数　res_tx

    序号|字段|类型|描述
    :-:|:--|:-|:--
    1|Txs|[]Tx|交易信息集合(Tx的字段请看GetTxByHash接口的响应参数)


# 5.GetBalance
**通过地址获取改地址的余额**
- 接口定义

```rpc
    rpc GetBalance(req_balance) returns (res_balance) {}
```

- 请求参数　req_balance

    序号|字段|类型|描述
    :-:|:--|:-|:--
    1|address|string|地址

- 响应参数　res_balance

    序号|字段|类型|描述
    :-:|:--|:-|:--
    1|balnce|uint64|该地址的余额


# 6.CreateAddr
**创建地址和私钥**
- 接口定义

```rpc
    rpc CreateAddr(req_create_addr) returns (resp_create_addr) {}
```

- 请求参数　req_create_addr
  字段为空

- 响应参数　resp_create_addr

    序号|字段|类型|描述
    :-:|:--|:-|:--
    1|address|string|地址
    2|privkey|string|私钥

# 7.GetAddrByPriv
**通过私钥获取地址**
- 接口定义

```rpc
    rpc GetAddrByPriv(req_addr_by_priv) returns (resp_addr_by_priv) {}
```

- 请求参数　req_addr_by_priv

    序号|字段|类型|描述
    :-:|:--|:-|:--
    1|priv|string|私钥

- 响应参数　resp_addr_by_priv

    序号|字段|类型|描述
    :-:|:--|:-|:--
    1|addr|string|地址

# 8.GetBlockByNum
**通过块号获取区块的信息**
- 接口定义

```rpc
    rpc GetBlockByNum(req_block_by_number) returns (resp_block) {}
```

- 请求参数 req_block_by_number
    序号|字段|类型|描述
    :-:|:--|:-|:--
    1|height|uint64|区块号

- 相应参数 resp_block
    序号|字段|类型|描述
    :-:|:--|:-|:--
    1|Height|uint64|区块号
    2|PrevBlockHash|string|上一个区块的哈希值
    3|Txs|[]Tx|交易信息集合(Tx的字段请看GetTxByHash接口的响应参数)
    4|Root|string|根
    5|Version|uint64|版本号
    6|Timestamp|int64|该区块的时间戳,单位为秒
    7|Hash|string|该区块的哈希
    8|Miner|string|矿工地址