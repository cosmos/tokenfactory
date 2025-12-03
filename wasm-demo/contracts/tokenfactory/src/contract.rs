#[cfg(not(feature = "library"))]
use cosmwasm_std::entry_point;
use cosmwasm_std::{
    to_json_binary, Binary, Deps, DepsMut, Env, MessageInfo, Response, StdResult, Uint128,
};
use cw2::set_contract_version;

use crate::error::TokenFactoryError;
use crate::msg::{ExecuteMsg, GetDenomResponse, InstantiateMsg, QueryMsg};
use crate::state::{State, STATE};
use token_bindings::{TokenFactoryMsg, TokenFactoryQuery, TokenQuerier};

// version info for migration info
const CONTRACT_NAME: &str = "crates.io:tokenfactory-demo";
const CONTRACT_VERSION: &str = env!("CARGO_PKG_VERSION");

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn instantiate(
    deps: DepsMut<TokenFactoryQuery>,
    _env: Env,
    info: MessageInfo,
    _msg: InstantiateMsg,
) -> Result<Response, TokenFactoryError> {
    let state = State {
        owner: info.sender.clone(),
    };
    set_contract_version(deps.storage, CONTRACT_NAME, CONTRACT_VERSION)?;
    STATE.save(deps.storage, &state)?;

    Ok(Response::new()
        .add_attribute("method", "instantiate")
        .add_attribute("owner", info.sender))
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn execute(
    deps: DepsMut<TokenFactoryQuery>,
    env: Env,
    _info: MessageInfo,
    msg: ExecuteMsg,
) -> Result<Response<TokenFactoryMsg>, TokenFactoryError> {
    match msg {
        ExecuteMsg::CreateDenom { subdenom } => create_denom(subdenom),
        ExecuteMsg::ChangeAdmin {
            denom,
            new_admin_address,
        } => change_admin(deps, denom, new_admin_address),
        ExecuteMsg::MintTokens {
            denom,
            amount,
            mint_to_address,
        } => mint_tokens(deps, env, denom, amount, mint_to_address),
        ExecuteMsg::BurnTokens {
            denom,
            amount,
            burn_from_address,
        } => burn_tokens(deps, env, denom, amount, burn_from_address),
        ExecuteMsg::ForceTransfer {
            denom,
            amount,
            from_address,
            to_address,
        } => force_transfer(deps, denom, amount, from_address, to_address),
    }
}

pub fn create_denom(subdenom: String) -> Result<Response<TokenFactoryMsg>, TokenFactoryError> {
    if subdenom.eq("") {
        return Err(TokenFactoryError::InvalidSubdenom { subdenom });
    }

    let create_denom_msg = TokenFactoryMsg::CreateDenom {
        subdenom,
        metadata: None,
    };

    let res = Response::new()
        .add_attribute("method", "create_denom")
        .add_message(create_denom_msg);

    Ok(res)
}

pub fn change_admin(
    deps: DepsMut<TokenFactoryQuery>,
    denom: String,
    new_admin_address: String,
) -> Result<Response<TokenFactoryMsg>, TokenFactoryError> {
    deps.api.addr_validate(&new_admin_address)?;

    validate_denom(deps, denom.clone())?;

    let change_admin_msg = TokenFactoryMsg::ChangeAdmin {
        denom,
        new_admin_address,
    };

    let res = Response::new()
        .add_attribute("method", "change_admin")
        .add_message(change_admin_msg);

    Ok(res)
}

pub fn mint_tokens(
    deps: DepsMut<TokenFactoryQuery>,
    env: Env,
    denom: String,
    amount: Uint128,
    mint_to_address: Option<String>,
) -> Result<Response<TokenFactoryMsg>, TokenFactoryError> {
    if amount.eq(&Uint128::new(0_u128)) {
        return Result::Err(TokenFactoryError::ZeroAmount {});
    }

    // Validate address first if provided (before consuming deps)
    if let Some(ref addr) = mint_to_address {
        deps.api.addr_validate(addr)?;
    }

    validate_denom(deps, denom.clone())?;

    // Default to contract address if mint_to_address is not provided
    let recipient = match mint_to_address {
        Some(addr) => addr,
        None => env.contract.address.to_string(),
    };

    let mint_tokens_msg = TokenFactoryMsg::mint_contract_tokens(denom, amount, recipient);

    let res = Response::new()
        .add_attribute("method", "mint_tokens")
        .add_message(mint_tokens_msg);

    Ok(res)
}

pub fn burn_tokens(
    deps: DepsMut<TokenFactoryQuery>,
    _env: Env,
    denom: String,
    amount: Uint128,
    burn_from_address: Option<String>,
) -> Result<Response<TokenFactoryMsg>, TokenFactoryError> {
    if amount.eq(&Uint128::new(0_u128)) {
        return Result::Err(TokenFactoryError::ZeroAmount {});
    }

    // Validate address first if provided (before consuming deps)
    if let Some(ref addr) = burn_from_address {
        deps.api.addr_validate(addr)?;
    }

    validate_denom(deps, denom.clone())?;

    // Create the appropriate burn message based on whether burn_from_address is provided
    let burn_token_msg = match burn_from_address {
        Some(addr) => {
            // Burn from that specific address (requires EnableBurnFrom)
            TokenFactoryMsg::burn_contract_tokens(denom, amount, addr)
        }
        None => {
            // Burn from contract's own balance (does not require EnableBurnFrom)
            TokenFactoryMsg::burn_contract_tokens_from_self(denom, amount)
        }
    };

    let res = Response::new()
        .add_attribute("method", "burn_tokens")
        .add_message(burn_token_msg);

    Ok(res)
}

pub fn force_transfer(
    deps: DepsMut<TokenFactoryQuery>,
    denom: String,
    amount: Uint128,
    from_address: String,
    to_address: String,
) -> Result<Response<TokenFactoryMsg>, TokenFactoryError> {
    if amount.eq(&Uint128::new(0_u128)) {
        return Result::Err(TokenFactoryError::ZeroAmount {});
    }

    validate_denom(deps, denom.clone())?;

    let force_msg = TokenFactoryMsg::force_transfer_tokens(denom, amount, from_address, to_address);

    let res = Response::new()
        .add_attribute("method", "force_transfer_tokens")
        .add_message(force_msg);

    Ok(res)
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn query(deps: Deps<TokenFactoryQuery>, _env: Env, msg: QueryMsg) -> StdResult<Binary> {
    match msg {
        QueryMsg::GetDenom {
            creator_address,
            subdenom,
        } => to_json_binary(&get_denom(deps, creator_address, subdenom)),
    }
}

fn get_denom(
    deps: Deps<TokenFactoryQuery>,
    creator_addr: String,
    subdenom: String,
) -> GetDenomResponse {
    let querier = TokenQuerier::new(&deps.querier);
    let response = querier.full_denom(creator_addr, subdenom).unwrap();

    GetDenomResponse {
        denom: response.denom,
    }
}

fn validate_denom(
    deps: DepsMut<TokenFactoryQuery>,
    denom: String,
) -> Result<(), TokenFactoryError> {
    let denom_to_split = denom.clone();
    let tokenfactory_denom_parts: Vec<&str> = denom_to_split.split('/').collect();

    if tokenfactory_denom_parts.len() != 3 {
        return Result::Err(TokenFactoryError::InvalidDenom {
            denom,
            message: std::format!(
                "denom must have 3 parts separated by /, had {}",
                tokenfactory_denom_parts.len()
            ),
        });
    }

    let prefix = tokenfactory_denom_parts[0];
    let creator_address = tokenfactory_denom_parts[1];
    let subdenom = tokenfactory_denom_parts[2];

    if !prefix.eq_ignore_ascii_case("factory") {
        return Result::Err(TokenFactoryError::InvalidDenom {
            denom,
            message: std::format!("prefix must be 'factory', was {}", prefix),
        });
    }

    // Validate denom by attempting to query for full denom
    let response = TokenQuerier::new(&deps.querier)
        .full_denom(String::from(creator_address), String::from(subdenom));
    if response.is_err() {
        return Result::Err(TokenFactoryError::InvalidDenom {
            denom,
            message: response.err().unwrap().to_string(),
        });
    }

    Result::Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use cosmwasm_std::testing::{
        mock_env, MockApi, MockQuerier, MockStorage, MOCK_CONTRACT_ADDR,
    };
    use cosmwasm_std::{
        coins, from_json, Attribute, ContractResult, CosmosMsg, MessageInfo, OwnedDeps, Querier,
        SystemError, SystemResult,
    };
    use std::marker::PhantomData;
    use token_bindings::TokenFactoryQuery;
    // use token_bindings_test::TokenFactoryApp;

    fn mock_info(sender: &str, funds: &[cosmwasm_std::Coin]) -> MessageInfo {
        MessageInfo {
            sender: cosmwasm_std::Addr::unchecked(sender),
            funds: funds.to_vec(),
        }
    }

    const DENOM_NAME: &str = "mydenom";
    const DENOM_PREFIX: &str = "factory";

    fn mock_dependencies_with_custom_quierier<Q: Querier>(
        querier: Q,
    ) -> OwnedDeps<MockStorage, MockApi, Q, TokenFactoryQuery> {
        OwnedDeps {
            storage: MockStorage::default(),
            api: MockApi::default(),
            querier,
            custom_query_type: PhantomData,
        }
    }

    fn mock_dependencies_with_query_error(
    ) -> OwnedDeps<MockStorage, MockApi, MockQuerier<TokenFactoryQuery>, TokenFactoryQuery> {
        let custom_querier: MockQuerier<TokenFactoryQuery> =
            MockQuerier::new(&[(MOCK_CONTRACT_ADDR, &[])]).with_custom_handler(|a| match a {
                TokenFactoryQuery::FullDenom {
                    creator_addr,
                    subdenom,
                } => {
                    let binary_request = to_json_binary(a).unwrap();

                    if creator_addr.eq("") {
                        return SystemResult::Err(SystemError::InvalidRequest {
                            error: String::from("invalid creator address"),
                            request: binary_request,
                        });
                    }
                    if subdenom.eq("") {
                        return SystemResult::Err(SystemError::InvalidRequest {
                            error: String::from("invalid subdenom"),
                            request: binary_request,
                        });
                    }
                    SystemResult::Ok(ContractResult::Ok(binary_request))
                }
                _ => todo!(),
            });
        mock_dependencies_with_custom_quierier(custom_querier)
    }

    pub fn mock_dependencies() -> OwnedDeps<MockStorage, MockApi, MockQuerier<TokenFactoryQuery>, TokenFactoryQuery>
    {
        let custom_querier: MockQuerier<TokenFactoryQuery> =
            MockQuerier::new(&[(MOCK_CONTRACT_ADDR, &[])]).with_custom_handler(|a| match a {
                TokenFactoryQuery::FullDenom {
                    creator_addr,
                    subdenom,
                } => {
                    let denom = format!("factory/{}/{}", creator_addr, subdenom);
                    let response = token_bindings::FullDenomResponse { denom };
                    SystemResult::Ok(ContractResult::Ok(to_json_binary(&response).unwrap()))
                }
                _ => todo!(),
            });
        mock_dependencies_with_custom_quierier(custom_querier)
    }

    #[test]
    fn proper_initialization() {
        let mut deps = mock_dependencies();

        let msg = InstantiateMsg {};
        let info = mock_info("creator", &coins(1000, "utoken"));

        let res = instantiate(deps.as_mut(), mock_env(), info, msg).unwrap();
        assert_eq!(0, res.messages.len());
    }

    #[test]
    fn query_get_denom() {
        let deps = mock_dependencies();
        let get_denom_query = QueryMsg::GetDenom {
            creator_address: String::from(MOCK_CONTRACT_ADDR),
            subdenom: String::from(DENOM_NAME),
        };
        let response = query(deps.as_ref(), mock_env(), get_denom_query).unwrap();
        let get_denom_response: GetDenomResponse = from_json(&response).unwrap();
        assert_eq!(
            format!("{}/{}/{}", DENOM_PREFIX, MOCK_CONTRACT_ADDR, DENOM_NAME),
            get_denom_response.denom
        );
    }

    #[test]
    fn msg_create_denom_success() {
        let mut deps = mock_dependencies();

        let subdenom: String = String::from(DENOM_NAME);

        let msg = ExecuteMsg::CreateDenom { subdenom };
        let info = mock_info("creator", &coins(2, "utoken"));
        let res = execute(deps.as_mut(), mock_env(), info, msg).unwrap();

        assert_eq!(1, res.messages.len());

        let expected_message = CosmosMsg::from(TokenFactoryMsg::CreateDenom {
            subdenom: String::from(DENOM_NAME),
            metadata: None,
        });
        let actual_message = res.messages.get(0).unwrap();
        assert_eq!(expected_message, actual_message.msg);

        assert_eq!(1, res.attributes.len());

        let expected_attribute = Attribute::new("method", "create_denom");
        let actual_attribute = res.attributes.get(0).unwrap();
        assert_eq!(expected_attribute, actual_attribute);

        assert_eq!(res.data.ok_or(0), Err(0));
    }

    #[test]
    fn msg_create_denom_invalid_subdenom() {
        let mut deps = mock_dependencies();

        let subdenom: String = String::from("");

        let msg = ExecuteMsg::CreateDenom { subdenom };
        let info = mock_info("creator", &coins(2, "utoken"));
        let err = execute(deps.as_mut(), mock_env(), info, msg).unwrap_err();
        assert!(matches!(err, TokenFactoryError::InvalidSubdenom { .. }));
        assert!(err.to_string().contains("Invalid subdenom"));
    }

    #[test]
    fn msg_validate_denom_too_many_parts_valid() {
        let mut deps = mock_dependencies();

        // too many parts in denom
        let full_denom_name: &str =
            &format!("{}/{}/{}", DENOM_PREFIX, MOCK_CONTRACT_ADDR, DENOM_NAME)[..];

        validate_denom(deps.as_mut(), String::from(full_denom_name)).unwrap()
    }

    #[test]
    fn msg_burn_tokens_success() {
        let mut deps = mock_dependencies();

        let mint_amount = Uint128::new(100_u128);
        let full_denom_name: &str =
            &format!("{}/{}/{}", DENOM_PREFIX, MOCK_CONTRACT_ADDR, DENOM_NAME)[..];

        let info = mock_info("creator", &coins(2, "utoken"));

        let msg = ExecuteMsg::BurnTokens {
            denom: String::from(full_denom_name),
            burn_from_address: None,
            amount: mint_amount,
        };
        let res = execute(deps.as_mut(), mock_env(), info, msg).unwrap();

        assert_eq!(1, res.messages.len());
        let expected_message = CosmosMsg::from(TokenFactoryMsg::BurnTokens {
            denom: String::from(full_denom_name),
            amount: mint_amount,
            burn_from_address: None,
        });
        let actual_message = res.messages.get(0).unwrap();
        assert_eq!(expected_message, actual_message.msg);

        assert_eq!(1, res.attributes.len());

        let expected_attribute = Attribute::new("method", "burn_tokens");
        let actual_attribute = res.attributes.get(0).unwrap();
        assert_eq!(expected_attribute, actual_attribute);

        assert_eq!(res.data.ok_or(0), Err(0))
    }

    #[test]
    fn msg_burn_tokens_input_address() {
        let mut deps = mock_dependencies();

        let burn_amount = Uint128::new(100_u128);
        let full_denom_name: &str =
            &format!("{}/{}/{}", DENOM_PREFIX, MOCK_CONTRACT_ADDR, DENOM_NAME)[..];

        let info = mock_info("creator", &coins(2, "utoken"));

        // Test that we can provide Some(address)
        let msg = ExecuteMsg::BurnTokens {
            denom: String::from(full_denom_name),
            burn_from_address: Some(MOCK_CONTRACT_ADDR.to_string()),
            amount: burn_amount,
        };
        let result = execute(deps.as_mut(), mock_env(), info, msg);
        // Should succeed since we're using MOCK_CONTRACT_ADDR which is valid
        assert!(result.is_ok())
    }

    #[test]
    fn msg_force_transfer_tokens_address() {
        let mut deps = mock_dependencies();

        const TRANSFER_FROM_ADDR: &str = "transferme";
        const TRANSFER_TO_ADDR: &str = "tome";

        let transfer_amount = Uint128::new(100_u128);
        let full_denom_name: &str =
            &format!("{}/{}/{}", DENOM_PREFIX, MOCK_CONTRACT_ADDR, DENOM_NAME)[..];

        let info = mock_info("creator", &coins(2, "utoken"));

        let msg = ExecuteMsg::ForceTransfer {
            denom: String::from(full_denom_name),
            amount: transfer_amount,
            from_address: TRANSFER_FROM_ADDR.to_string(),
            to_address: TRANSFER_TO_ADDR.to_string(),
        };

        let err = execute(deps.as_mut(), mock_env(), info, msg).is_ok();
        assert!(err)
    }

    #[test]
    fn msg_validate_denom_too_many_parts_invalid() {
        let mut deps = mock_dependencies();

        // too many parts in denom
        let full_denom_name: &str = &format!(
            "{}/{}/{}/invalid",
            DENOM_PREFIX, MOCK_CONTRACT_ADDR, DENOM_NAME
        )[..];

        let err = validate_denom(deps.as_mut(), String::from(full_denom_name)).unwrap_err();

        let expected_error = TokenFactoryError::InvalidDenom {
            denom: String::from(full_denom_name),
            message: String::from("denom must have 3 parts separated by /, had 4"),
        };

        assert!(matches!(err, TokenFactoryError::InvalidDenom { .. }));
        assert_eq!(err.to_string(), expected_error.to_string());
    }

    #[test]
    fn msg_validate_denom_not_enough_parts_invalid() {
        let mut deps = mock_dependencies();

        // too little parts in denom
        let full_denom_name: &str = &format!("{}/{}", DENOM_PREFIX, MOCK_CONTRACT_ADDR)[..];

        let err = validate_denom(deps.as_mut(), String::from(full_denom_name)).unwrap_err();

        let expected_error = TokenFactoryError::InvalidDenom {
            denom: String::from(full_denom_name),
            message: String::from("denom must have 3 parts separated by /, had 2"),
        };

        assert!(matches!(err, TokenFactoryError::InvalidDenom { .. }));
        assert_eq!(err.to_string(), expected_error.to_string());
    }

    #[test]
    fn msg_validate_denom_denom_prefix_invalid() {
        let mut deps = mock_dependencies();

        // invalid denom prefix
        let full_denom_name: &str =
            &format!("{}/{}/{}", "invalid", MOCK_CONTRACT_ADDR, DENOM_NAME)[..];

        let err = validate_denom(deps.as_mut(), String::from(full_denom_name)).unwrap_err();

        let expected_error = TokenFactoryError::InvalidDenom {
            denom: String::from(full_denom_name),
            message: String::from("prefix must be 'factory', was invalid"),
        };

        assert!(matches!(err, TokenFactoryError::InvalidDenom { .. }));
        assert_eq!(err.to_string(), expected_error.to_string());
    }

    #[test]
    fn msg_validate_denom_creator_address_invalid() {
        let mut deps = mock_dependencies_with_query_error();

        let full_denom_name: &str = &format!("{}/{}/{}", DENOM_PREFIX, "", DENOM_NAME)[..]; // empty contract address

        let err = validate_denom(deps.as_mut(), String::from(full_denom_name)).unwrap_err();

        match err {
            TokenFactoryError::InvalidDenom { denom, message } => {
                assert_eq!(String::from(full_denom_name), denom);
                assert!(message.contains("invalid creator address"))
            }
            err => panic!("Unexpected error: {:?}", err),
        }
    }

    #[test]
    fn test_burn_tokens_serialization() {
        use cosmwasm_std::to_json_string;

        // Test that burn_from_address is omitted when None
        let burn_none = TokenFactoryMsg::BurnTokens {
            denom: "factory/cosmos1.../subdenom".to_string(),
            amount: Uint128::new(100),
            burn_from_address: None,
        };
        let json_none = to_json_string(&burn_none).unwrap();
        assert!(!json_none.contains("burn_from_address"), "burn_from_address should not appear when None: {}", json_none);

        // Test that burn_from_address is included when Some
        let burn_some = TokenFactoryMsg::BurnTokens {
            denom: "factory/cosmos1.../subdenom".to_string(),
            amount: Uint128::new(100),
            burn_from_address: Some("cosmos1abc".to_string()),
        };
        let json_some = to_json_string(&burn_some).unwrap();
        assert!(json_some.contains("burn_from_address"), "burn_from_address should appear when Some: {}", json_some);
        assert!(json_some.contains("cosmos1abc"), "burn_from_address value should be included: {}", json_some);
    }
}
