#!/bin/bash
# Run this script after starting a local chain with test_node.sh
# This will demonstrate storing, instantiating, executing, and querying a tokenfactory CosmWasm contract.

USER1_ADDR=cosmos1hj5fveer5cjtn4wd6wstzugjfdxzl0xpxvjjvr # from scripts/test_node.sh
USER2_ADDR=cosmos1efd63aw40lxf3n4mhf7dzhjkr453axur6cpk92 # from scripts/test_node.sh

echo "> Upload contract and store transaction hash in TX environment variable."
TX=$(tokend tx wasm store x/tokenfactory/bindings/testdata/tokenfactory.wasm  --from $USER1_ADDR --gas-prices 0.1utoken --gas auto --gas-adjustment 2 --output json -y | jq -r '.txhash')
sleep 6
CODE_ID=$(tokend q tx $TX --output json | jq -r '.events[] | select(.type=="store_code").attributes[] | select(.key=="code_id").value')
echo "The contract code_id is $CODE_ID"

echo "> Instantiate contract"
tokend tx wasm instantiate $CODE_ID "{}" --amount 100000000utoken  --label "Token Factory Contract" --from $USER1_ADDR --gas-prices 0.1utoken --gas auto --gas-adjustment 2 -y --no-admin
sleep 6

echo "> Get contract address"
CONTRACT_ADDR=$(tokend query wasm list-contract-by-code $CODE_ID --output json | jq -r '.contracts[0]')
echo "The contract address is $CONTRACT_ADDR"

echo "> Create denom"
tokend tx wasm execute $CONTRACT_ADDR '{ "create_denom": { "subdenom": "mydenom" } }' --from $USER1_ADDR --amount 10000000utoken --gas 1000000 --gas-prices 0.005utoken -y
sleep 6
tokend q tokenfactory denoms-from-admin $CONTRACT_ADDR

echo "> Mint tokens executing from USER1, minting to USER2"
tokend tx wasm execute $CONTRACT_ADDR "{ \"mint_tokens\": {\"amount\": \"100\", \"denom\": \"factory/${CONTRACT_ADDR}/mydenom\", \"mint_to_address\": \"$USER2_ADDR\"}}" --from $USER1_ADDR --gas auto --gas-adjustment 2 --gas-prices 0.005utoken -y
sleep 6
tokend q bank total-supply-of factory/$CONTRACT_ADDR/mydenom
tokend q bank balances $USER2_ADDR

echo "> Burn tokens executing from USER1, burning from USER2"
tokend tx wasm execute $CONTRACT_ADDR "{ \"burn_tokens\": {\"amount\": \"50\", \"denom\": \"factory/${CONTRACT_ADDR}/mydenom\", \"burn_from_address\": \"$USER2_ADDR\"}}" --from $USER1_ADDR --gas auto --gas-adjustment 2 --gas-prices 0.005utoken -y
sleep 6
tokend q bank total-supply-of factory/$CONTRACT_ADDR/mydenom
tokend q bank balances $USER2_ADDR

echo "> Force transfer tokens executing from USER1, transferring from USER2 to USER1"
tokend tx wasm execute $CONTRACT_ADDR "{ \"force_transfer\": {\"amount\": \"25\", \"denom\": \"factory/${CONTRACT_ADDR}/mydenom\", \"from_address\": \"$USER2_ADDR\", \"to_address\": \"$USER1_ADDR\"}}" --from $USER1_ADDR --gas auto --gas-adjustment 2 --gas-prices 0.005utoken -y
sleep 6
tokend q bank balances $USER2_ADDR
tokend q bank balances $USER1_ADDR

echo "> Query denom info from contract"
tokend query wasm contract-state smart $CONTRACT_ADDR "{ \"get_denom\": {\"creator_address\": \"${CONTRACT_ADDR}\", \"subdenom\": \"mydenom\" }}"