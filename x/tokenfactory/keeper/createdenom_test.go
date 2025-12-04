package keeper_test

import (
	"fmt"

	"github.com/cosmos/tokenfactory/x/tokenfactory/keeper"
	"github.com/cosmos/tokenfactory/x/tokenfactory/types"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

func (suite *KeeperTestSuite) TestMsgCreateDenom() {
	// Creating a denom should work
	res, err := suite.msgServer.CreateDenom(suite.Ctx, types.NewMsgCreateDenom(suite.TestAccs[0].String(), "bitcoin"))
	suite.Require().NoError(err)
	suite.Require().NotEmpty(res.GetNewTokenDenom())

	// Make sure that the admin is set correctly
	queryRes, err := suite.queryClient.DenomAuthorityMetadata(suite.Ctx.Context(), &types.QueryDenomAuthorityMetadataRequest{
		Denom: res.GetNewTokenDenom(),
	})
	suite.Require().NoError(err)
	suite.Require().Equal(suite.TestAccs[0].String(), queryRes.AuthorityMetadata.Admin)

	// Make sure that the denom is valid from the perspective of x/bank
	bankQueryRes, err := suite.bankQueryClient.DenomMetadata(suite.Ctx.Context(), &banktypes.QueryDenomMetadataRequest{
		Denom: res.GetNewTokenDenom(),
	})
	suite.Require().NoError(err)
	suite.Require().NoError(bankQueryRes.Metadata.Validate())

	// Make sure that a second version of the same denom can't be recreated
	_, err = suite.msgServer.CreateDenom(suite.Ctx, types.NewMsgCreateDenom(suite.TestAccs[0].String(), "bitcoin"))
	suite.Require().Error(err)

	// Creating a second denom should work
	res, err = suite.msgServer.CreateDenom(suite.Ctx, types.NewMsgCreateDenom(suite.TestAccs[0].String(), "litecoin"))
	suite.Require().NoError(err)
	suite.Require().NotEmpty(res.GetNewTokenDenom())

	// Try querying all the denoms created by suite.TestAccs[0]
	queryRes2, err := suite.queryClient.DenomsFromCreator(suite.Ctx.Context(), &types.QueryDenomsFromCreatorRequest{
		Creator: suite.TestAccs[0].String(),
	})
	suite.Require().NoError(err)
	suite.Require().Len(queryRes2.Denoms, 2)

	// Make sure that a second account can create a denom with the same subdenom
	res, err = suite.msgServer.CreateDenom(suite.Ctx, types.NewMsgCreateDenom(suite.TestAccs[1].String(), "bitcoin"))
	suite.Require().NoError(err)
	suite.Require().NotEmpty(res.GetNewTokenDenom())

	// Make sure that an address with a "/" in it can't create denoms
	_, err = suite.msgServer.CreateDenom(suite.Ctx, types.NewMsgCreateDenom("osmosis.eth/creator", "bitcoin"))
	suite.Require().Error(err)
}

func (suite *KeeperTestSuite) TestCreateDenom() {
	var (
		primaryDenom            = types.DefaultParams().DenomCreationFee[0].Denom
		secondaryDenom          = "utwo"
		defaultDenomCreationFee = types.Params{DenomCreationFee: sdk.NewCoins(sdk.NewCoin(primaryDenom, sdkmath.NewInt(50000000)))}
		twoDenomCreationFee     = types.Params{DenomCreationFee: sdk.NewCoins(sdk.NewCoin(primaryDenom, sdkmath.NewInt(50000000)), sdk.NewCoin(secondaryDenom, sdkmath.NewInt(50000000)))}
		nilCreationFee          = types.Params{DenomCreationFee: nil}
		largeCreationFee        = types.Params{DenomCreationFee: sdk.NewCoins(sdk.NewCoin(primaryDenom, sdkmath.NewInt(5000000000)))}
	)

	for _, tc := range []struct {
		desc             string
		denomCreationFee types.Params
		setup            func()
		subdenom         string
		valid            bool
	}{
		{
			desc:             "subdenom too long",
			denomCreationFee: defaultDenomCreationFee,
			subdenom:         "assadsadsadasdasdsadsadsadsadsadsadsklkadaskkkdasdasedskhanhassyeunganassfnlksdflksafjlkasd",
			valid:            false,
		},
		{
			desc:             "subdenom and creator pair already exists",
			denomCreationFee: defaultDenomCreationFee,
			setup: func() {
				_, err := suite.msgServer.CreateDenom(suite.Ctx, types.NewMsgCreateDenom(suite.TestAccs[0].String(), "bitcoin"))
				suite.Require().NoError(err)
			},
			subdenom: "bitcoin",
			valid:    false,
		},
		{
			desc:             "success case: defaultDenomCreationFee",
			denomCreationFee: defaultDenomCreationFee,
			subdenom:         "evmos",
			valid:            true,
		},
		{
			desc:             "success case: twoDenomCreationFee",
			denomCreationFee: twoDenomCreationFee,
			subdenom:         "catcoin",
			valid:            true,
		},
		{
			desc:             "success case: nilCreationFee",
			denomCreationFee: nilCreationFee,
			subdenom:         "czcoin",
			valid:            true,
		},
		{
			desc:             "account doesn't have enough to pay for denom creation fee",
			denomCreationFee: largeCreationFee,
			subdenom:         "tooexpensive",
			valid:            false,
		},
		{
			desc:             "subdenom having invalid characters",
			denomCreationFee: defaultDenomCreationFee,
			subdenom:         "bit/***///&&&/coin",
			valid:            false,
		},
	} {
		suite.SetupTest()
		suite.Run(fmt.Sprintf("Case %s", tc.desc), func() {
			if tc.setup != nil {
				tc.setup()
			}
			tokenFactoryKeeper := suite.App.TokenFactoryKeeper
			bankKeeper := suite.App.BankKeeper
			// Set denom creation fee in params
			if err := tokenFactoryKeeper.SetParams(suite.Ctx, tc.denomCreationFee); err != nil {
				suite.Require().NoError(err)
			}
			denomCreationFee := tokenFactoryKeeper.GetParams(suite.Ctx).DenomCreationFee
			suite.Require().Equal(tc.denomCreationFee.DenomCreationFee, denomCreationFee)

			// note balance, create a tokenfactory denom, then note balance again
			// preCreateBalance := bankKeeper.GetAllBalances(suite.Ctx, suite.TestAccs[0])
			preCreateBalance := bankKeeper.GetBalance(suite.Ctx, suite.TestAccs[0], "stake")
			res, err := suite.msgServer.CreateDenom(suite.Ctx, types.NewMsgCreateDenom(suite.TestAccs[0].String(), tc.subdenom))
			// postCreateBalance := bankKeeper.GetAllBalances(suite.Ctx, suite.TestAccs[0])
			postCreateBalance := bankKeeper.GetBalance(suite.Ctx, suite.TestAccs[0], "stake")
			if tc.valid {
				suite.Require().NoError(err)
				if denomCreationFee != nil {
					suite.Require().True(preCreateBalance.Sub(postCreateBalance).IsEqual(denomCreationFee[0]))
				}

				// Make sure that the admin is set correctly
				queryRes, err := suite.queryClient.DenomAuthorityMetadata(suite.Ctx.Context(), &types.QueryDenomAuthorityMetadataRequest{
					Denom: res.GetNewTokenDenom(),
				})

				suite.Require().NoError(err)
				suite.Require().Equal(suite.TestAccs[0].String(), queryRes.AuthorityMetadata.Admin)

			} else {
				suite.Require().Error(err)
				// Ensure we don't charge if we expect an error
				suite.Require().True(preCreateBalance.IsEqual(postCreateBalance))
			}
		})
	}
}

func (suite *KeeperTestSuite) TestCreateDenomGasConsumption() {
	for _, tc := range []struct {
		desc                    string
		denomCreationGasConsume uint64
		expectedGasConsumed     uint64
	}{
		{
			desc:                    "gas consumption is zero",
			denomCreationGasConsume: 0,
			expectedGasConsumed:     0,
		},
		{
			desc:                    "gas consumption is set to 1000",
			denomCreationGasConsume: 1000,
			expectedGasConsumed:     1000,
		},
		{
			desc:                    "gas consumption is set to 2_000_000 (default)",
			denomCreationGasConsume: 2_000_000,
			expectedGasConsumed:     2_000_000,
		},
		{
			desc:                    "gas consumption is set to large value",
			denomCreationGasConsume: 10_000_000,
			expectedGasConsumed:     10_000_000,
		},
	} {
		suite.Run(fmt.Sprintf("Case %s", tc.desc), func() {
			suite.SetupTest()

			// Set the gas consumption parameter
			params := types.DefaultParams()
			params.DenomCreationGasConsume = tc.denomCreationGasConsume
			err := suite.App.TokenFactoryKeeper.SetParams(suite.Ctx, params)
			suite.Require().NoError(err)

			// Get gas consumed before creating denom
			gasConsumedBefore := suite.Ctx.GasMeter().GasConsumed()

			// Create a denom
			_, err = suite.msgServer.CreateDenom(suite.Ctx, types.NewMsgCreateDenom(suite.TestAccs[0].String(), "testcoin"))
			suite.Require().NoError(err)

			// Get gas consumed after creating denom
			gasConsumedAfter := suite.Ctx.GasMeter().GasConsumed()

			// Calculate the gas consumed by CreateDenom
			actualGasConsumed := gasConsumedAfter - gasConsumedBefore

			// The actual gas consumed should be at least the expected amount
			// (it may be slightly more due to other operations in CreateDenom)
			suite.Require().GreaterOrEqual(actualGasConsumed, tc.expectedGasConsumed,
				"Expected at least %d gas to be consumed, but only %d was consumed",
				tc.expectedGasConsumed, actualGasConsumed)

			// Verify the gas was consumed with the correct descriptor
			// We can't directly check the descriptor, but we can verify the amount is within a reasonable range
			// CreateDenom has other operations, so allow for some overhead (e.g., 100k gas)
			const overhead = uint64(100_000)
			suite.Require().LessOrEqual(actualGasConsumed, tc.expectedGasConsumed+overhead,
				"Gas consumed (%d) exceeds expected (%d) plus overhead (%d)",
				actualGasConsumed, tc.expectedGasConsumed, overhead)
		})
	}
}

func (suite *KeeperTestSuite) TestCommunityPoolFunding() {
	for _, tc := range []struct {
		desc                     string
		enableCommunityPoolFee   bool
		denomCreationFee         sdk.Coins
		expectCommunityPoolDelta bool
	}{
		{
			desc:                     "community pool funding enabled - fees should go to community pool",
			enableCommunityPoolFee:   true,
			denomCreationFee:         sdk.NewCoins(sdk.NewCoin("stake", sdkmath.NewInt(50_000_000))),
			expectCommunityPoolDelta: true,
		},
		{
			desc:                     "community pool funding disabled - fees should be burned",
			enableCommunityPoolFee:   false,
			denomCreationFee:         sdk.NewCoins(sdk.NewCoin("stake", sdkmath.NewInt(50_000_000))),
			expectCommunityPoolDelta: false,
		},
		{
			desc:                     "nil fee - no changes expected",
			enableCommunityPoolFee:   true,
			denomCreationFee:         nil,
			expectCommunityPoolDelta: false,
		},
	} {
		suite.Run(fmt.Sprintf("Case %s", tc.desc), func() {
			suite.SetupTest()

			// Configure the capability
			// Note: TokenFactoryKeeper is stored by value in the app. We can modify it using
			// SetEnabledCapabilities (which has a pointer receiver), but we must recreate the
			// msgServer afterward because it has its own copy of the keeper from SetupTest().
			var capabilities []string
			if tc.enableCommunityPoolFee {
				capabilities = []string{types.EnableCommunityPoolFeeFunding}
			}

			// Use the pointer to the keeper in the app struct to set capabilities
			keeperPtr := &suite.App.TokenFactoryKeeper
			keeperPtr.SetEnabledCapabilities(suite.Ctx, capabilities)

			// IMPORTANT: Recreate the msgServer because it has a copy of the keeper
			// The msgServer was created in SetupTest() before we modified the capabilities
			suite.msgServer = keeper.NewMsgServerImpl(suite.App.TokenFactoryKeeper)

			// Set the denom creation fee parameter
			params := types.DefaultParams()
			params.DenomCreationFee = tc.denomCreationFee
			err := suite.App.TokenFactoryKeeper.SetParams(suite.Ctx, params)
			suite.Require().NoError(err)

			// Get initial community pool balance
			communityPoolBefore := suite.GetCommunityPoolBalance()

			// Get initial user balance
			userBalanceBefore := suite.App.BankKeeper.GetBalance(suite.Ctx, suite.TestAccs[0], "stake")

			// Create a denom
			_, err = suite.msgServer.CreateDenom(suite.Ctx, types.NewMsgCreateDenom(suite.TestAccs[0].String(), "testcoin"))
			suite.Require().NoError(err)

			// Get final community pool balance
			communityPoolAfter := suite.GetCommunityPoolBalance()

			// Get final user balance
			userBalanceAfter := suite.App.BankKeeper.GetBalance(suite.Ctx, suite.TestAccs[0], "stake")

			// Calculate changes
			communityPoolDelta := communityPoolAfter.Sub(communityPoolBefore)
			userBalanceDelta := userBalanceBefore.Sub(userBalanceAfter)

			if tc.expectCommunityPoolDelta {
				// Verify fees went to community pool
				expectedDelta := sdk.NewDecCoinsFromCoins(tc.denomCreationFee...)
				suite.Require().Equal(expectedDelta, communityPoolDelta,
					"Community pool should increase by the denom creation fee amount")

				// Verify user balance decreased by the fee amount
				suite.Require().Equal(tc.denomCreationFee[0], userBalanceDelta,
					"User balance should decrease by the denom creation fee amount")
			} else {
				// Verify community pool did NOT increase
				suite.Require().True(communityPoolDelta.IsZero(),
					"Community pool should not increase when capability is disabled or fee is nil")

				if tc.denomCreationFee != nil {
					// Verify user was still charged (fees were burned)
					suite.Require().Equal(tc.denomCreationFee[0], userBalanceDelta,
						"User balance should decrease when fees are burned")
				} else {
					// Verify user was not charged when fee is nil
					suite.Require().True(userBalanceDelta.IsZero(),
						"User balance should not change when fee is nil")
				}
			}
		})
	}
}
