package bindings_test

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/cosmos/tokenfactory/app"
	"github.com/cosmos/tokenfactory/x/tokenfactory/types"
	"github.com/stretchr/testify/require"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// This file demonstrates direct usage of the tokenfactory CosmWasm contract
// from wasm/contracts/tokenfactory

// To compile a compatible version:
// 1. Clone https://github.com/cosmos/tokenfactory
// 2. Build with: RUSTFLAGS='-C link-arg=-s' cargo build --release --lib --target wasm32-unknown-unknown
// 3. Copy target/wasm32-unknown-unknown/release/tokenfactory.wasm to testdata/

// The contract provides these execute messages:
// - CreateDenom: Create a new token denomination
// - MintTokens: Mint tokens to an address
// - BurnTokens: Burn tokens from an address
// - ChangeAdmin: Change the admin of a denom
// - ForceTransfer: Force transfer tokens between addresses
//
// And this query message:
// - GetDenom: Get the full denom name from creator address and subdenom

// Message types matching the tokenfactory contract schema

type TokenFactoryInstantiateMsg struct{}

type TokenFactoryExecuteMsg struct {
	CreateDenom   *CreateDenomMsg   `json:"create_denom,omitempty"`
	ChangeAdmin   *ChangeAdminMsg   `json:"change_admin,omitempty"`
	MintTokens    *MintTokensMsg    `json:"mint_tokens,omitempty"`
	BurnTokens    *BurnTokensMsg    `json:"burn_tokens,omitempty"`
	ForceTransfer *ForceTransferMsg `json:"force_transfer,omitempty"`
}

type CreateDenomMsg struct {
	Subdenom string `json:"subdenom"`
}

type ChangeAdminMsg struct {
	Denom           string `json:"denom"`
	NewAdminAddress string `json:"new_admin_address"`
}

type MintTokensMsg struct {
	Denom         string `json:"denom"`
	Amount        string `json:"amount"`
	MintToAddress string `json:"mint_to_address,omitempty"`
}

type BurnTokensMsg struct {
	Denom           string `json:"denom"`
	Amount          string `json:"amount"`
	BurnFromAddress string `json:"burn_from_address,omitempty"`
}

type ForceTransferMsg struct {
	Denom       string `json:"denom"`
	Amount      string `json:"amount"`
	FromAddress string `json:"from_address"`
	ToAddress   string `json:"to_address"`
}

type TokenFactoryQueryMsg struct {
	GetDenom *GetDenomQuery `json:"get_denom,omitempty"`
}

type GetDenomQuery struct {
	CreatorAddress string `json:"creator_address"`
	Subdenom       string `json:"subdenom"`
}

type GetDenomResponse struct {
	Denom string `json:"denom"`
}

// Helper functions for tokenfactory contract

func storeTokenFactoryCode(t *testing.T, ctx sdk.Context, app *app.TokenFactoryApp, addr sdk.AccAddress) uint64 {
	wasmCode, err := os.ReadFile("./testdata/tokenfactory.wasm")
	require.NoError(t, err)

	instantiateAccess := wasmtypes.AccessTypeEverybody.With()
	contractKeeper := keeper.NewDefaultPermissionKeeper(app.WasmKeeper)
	codeID, _, err := contractKeeper.Create(ctx, addr, wasmCode, &instantiateAccess)
	require.NoError(t, err)

	return codeID
}

func instantiateTokenFactoryContract(t *testing.T, ctx sdk.Context, app *app.TokenFactoryApp, funder sdk.AccAddress, codeID uint64) sdk.AccAddress {
	initMsg := TokenFactoryInstantiateMsg{}
	initMsgBz, err := json.Marshal(initMsg)
	require.NoError(t, err)

	contractKeeper := keeper.NewDefaultPermissionKeeper(app.WasmKeeper)
	addr, _, err := contractKeeper.Instantiate(ctx, codeID, funder, funder, initMsgBz, "tokenfactory contract", nil)
	require.NoError(t, err)

	return addr
}

func executeTokenFactoryContract(t *testing.T, ctx sdk.Context, app *app.TokenFactoryApp, contract sdk.AccAddress, sender sdk.AccAddress, msg TokenFactoryExecuteMsg, funds sdk.Coins) error {
	msgBz, err := json.Marshal(msg)
	require.NoError(t, err)

	contractKeeper := keeper.NewDefaultPermissionKeeper(app.WasmKeeper)
	_, err = contractKeeper.Execute(ctx, contract, sender, msgBz, funds)
	return err
}

func queryTokenFactoryContract(t *testing.T, ctx sdk.Context, app *app.TokenFactoryApp, contract sdk.AccAddress, msg TokenFactoryQueryMsg, response interface{}) {
	msgBz, err := json.Marshal(msg)
	require.NoError(t, err)

	queryResponse, err := app.WasmKeeper.QuerySmart(ctx, contract, msgBz)
	require.NoError(t, err)

	err = json.Unmarshal(queryResponse, response)
	require.NoError(t, err)
}

func setupTokenFactoryContractTest(t *testing.T, addr sdk.AccAddress) (*app.TokenFactoryApp, sdk.Context, uint64) {
	app, ctx := CreateTestInput(t)

	codeID := storeTokenFactoryCode(t, ctx, app, addr)

	cInfo := app.WasmKeeper.GetCodeInfo(ctx, codeID)
	require.NotNil(t, cInfo)

	return app, ctx, codeID
}

// Tests
func TestTokenFactory_CreateDenom(t *testing.T) {
	creator := RandomAccountAddress()
	app, ctx, codeID := setupTokenFactoryContractTest(t, creator)

	owner := RandomAccountAddress()
	contract := instantiateTokenFactoryContract(t, ctx, app, owner, codeID)
	require.NotEmpty(t, contract)

	// Fund contract with denom creation fees
	creationFee := types.DefaultParams().DenomCreationFee[0]
	fundAmount := sdk.NewCoins(sdk.NewCoin(creationFee.Denom, creationFee.Amount.MulRaw(10)))
	fundAccount(t, ctx, app, contract, fundAmount)

	// Create a denom
	msg := TokenFactoryExecuteMsg{
		CreateDenom: &CreateDenomMsg{
			Subdenom: "SOLAR",
		},
	}
	err := executeTokenFactoryContract(t, ctx, app, contract, owner, msg, nil)
	require.NoError(t, err)

	// Query the denom to verify it was created
	queryMsg := TokenFactoryQueryMsg{
		GetDenom: &GetDenomQuery{
			CreatorAddress: contract.String(),
			Subdenom:       "SOLAR",
		},
	}
	var resp GetDenomResponse
	queryTokenFactoryContract(t, ctx, app, contract, queryMsg, &resp)

	expectedDenom := fmt.Sprintf("factory/%s/SOLAR", contract.String())
	require.Equal(t, expectedDenom, resp.Denom)
}

func TestTokenFactory_MintTokens(t *testing.T) {
	creator := RandomAccountAddress()
	app, ctx, codeID := setupTokenFactoryContractTest(t, creator)

	owner := RandomAccountAddress()
	// recipient := RandomAccountAddress()
	contract := instantiateTokenFactoryContract(t, ctx, app, owner, codeID)
	require.NotEmpty(t, contract)

	// Fund contract with denom creation fees
	creationFee := types.DefaultParams().DenomCreationFee[0]
	fundAmount := sdk.NewCoins(sdk.NewCoin(creationFee.Denom, creationFee.Amount.MulRaw(10)))
	fundAccount(t, ctx, app, contract, fundAmount)

	// Create a denom
	createMsg := TokenFactoryExecuteMsg{
		CreateDenom: &CreateDenomMsg{
			Subdenom: "LUNAR",
		},
	}
	err := executeTokenFactoryContract(t, ctx, app, contract, owner, createMsg, nil)
	require.NoError(t, err)

	denom := fmt.Sprintf("factory/%s/LUNAR", contract.String())

	// Mint tokens to recipient
	mintAmount := "1000000"
	mintMsg := TokenFactoryExecuteMsg{
		MintTokens: &MintTokensMsg{
			Denom:  denom,
			Amount: mintAmount,
		},
	}
	err = executeTokenFactoryContract(t, ctx, app, contract, owner, mintMsg, nil)
	require.NoError(t, err)

	// Verify recipient received the tokens
	balances := app.BankKeeper.GetAllBalances(ctx, contract)
	require.Len(t, balances, 2)
	require.Equal(t, denom, balances[0].Denom)
	require.Equal(t, mintAmount, balances[0].Amount.String())
}

func TestTokenFactory_MintTokensTo(t *testing.T) {
	creator := RandomAccountAddress()
	app, ctx, codeID := setupTokenFactoryContractTest(t, creator)

	owner := RandomAccountAddress()
	recipient := RandomAccountAddress()
	contract := instantiateTokenFactoryContract(t, ctx, app, owner, codeID)
	require.NotEmpty(t, contract)

	// Fund contract with denom creation fees
	creationFee := types.DefaultParams().DenomCreationFee[0]
	fundAmount := sdk.NewCoins(sdk.NewCoin(creationFee.Denom, creationFee.Amount.MulRaw(10)))
	fundAccount(t, ctx, app, contract, fundAmount)

	// Verify recipient starts with no balance
	balances := app.BankKeeper.GetAllBalances(ctx, recipient)
	require.Empty(t, balances)

	// Create a denom
	createMsg := TokenFactoryExecuteMsg{
		CreateDenom: &CreateDenomMsg{
			Subdenom: "LUNAR",
		},
	}
	err := executeTokenFactoryContract(t, ctx, app, contract, owner, createMsg, nil)
	require.NoError(t, err)

	denom := fmt.Sprintf("factory/%s/LUNAR", contract.String())

	// Mint tokens to recipient
	mintAmount := "1000000"
	mintMsg := TokenFactoryExecuteMsg{
		MintTokens: &MintTokensMsg{
			Denom:         denom,
			Amount:        mintAmount,
			MintToAddress: recipient.String(),
		},
	}
	err = executeTokenFactoryContract(t, ctx, app, contract, owner, mintMsg, nil)
	require.NoError(t, err)

	// Verify recipient received the tokens
	balances = app.BankKeeper.GetAllBalances(ctx, recipient)
	require.Len(t, balances, 1)
	require.Equal(t, denom, balances[0].Denom)
	require.Equal(t, mintAmount, balances[0].Amount.String())
}

func TestTokenFactory_BurnTokens(t *testing.T) {
	creator := RandomAccountAddress()
	app, ctx, codeID := setupTokenFactoryContractTest(t, creator)

	owner := RandomAccountAddress()
	contract := instantiateTokenFactoryContract(t, ctx, app, owner, codeID)
	require.NotEmpty(t, contract)

	// Fund contract with denom creation fees
	creationFee := types.DefaultParams().DenomCreationFee[0]
	fundAmount := sdk.NewCoins(sdk.NewCoin(creationFee.Denom, creationFee.Amount.MulRaw(10)))
	fundAccount(t, ctx, app, contract, fundAmount)

	// Create a denom
	createMsg := TokenFactoryExecuteMsg{
		CreateDenom: &CreateDenomMsg{
			Subdenom: "BURN",
		},
	}
	err := executeTokenFactoryContract(t, ctx, app, contract, owner, createMsg, nil)
	require.NoError(t, err)

	denom := fmt.Sprintf("factory/%s/BURN", contract.String())

	// Mint tokens to contract first
	mintAmount := "5000000"
	mintMsg := TokenFactoryExecuteMsg{
		MintTokens: &MintTokensMsg{
			Denom:  denom,
			Amount: mintAmount,
		},
	}
	err = executeTokenFactoryContract(t, ctx, app, contract, owner, mintMsg, nil)
	require.NoError(t, err)

	// Verify contract has the tokens
	balance := app.BankKeeper.GetBalance(ctx, contract, denom)
	require.Equal(t, mintAmount, balance.Amount.String())

	// Burn tokens from contract
	burnAmount := "2000000"
	burnMsg := TokenFactoryExecuteMsg{
		BurnTokens: &BurnTokensMsg{
			Denom:  denom,
			Amount: burnAmount,
		},
	}
	err = executeTokenFactoryContract(t, ctx, app, contract, owner, burnMsg, nil)
	require.NoError(t, err)

	// Verify tokens were burned
	balance = app.BankKeeper.GetBalance(ctx, contract, denom)
	expectedBalance := sdkmath.NewInt(3000000) // 5000000 - 2000000
	require.Equal(t, expectedBalance.String(), balance.Amount.String())
}

func TestTokenFactory_BurnTokensFrom(t *testing.T) {
	creator := RandomAccountAddress()
	app, ctx, codeID := setupTokenFactoryContractTest(t, creator)

	owner := RandomAccountAddress()
	contract := instantiateTokenFactoryContract(t, ctx, app, owner, codeID)
	recipient := RandomAccountAddress()
	require.NotEmpty(t, contract)

	// Fund contract with denom creation fees
	creationFee := types.DefaultParams().DenomCreationFee[0]
	fundAmount := sdk.NewCoins(sdk.NewCoin(creationFee.Denom, creationFee.Amount.MulRaw(10)))
	fundAccount(t, ctx, app, contract, fundAmount)

	// Create a denom
	createMsg := TokenFactoryExecuteMsg{
		CreateDenom: &CreateDenomMsg{
			Subdenom: "BURN",
		},
	}
	err := executeTokenFactoryContract(t, ctx, app, contract, owner, createMsg, nil)
	require.NoError(t, err)

	denom := fmt.Sprintf("factory/%s/BURN", contract.String())

	// Mint tokens to contract first
	mintAmount := "5000000"
	mintMsg := TokenFactoryExecuteMsg{
		MintTokens: &MintTokensMsg{
			Denom:         denom,
			Amount:        mintAmount,
			MintToAddress: recipient.String(),
		},
	}
	err = executeTokenFactoryContract(t, ctx, app, contract, owner, mintMsg, nil)
	require.NoError(t, err)

	// Verify contract has the tokens
	balance := app.BankKeeper.GetBalance(ctx, recipient, denom)
	require.Equal(t, mintAmount, balance.Amount.String())

	// Burn tokens from contract
	burnAmount := "2000000"
	burnMsg := TokenFactoryExecuteMsg{
		BurnTokens: &BurnTokensMsg{
			Denom:           denom,
			Amount:          burnAmount,
			BurnFromAddress: recipient.String(),
		},
	}
	err = executeTokenFactoryContract(t, ctx, app, contract, owner, burnMsg, nil)
	require.NoError(t, err)

	// Verify tokens were burned
	balance = app.BankKeeper.GetBalance(ctx, recipient, denom)
	expectedBalance := sdkmath.NewInt(3000000) // 5000000 - 2000000
	require.Equal(t, expectedBalance.String(), balance.Amount.String())
}

func TestTokenFactory_ChangeAdmin(t *testing.T) {
	creator := RandomAccountAddress()
	app, ctx, codeID := setupTokenFactoryContractTest(t, creator)

	owner := RandomAccountAddress()
	newAdmin := RandomAccountAddress()
	contract := instantiateTokenFactoryContract(t, ctx, app, owner, codeID)
	require.NotEmpty(t, contract)

	// Fund contract with denom creation fees
	creationFee := types.DefaultParams().DenomCreationFee[0]
	fundAmount := sdk.NewCoins(sdk.NewCoin(creationFee.Denom, creationFee.Amount.MulRaw(10)))
	fundAccount(t, ctx, app, contract, fundAmount)

	// Create a denom
	createMsg := TokenFactoryExecuteMsg{
		CreateDenom: &CreateDenomMsg{
			Subdenom: "ADMIN",
		},
	}
	err := executeTokenFactoryContract(t, ctx, app, contract, owner, createMsg, nil)
	require.NoError(t, err)

	denom := fmt.Sprintf("factory/%s/ADMIN", contract.String())

	// Change admin
	changeAdminMsg := TokenFactoryExecuteMsg{
		ChangeAdmin: &ChangeAdminMsg{
			Denom:           denom,
			NewAdminAddress: newAdmin.String(),
		},
	}
	err = executeTokenFactoryContract(t, ctx, app, contract, owner, changeAdminMsg, nil)
	require.NoError(t, err)

	// Verify admin was changed
	authority, err := app.TokenFactoryKeeper.GetAuthorityMetadata(ctx, denom)
	require.NoError(t, err)
	require.Equal(t, newAdmin.String(), authority.Admin)
}

func TestTokenFactory_ForceTransfer(t *testing.T) {
	creator := RandomAccountAddress()
	app, ctx, codeID := setupTokenFactoryContractTest(t, creator)

	owner := RandomAccountAddress()
	alice := RandomAccountAddress()
	bob := RandomAccountAddress()
	contract := instantiateTokenFactoryContract(t, ctx, app, owner, codeID)
	require.NotEmpty(t, contract)

	// Fund contract with denom creation fees
	creationFee := types.DefaultParams().DenomCreationFee[0]
	fundAmount := sdk.NewCoins(sdk.NewCoin(creationFee.Denom, creationFee.Amount.MulRaw(10)))
	fundAccount(t, ctx, app, contract, fundAmount)

	// Create a denom
	createMsg := TokenFactoryExecuteMsg{
		CreateDenom: &CreateDenomMsg{
			Subdenom: "FORCE",
		},
	}
	err := executeTokenFactoryContract(t, ctx, app, contract, owner, createMsg, nil)
	require.NoError(t, err)

	denom := fmt.Sprintf("factory/%s/FORCE", contract.String())

	// Mint tokens to alice
	mintAmount := "10000000"
	mintMsg := TokenFactoryExecuteMsg{
		MintTokens: &MintTokensMsg{
			Denom:         denom,
			Amount:        mintAmount,
			MintToAddress: alice.String(),
		},
	}
	err = executeTokenFactoryContract(t, ctx, app, contract, owner, mintMsg, nil)
	require.NoError(t, err)

	// Verify alice has the tokens
	aliceBalance := app.BankKeeper.GetBalance(ctx, alice, denom)
	require.Equal(t, mintAmount, aliceBalance.Amount.String())

	// Verify bob has no tokens
	bobBalance := app.BankKeeper.GetBalance(ctx, bob, denom)
	require.True(t, bobBalance.Amount.IsZero())

	// Force transfer from alice to bob
	transferAmount := "3000000"
	forceTransferMsg := TokenFactoryExecuteMsg{
		ForceTransfer: &ForceTransferMsg{
			Denom:       denom,
			Amount:      transferAmount,
			FromAddress: alice.String(),
			ToAddress:   bob.String(),
		},
	}
	err = executeTokenFactoryContract(t, ctx, app, contract, owner, forceTransferMsg, nil)
	require.NoError(t, err)

	// Verify balances after force transfer
	aliceBalance = app.BankKeeper.GetBalance(ctx, alice, denom)
	bobBalance = app.BankKeeper.GetBalance(ctx, bob, denom)

	expectedAliceBalance := sdkmath.NewInt(7000000) // 10000000 - 3000000
	expectedBobBalance := sdkmath.NewInt(3000000)

	require.Equal(t, expectedAliceBalance.String(), aliceBalance.Amount.String())
	require.Equal(t, expectedBobBalance.String(), bobBalance.Amount.String())
}

func TestTokenFactory_MultipleOperations(t *testing.T) {
	creator := RandomAccountAddress()
	app, ctx, codeID := setupTokenFactoryContractTest(t, creator)

	owner := RandomAccountAddress()
	user1 := RandomAccountAddress()
	contract := instantiateTokenFactoryContract(t, ctx, app, owner, codeID)
	require.NotEmpty(t, contract)

	// Fund contract with denom creation fees for multiple denoms
	creationFee := types.DefaultParams().DenomCreationFee[0]
	fundAmount := sdk.NewCoins(sdk.NewCoin(creationFee.Denom, creationFee.Amount.MulRaw(100)))
	fundAccount(t, ctx, app, contract, fundAmount)

	// Create multiple denoms
	denoms := []string{"GOLD", "SILVER", "BRONZE"}
	for _, subdenom := range denoms {
		createMsg := TokenFactoryExecuteMsg{
			CreateDenom: &CreateDenomMsg{
				Subdenom: subdenom,
			},
		}
		err := executeTokenFactoryContract(t, ctx, app, contract, owner, createMsg, nil)
		require.NoError(t, err)
	}

	// Mint different amounts to user1
	for i, subdenom := range denoms {
		denom := fmt.Sprintf("factory/%s/%s", contract.String(), subdenom)
		amount := fmt.Sprintf("%d000000", (i+1)*100) // 100000000, 200000000, 300000000

		mintMsg := TokenFactoryExecuteMsg{
			MintTokens: &MintTokensMsg{
				Denom:         denom,
				Amount:        amount,
				MintToAddress: user1.String(),
			},
		}
		err := executeTokenFactoryContract(t, ctx, app, contract, owner, mintMsg, nil)
		require.NoError(t, err)
	}

	// Verify user1 has all three denoms
	balances := app.BankKeeper.GetAllBalances(ctx, user1)
	require.Len(t, balances, 3)

	// Query each denom to verify
	for _, subdenom := range denoms {
		queryMsg := TokenFactoryQueryMsg{
			GetDenom: &GetDenomQuery{
				CreatorAddress: contract.String(),
				Subdenom:       subdenom,
			},
		}
		var resp GetDenomResponse
		queryTokenFactoryContract(t, ctx, app, contract, queryMsg, &resp)

		expectedDenom := fmt.Sprintf("factory/%s/%s", contract.String(), subdenom)
		require.Equal(t, expectedDenom, resp.Denom)
	}
}

func TestTokenFactory_CreateWithFunds(t *testing.T) {
	creator := RandomAccountAddress()
	app, ctx, codeID := setupTokenFactoryContractTest(t, creator)

	owner := RandomAccountAddress()
	contract := instantiateTokenFactoryContract(t, ctx, app, owner, codeID)
	require.NotEmpty(t, contract)

	// Fund the owner (not the contract) with denom creation fees
	creationFee := types.DefaultParams().DenomCreationFee[0]
	fundAmount := sdk.NewCoins(sdk.NewCoin(creationFee.Denom, creationFee.Amount.MulRaw(10)))
	fundAccount(t, ctx, app, owner, fundAmount)

	// Create a denom, sending funds with the message
	msg := TokenFactoryExecuteMsg{
		CreateDenom: &CreateDenomMsg{
			Subdenom: "FUNDED",
		},
	}
	funds := sdk.NewCoins(creationFee)
	err := executeTokenFactoryContract(t, ctx, app, contract, owner, msg, funds)
	require.NoError(t, err)

	// Verify denom was created
	queryMsg := TokenFactoryQueryMsg{
		GetDenom: &GetDenomQuery{
			CreatorAddress: contract.String(),
			Subdenom:       "FUNDED",
		},
	}
	var resp GetDenomResponse
	queryTokenFactoryContract(t, ctx, app, contract, queryMsg, &resp)

	expectedDenom := fmt.Sprintf("factory/%s/FUNDED", contract.String())
	require.Equal(t, expectedDenom, resp.Denom)
}
