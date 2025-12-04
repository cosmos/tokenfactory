# tokenfactory CosmWasm Contract Example

This demo contract provides a 1:1 mapping to the tokenfactory bindings.

The contract messages only do some input validation and directly call into their respective bindings outlined in the "Messages" section below.

There are unit tests added to demonstrate how contract developers might utilize `token-bindings-test` package to import and use some test utilities.

## Messages

There are 4 messages:
- `ExecuteMsg::CreateDenom` maps to `TokenFactoryMsg::CreateDenom`
- `ExecuteMsg::ChangeAdmin` maps to `TokenFactoryMsg::ChangeAdmin`
- `ExecuteMsg::BurnTokens` maps to `TokenFactoryMsg::Burn`
- `ExecuteMsg::MintTokens` maps to `TokenFactoryMsg::MintTokens`

## Query

1 query:
- `QueryMsg::GetDenom` maps to `TokenFactoryQuery::FullDenom`

## Running with local `tokend`

### Run `tokend` node

Run the commands below from the repo root.

```sh
scripts/test_node.sh
```

### Build Contract

#### Option 1: Compile with `cargo wasm`

```sh
cd wasm-demo/contracts/tokenfactory
rustup default stable
cargo wasm
```

The resulting contract will be found in `wasm-demo/contracts/tokenfactory/target/wasm32-unknown-unknown/tokenfactory.wasm`.

#### Option 2: Optimized Compilation

```sh
cd wasm
docker run --rm -v "$(pwd)":/code \
  --mount type=volume,source="$(basename "$(pwd)")_cache",target=/target \
  --mount type=volume,source=registry_cache,target=/usr/local/cargo/registry \
  cosmwasm/workspace-optimizer:0.17.0
```

The resulting contract will be found in `wasm-demo/artifacts/tokenfactory.wasm`

### Upload Contract to Local `tokend` Chain

```sh
cd wasm-demo/artifacts
# Upload and store transaction hash in TX environment variable.
TX=$(tokend tx wasm store wasm-demo/artifacts/tokenfactory.wasm  --from user1 --gas-prices 0.1utoken --gas auto --gas-adjustment 2 --output json -y | jq -r '.txhash')
CODE_ID=$(tokend q tx $TX --output json | jq -r '.events[] | select(.type=="store_code").attributes[] | select(.key=="code_id").value')
echo "Your contract code_id is $CODE_ID"
```

### Instantiate the Contact
```sh
# Instantiate
tokend tx wasm instantiate $CODE_ID "{}" --amount 100000000utoken  --label "Token Factory Contract" --from user1 --gas-prices 0.1utoken --gas auto --gas-adjustment 2 -y --no-admin

# Get contract address
CONTRACT_ADDR=$(tokend query wasm list-contract-by-code $CODE_ID --output json | jq -r '.contracts[0]')
echo "Your contract address is $CONTRACT_ADDR"
```

### Execute & Queries

You can generate the schema to assist you with determining the structure for each CLI query:

```sh
cd wasm-demo/contracts/tokenfactory
cargo schema # generates schema in the contracts/tokenfactory/schema folder
```

For example, here is the schema for `CreateDenom` message:

```json
{
      "type": "object",
      "required": [
        "create_denom"
      ],
      "properties": {
        "create_denom": {
          "type": "object",
          "required": [
            "subdenom"
          ],
          "properties": {
            "subdenom": {
              "type": "string"
            }
          }
        }
      },
      "additionalProperties": false
    }
```

##### Messages

- `Create Denom`
```sh
tokend tx wasm execute $CONTRACT_ADDR '{ "create_denom": { "subdenom": "mydenom" } }' --from user1 --amount 10000000utoken --gas 1000000 --gas-prices 0.005utoken -y

# If you do this
tokend q tokenfactory denoms-from-admin $CONTRACT_ADDR
# You should see this:
# denoms:
# - factory/factory/cosmos14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9s4hmalr/mydenom
```

- `Mint Tokens` executing from user1, minting to user2
```sh
USER1_ADDR=cosmos1hj5fveer5cjtn4wd6wstzugjfdxzl0xpxvjjvr # from scripts/test_node.sh
USER2_ADDR=cosmos1efd63aw40lxf3n4mhf7dzhjkr453axur6cpk92 # from scripts/test_node.sh

tokend tx wasm execute $CONTRACT_ADDR "{ \"mint_tokens\": {\"amount\": \"100\", \"denom\": \"factory/${CONTRACT_ADDR}/mydenom\", \"mint_to_address\": \"$USER2_ADDR\"}}" --from user1 --gas auto --gas-adjustment 2 --gas-prices 0.005utoken -y

# If you do this
tokend q bank total-supply-of factory/$CONTRACT_ADDR/mydenom
# You should see this in the list:
# - amount: "100"
#   denom: factory/osmo14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9sq2r9
# You can also query the new balance of USER2_ADDR
tokend q bank balances $USER2_ADDR
# You should see this in the list:
# balances:
# - amount: "100"
#   denom: factory/cosmos14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9s4hmalr/mydenom
```

- `Burn Tokens` executing from test1, burning from test2

```sh
tokend tx wasm execute $CONTRACT_ADDR "{ \"burn_tokens\": {\"amount\": \"50\", \"denom\": \"factory/${CONTRACT_ADDR}/mydenom\", \"burn_from_address\": \"$USER2_ADDR\"}}" --from user1 --gas auto --gas-adjustment 2 --gas-prices 0.005utoken -y

# If you do this
tokend q bank total-supply-of factory/$CONTRACT_ADDR/mydenom
# You should see this in the list:
# - amount: "50"
#   denom: factory/osmo14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9sq2r9
# You can also query the new balance of USER2_ADDR
tokend q bank balances $USER2_ADDR
# You should see this in the list:
# balances:
# - amount: "50"
#   denom: factory/cosmos14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9s4hmalr/mydenom
```

- `Force Transfer` executing from user1, transferring from user2 to user1

```sh
USER1_ADDR=cosmos1phaxpevm5wecex2jyaqty2a4v02qj7qmlmzk5a # from scripts/test_node.sh

tokend tx wasm execute $CONTRACT_ADDR "{ \"force_transfer\": {\"amount\": \"25\", \"denom\": \"factory/${CONTRACT_ADDR}/mydenom\", \"from_address\": \"$USER2_ADDR\", \"to_address\": \"$USER1_ADDR\"}}" --from user1 --gas auto --gas-adjustment 2 --gas-prices 0.005utoken -y

# If you do this
tokend q bank balances $USER2_ADDR
# You should see user2's balance reduced:
# balances:
# - amount: "25"
#   denom: factory/cosmos14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9s4hmalr/mydenom
# And query user1's balance
tokend q bank balances $USER1_ADDR
# You should see user1 now has the transferred tokens:
# balances:
# - amount: "25"
#   denom: factory/cosmos14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9s4hmalr/mydenom
```

- `Change Admin` executing from user1, changing from `$CONTRACT_ADDR` to $USER2_ADDR

```sh
# Change Admin
tokend tx wasm execute $CONTRACT_ADDR "{ \"change_admin\": {\"denom\": \"factory/${CONTRACT_ADDR}/mydenom\", \"new_admin_address\": \"${USER2_ADDR}\"}}" --from user1 --gas auto --gas-adjustment 2 --gas-prices 0.005utoken -y

# Verify New Admin
tokend q tokenfactory denom-authority-metadata factory/${CONTRACT_ADDR}/mydenom
# You should see this:
# authority_metadata:
#   admin: cosmos1efd63aw40lxf3n4mhf7dzhjkr453axur6cpk92
```

##### Queries

- `Get Denom`
```sh
tokend query wasm contract-state smart $CONTRACT_ADDR "{ \"get_denom\": {\"creator_address\": \"${CONTRACT_ADDR}\", \"subdenom\": \"mydenom\" }}"
```
