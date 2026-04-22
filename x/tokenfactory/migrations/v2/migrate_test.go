package v2_test

import (
	"testing"

	"github.com/cosmos/tokenfactory/x/tokenfactory"
	v2 "github.com/cosmos/tokenfactory/x/tokenfactory/migrations/v2"
	"github.com/cosmos/tokenfactory/x/tokenfactory/types"
	"github.com/stretchr/testify/require"

	sdkstore "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/testutil"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
)

func TestMigrate(t *testing.T) {
	// x/param conversion
	encCfg := moduletestutil.MakeTestEncodingConfig(tokenfactory.AppModuleBasic{})
	cdc := encCfg.Codec

	storeKey := sdkstore.NewKVStoreKey(v2.ModuleName)
	tKey := sdkstore.NewTransientStoreKey("transient_test")
	ctx := testutil.DefaultContext(storeKey, tKey)
	store := ctx.KVStore(storeKey)

	expectedParams := types.Params{
		DenomCreationFee:        nil,
		DenomCreationGasConsume: 2_000_000,
	}
	require.NoError(t, v2.Migrate(ctx, store, cdc))

	var res types.Params
	bz := store.Get(v2.ParamsKey)
	require.NoError(t, cdc.Unmarshal(bz, &res))
	require.Equal(t, expectedParams, res)
}
