use cosmwasm_schema::{cw_serde, QueryResponses};
use cosmwasm_std::Uint128;

#[cw_serde]
pub struct InstantiateMsg {}

#[cw_serde]
pub enum ExecuteMsg {
    CreateDenom {
        subdenom: String,
    },
    ChangeAdmin {
        denom: String,
        new_admin_address: String,
    },
    MintTokens {
        denom: String,
        amount: Uint128,
        /// Optional recipient address. If not provided, mints to the contract address.
        mint_to_address: Option<String>,
    },
    BurnTokens {
        denom: String,
        amount: Uint128,
        /// Optional address to burn from. If not provided, burns from the contract address.
        burn_from_address: Option<String>,
    },
    ForceTransfer {
        denom: String,
        amount: Uint128,
        from_address: String,
        to_address: String,
    },
}

#[cw_serde]
#[derive(QueryResponses)]
pub enum QueryMsg {
    #[returns(GetDenomResponse)]
    GetDenom {
        creator_address: String,
        subdenom: String,
    },
}

// We define a custom struct for each query response
#[cw_serde]
pub struct GetDenomResponse {
    pub denom: String,
}
