# Примеры Cosmos-сообщений транзакций

Запуск из модуля `cosmos`:

```sh
go run ./examples/messages
go run ./examples/messages -json
go test ./examples/messages
```

Программа вызывает все 43 именованных конструктора из
`examples/messages/fixtures.go`. Каждый заполняется публичным детерминированным
protobuf payload из `schema/validation-node-corpus.json`, проходит `ValidateBasic`,
`GetSigners`, упаковку в `Any` и кодирование отдельного `TxBody`.
Эти точные входные данные прошли локальное сравнение с валидаторами официальной
ноды зафиксированной версии. Подписей и отправок нет.

Ниже приведены **фикстуры статической валидации**, а не готовые бизнес-транзакции.
Замените адреса, суммы, идентификаторы, metadata, authorities, proofs и вложенные
сообщения своими проверенными данными. В частности, поле
`MsgCreateTransaction.content` содержит placeholder Any с `type_url` равным
`fixture`. Upstream stateless validation его допускает, но для исполнения нужен
настоящий поддерживаемый тип сообщения. Пустые поля check/proof отражают пробелы
upstream-валидации, а не действительные доказательства или подписи.

Payload записан в protobuf text format. Bytes используют protobuf escaping.
Строки целочисленных сумм содержат минимальные единицы, без float. Для native DEL
1 DEL равен 1000000000000000000 минимальных единиц; у других активов свои единицы.
Десятичные коэффициенты и ID не являются суммами токенов: сохраняйте типы схемы.
Точные wire bytes и encoded body доступны через `-json`. Текстовый вывод опускает
protobuf defaults и не заменяет схему; полный список полей доступен через
`m.Descriptor().Fields()`.

Для своего сценария вызовите указанный конструктор, загрузите публичный fixture
через `proto.Unmarshal`, затем измените поля dynamic message. После заполнения
вызовите `cosmos.ValidateBasic(m)`, `cosmos.GetSigners(m)`, `cosmos.Pack(m)` и
`cosmos.EncodeBody([]proto.Message{m}, "memo", timeoutHeight)`. Полный пример
декодирования и проверок находится в `examples/messages/main.go`.

Только 8 типов допускаются как самостоятельные native-транзакции ante-цепочкой
зафиксированной ноды. Остальные 35 - исторические типы или поддержка схем.
Это вывод по исходникам, а не обещание маршрута в текущей сети. Права, балансы,
nonce, oracle membership, proofs и исполнение проверяет нода. Положительная
валидация не даёт прав на административные операции. `GetSigners` возвращает
байты адресов; приложение должно связать их с разрешёнными ключами.

Подпись - отдельный этап: нужны chain ID, account number, sequence, fee/gas и
публичные ключи. `SignDirectTx` принимает явный порядок подписантов и локальный
callback подписи. Пример останавливается на неподписанном `TxBody`.

## Все 43 сообщения

### 1. `NewMsgCancelUpgrade`

Type URL: `/cosmos.upgrade.v1beta1.MsgCancelUpgrade`. native-маршрут по исходникам.

```textproto
authority: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
```

### 2. `NewMsgSoftwareUpgrade`

Type URL: `/cosmos.upgrade.v1beta1.MsgSoftwareUpgrade`. native-маршрут по исходникам.

```textproto
authority: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
plan: {
  name: "fixture"
  height: 1
  info: "fixture"
}
```

### 3. `NewMsgBurnCoin`

Type URL: `/decimal.coin.v1.MsgBurnCoin`. историческая поддержка/схема.

```textproto
sender: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
coin: {
  denom: "abc"
  amount: "1"
}
```

### 4. `NewMsgBuyCoin`

Type URL: `/decimal.coin.v1.MsgBuyCoin`. историческая поддержка/схема.

```textproto
sender: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
coin_to_buy: {
  denom: "abc"
  amount: "1"
}
max_coin_to_sell: {
  denom: "xyz"
  amount: "1"
}
```

### 5. `NewMsgCreateCoin`

Type URL: `/decimal.coin.v1.MsgCreateCoin`. историческая поддержка/схема.

```textproto
sender: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
denom: "abc"
title: "fixture"
crr: 50
initial_volume: "1000000000000000000"
initial_reserve: "1000000000000000000"
limit_volume: "100000000000000000000"
identity: "fixture"
min_volume: "1000000000000000000"
```

### 6. `NewMsgMultiSendCoin`

Type URL: `/decimal.coin.v1.MsgMultiSendCoin`. историческая поддержка/схема.

```textproto
sender: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
sends: {
  recipient: "d01qgpqyqszqgpqyqszqgpqyqszqgpqyqszpfgulq"
  coin: {
    denom: "abc"
    amount: "1"
  }
}
```

### 7. `NewMsgRedeemCheck`

Type URL: `/decimal.coin.v1.MsgRedeemCheck`. историческая поддержка/схема.

```textproto
sender: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
check: "fixture"
proof: "fixture"
```

### 8. `NewMsgSellAllCoin`

Type URL: `/decimal.coin.v1.MsgSellAllCoin`. историческая поддержка/схема.

```textproto
sender: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
coin_denom_to_sell: "abc"
min_coin_to_buy: {
  denom: "xyz"
  amount: "1"
}
```

### 9. `NewMsgSellCoin`

Type URL: `/decimal.coin.v1.MsgSellCoin`. историческая поддержка/схема.

```textproto
sender: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
coin_to_sell: {
  denom: "abc"
  amount: "1"
}
min_coin_to_buy: {
  denom: "xyz"
  amount: "1"
}
```

### 10. `NewMsgSendCoin`

Type URL: `/decimal.coin.v1.MsgSendCoin`. историческая поддержка/схема.

```textproto
sender: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
recipient: "d01qgpqyqszqgpqyqszqgpqyqszqgpqyqszpfgulq"
coin: {
  denom: "abc"
  amount: "1"
}
```

### 11. `NewMsgUpdateCoin`

Type URL: `/decimal.coin.v1.MsgUpdateCoin`. историческая поддержка/схема.

```textproto
sender: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
denom: "abc"
limit_volume: "100000000000000000000"
identity: "fixture"
min_volume: "1000000000000000000"
```

### 12. `NewMsgUpdateCoinPrices`

Type URL: `/decimal.fee.v1.MsgUpdateCoinPrices`. native-маршрут по исходникам.

```textproto
oracle: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
prices: {
  denom: "abc"
  quote: "abc"
  price: "1"
  updated_at: {
    seconds: 1
  }
}
```

### 13. `NewMsgReturnLegacy`

Type URL: `/decimal.legacy.v1.MsgReturnLegacy`. историческая поддержка/схема.

```textproto
sender: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
public_key: "\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00"
```

### 14. `NewMsgCreateTransaction`

Type URL: `/decimal.multisig.v1.MsgCreateTransaction`. историческая поддержка/схема.

```textproto
sender: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
wallet: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
content: {
  type_url: "fixture"
  value: "\x00"
}
```

### 15. `NewMsgCreateWallet`

Type URL: `/decimal.multisig.v1.MsgCreateWallet`. историческая поддержка/схема.

```textproto
sender: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
owners: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
owners: "d01qgpqyqszqgpqyqszqgpqyqszqgpqyqszpfgulq"
weights: 1
weights: 2
threshold: 1
```

### 16. `NewMsgSignTransaction`

Type URL: `/decimal.multisig.v1.MsgSignTransaction`. историческая поддержка/схема.

```textproto
sender: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
id: "d0mstx1qyxd5h6p"
```

### 17. `NewMsgBurnToken`

Type URL: `/decimal.nft.v1.MsgBurnToken`. историческая поддержка/схема.

```textproto
sender: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
token_id: "fixture"
sub_token_ids: 1
sub_token_ids: 2
```

### 18. `NewMsgMintToken`

Type URL: `/decimal.nft.v1.MsgMintToken`. историческая поддержка/схема.

```textproto
sender: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
denom: "abc"
token_id: "fixture"
token_uri: "fixture"
recipient: "d01qgpqyqszqgpqyqszqgpqyqszqgpqyqszpfgulq"
quantity: 1
reserve: {
  denom: "abc"
  amount: "1"
}
```

### 19. `NewMsgSendToken`

Type URL: `/decimal.nft.v1.MsgSendToken`. историческая поддержка/схема.

```textproto
sender: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
token_id: "fixture"
sub_token_ids: 1
sub_token_ids: 2
recipient: "d01qgpqyqszqgpqyqszqgpqyqszqgpqyqszpfgulq"
```

### 20. `NewMsgUpdateReserve`

Type URL: `/decimal.nft.v1.MsgUpdateReserve`. историческая поддержка/схема.

```textproto
sender: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
token_id: "fixture"
sub_token_ids: 1
sub_token_ids: 2
reserve: {
  denom: "abc"
  amount: "1"
}
```

### 21. `NewMsgUpdateToken`

Type URL: `/decimal.nft.v1.MsgUpdateToken`. историческая поддержка/схема.

```textproto
sender: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
token_id: "fixture"
token_uri: "fixture"
```

### 22. `NewMsgActivateChain`

Type URL: `/decimal.swap.v1.MsgActivateChain`. историческая поддержка/схема.

```textproto
sender: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
id: 1
name: "fixture"
```

### 23. `NewMsgDeactivateChain`

Type URL: `/decimal.swap.v1.MsgDeactivateChain`. историческая поддержка/схема.

```textproto
sender: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
id: 1
```

### 24. `NewMsgInitializeSwap`

Type URL: `/decimal.swap.v1.MsgInitializeSwap`. историческая поддержка/схема.

```textproto
sender: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
recipient: "d01qgpqyqszqgpqyqszqgpqyqszqgpqyqszpfgulq"
amount: "1"
token_symbol: "fixture"
transaction_number: "1"
from_chain: 1
dest_chain: 2
```

### 25. `NewMsgRedeemSwap`

Type URL: `/decimal.swap.v1.MsgRedeemSwap`. историческая поддержка/схема.

```textproto
sender: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
from: "fixture"
recipient: "d01qgpqyqszqgpqyqszqgpqyqszqgpqyqszpfgulq"
amount: "1"
token_symbol: "fixture"
transaction_number: "1"
from_chain: 1
dest_chain: 2
v: 1
r: "0000000000000000000000000000000000000000000000000000000000000000"
s: "0000000000000000000000000000000000000000000000000000000000000000"
```

### 26. `NewMsgCancelRedelegation`

Type URL: `/decimal.validator.v1.MsgCancelRedelegation`. историческая поддержка/схема.

```textproto
delegator: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
validator_src: "d0valoper1qyqszqgpqyqszqgpqyqszqgpqyqszqgpseaenx"
validator_dst: "d0valoper1qgpqyqszqgpqyqszqgpqyqszqgpqyqszpamucs"
creation_height: 1
coin: {
  denom: "abc"
  amount: "1"
}
```

### 27. `NewMsgCancelRedelegationNFT`

Type URL: `/decimal.validator.v1.MsgCancelRedelegationNFT`. историческая поддержка/схема.

```textproto
delegator: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
validator_src: "d0valoper1qyqszqgpqyqszqgpqyqszqgpqyqszqgpseaenx"
validator_dst: "d0valoper1qgpqyqszqgpqyqszqgpqyqszqgpqyqszpamucs"
creation_height: 1
token_id: "fixture"
sub_token_ids: 1
sub_token_ids: 2
```

### 28. `NewMsgCancelUndelegation`

Type URL: `/decimal.validator.v1.MsgCancelUndelegation`. историческая поддержка/схема.

```textproto
delegator: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
validator: "d0valoper1qyqszqgpqyqszqgpqyqszqgpqyqszqgpseaenx"
creation_height: 1
coin: {
  denom: "abc"
  amount: "1"
}
```

### 29. `NewMsgCancelUndelegationNFT`

Type URL: `/decimal.validator.v1.MsgCancelUndelegationNFT`. историческая поддержка/схема.

```textproto
delegator: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
validator: "d0valoper1qyqszqgpqyqszqgpqyqszqgpqyqszqgpseaenx"
creation_height: 1
token_id: "fixture"
sub_token_ids: 1
sub_token_ids: 2
```

### 30. `NewMsgCreateValidator`

Type URL: `/decimal.validator.v1.MsgCreateValidator`. историческая поддержка/схема.

```textproto
operator_address: "d0valoper1qyqszqgpqyqszqgpqyqszqgpqyqszqgpseaenx"
reward_address: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
consensus_pubkey: {
  type_url: "fixture"
  value: "\x00"
}
description: {
  moniker: "fixture"
  identity: "fixture"
  website: "fixture"
  security_contact: "fixture"
  details: "fixture"
}
commission: "1"
stake: {
  denom: "abc"
  amount: "1"
}
```

### 31. `NewMsgDelegate`

Type URL: `/decimal.validator.v1.MsgDelegate`. историческая поддержка/схема.

```textproto
delegator: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
validator: "d0valoper1qyqszqgpqyqszqgpqyqszqgpqyqszqgpseaenx"
coin: {
  denom: "abc"
  amount: "1"
}
```

### 32. `NewMsgDelegateNFT`

Type URL: `/decimal.validator.v1.MsgDelegateNFT`. историческая поддержка/схема.

```textproto
delegator: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
validator: "d0valoper1qyqszqgpqyqszqgpqyqszqgpqyqszqgpseaenx"
token_id: "fixture"
sub_token_ids: 1
sub_token_ids: 2
```

### 33. `NewMsgEditValidator`

Type URL: `/decimal.validator.v1.MsgEditValidator`. историческая поддержка/схема.

```textproto
operator_address: "d0valoper1qyqszqgpqyqszqgpqyqszqgpqyqszqgpseaenx"
reward_address: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
description: {
  moniker: "fixture"
  identity: "fixture"
  website: "fixture"
  security_contact: "fixture"
  details: "fixture"
}
```

### 34. `NewMsgFreezeChain`

Type URL: `/decimal.validator.v1.MsgFreezeChain`. native-маршрут по исходникам.

```textproto
sender: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
reason: "fixture"
```

### 35. `NewMsgHaltChain`

Type URL: `/decimal.validator.v1.MsgHaltChain`. native-маршрут по исходникам.

```textproto
sender: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
height: 1
reason: "fixture"
```

### 36. `NewMsgRedelegate`

Type URL: `/decimal.validator.v1.MsgRedelegate`. историческая поддержка/схема.

```textproto
delegator: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
validator_src: "d0valoper1qyqszqgpqyqszqgpqyqszqgpqyqszqgpseaenx"
validator_dst: "d0valoper1qgpqyqszqgpqyqszqgpqyqszqgpqyqszpamucs"
coin: {
  denom: "abc"
  amount: "1"
}
```

### 37. `NewMsgRedelegateNFT`

Type URL: `/decimal.validator.v1.MsgRedelegateNFT`. историческая поддержка/схема.

```textproto
delegator: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
validator_src: "d0valoper1qyqszqgpqyqszqgpqyqszqgpqyqszqgpseaenx"
validator_dst: "d0valoper1qgpqyqszqgpqyqszqgpqyqszqgpqyqszpamucs"
token_id: "fixture"
sub_token_ids: 1
sub_token_ids: 2
```

### 38. `NewMsgResumeChain`

Type URL: `/decimal.validator.v1.MsgResumeChain`. native-маршрут по исходникам.

```textproto
sender: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
```

### 39. `NewMsgSetOffline`

Type URL: `/decimal.validator.v1.MsgSetOffline`. историческая поддержка/схема.

```textproto
validator: "d0valoper1qyqszqgpqyqszqgpqyqszqgpqyqszqgpseaenx"
```

### 40. `NewMsgSetOnline`

Type URL: `/decimal.validator.v1.MsgSetOnline`. native-маршрут по исходникам.

```textproto
validator: "d0valoper1qyqszqgpqyqszqgpqyqszqgpqyqszqgpseaenx"
```

### 41. `NewMsgUndelegate`

Type URL: `/decimal.validator.v1.MsgUndelegate`. историческая поддержка/схема.

```textproto
delegator: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
validator: "d0valoper1qyqszqgpqyqszqgpqyqszqgpqyqszqgpseaenx"
coin: {
  denom: "abc"
  amount: "1"
}
```

### 42. `NewMsgUndelegateNFT`

Type URL: `/decimal.validator.v1.MsgUndelegateNFT`. историческая поддержка/схема.

```textproto
delegator: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
validator: "d0valoper1qyqszqgpqyqszqgpqyqszqgpqyqszqgpseaenx"
token_id: "fixture"
sub_token_ids: 1
sub_token_ids: 2
```

### 43. `NewMsgUnfreezeChain`

Type URL: `/decimal.validator.v1.MsgUnfreezeChain`. native-маршрут по исходникам.

```textproto
sender: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
```

