package evm

import (
	"math/big"
	"testing"

	sdkmath "cosmossdk.io/math"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/log"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	"github.com/ethereum/go-ethereum/params"
	"github.com/mezo-org/mezod/encoding"
	"github.com/mezo-org/mezod/types"
	evmtypes "github.com/mezo-org/mezod/x/evm/types"
)

var _ DynamicFeeEVMKeeper = MockEVMKeeper{}

type MockEVMKeeper struct {
	BaseFee        *big.Int
	EnableLondonHF bool
}

func (m MockEVMKeeper) GetBaseFee(_ sdk.Context, _ *params.ChainConfig) *big.Int {
	if m.EnableLondonHF {
		return m.BaseFee
	}
	return nil
}

func (m MockEVMKeeper) GetParams(_ sdk.Context) evmtypes.Params {
	return evmtypes.DefaultParams()
}

func (m MockEVMKeeper) ChainID() *big.Int {
	return big.NewInt(31611)
}

func TestSDKTxFeeChecker(t *testing.T) {
	// testCases:
	//   fallback
	//      genesis tx
	//      checkTx, validate with min-gas-prices
	//      deliverTx, no validation
	//   dynamic fee
	//      with extension option
	//      without extension option
	//      london hardfork enableness
	encodingConfig := encoding.MakeConfig(module.NewBasicManager())
	minGasPrices := sdk.NewDecCoins(sdk.NewDecCoin(evmtypes.DefaultEVMDenom, sdkmath.NewInt(10)))

	genesisCtx := sdk.NewContext(nil, tmproto.Header{}, false, log.NewNopLogger())
	checkTxCtx := sdk.NewContext(nil, tmproto.Header{Height: 1}, true, log.NewNopLogger()).WithMinGasPrices(minGasPrices)
	deliverTxCtx := sdk.NewContext(nil, tmproto.Header{Height: 1}, false, log.NewNopLogger())

	testCases := []struct {
		name        string
		ctx         sdk.Context
		keeper      DynamicFeeEVMKeeper
		buildTx     func() sdk.FeeTx
		expFees     string
		expPriority int64
		expSuccess  bool
	}{
		{
			"success, genesis tx",
			genesisCtx,
			MockEVMKeeper{},
			func() sdk.FeeTx {
				return encodingConfig.TxConfig.NewTxBuilder().GetTx()
			},
			"",
			0,
			true,
		},
		{
			"fail, min-gas-prices",
			checkTxCtx,
			MockEVMKeeper{},
			func() sdk.FeeTx {
				return encodingConfig.TxConfig.NewTxBuilder().GetTx()
			},
			"",
			0,
			false,
		},
		{
			"success, min-gas-prices",
			checkTxCtx,
			MockEVMKeeper{},
			func() sdk.FeeTx {
				txBuilder := encodingConfig.TxConfig.NewTxBuilder()
				txBuilder.SetGasLimit(1)
				txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewCoin(evmtypes.DefaultEVMDenom, sdkmath.NewInt(10))))
				return txBuilder.GetTx()
			},
			"10abtc",
			0,
			true,
		},
		{
			"success, min-gas-prices deliverTx",
			deliverTxCtx,
			MockEVMKeeper{},
			func() sdk.FeeTx {
				return encodingConfig.TxConfig.NewTxBuilder().GetTx()
			},
			"",
			0,
			true,
		},
		{
			"fail, dynamic fee",
			deliverTxCtx,
			MockEVMKeeper{
				EnableLondonHF: true, BaseFee: big.NewInt(1),
			},
			func() sdk.FeeTx {
				txBuilder := encodingConfig.TxConfig.NewTxBuilder()
				txBuilder.SetGasLimit(1)
				return txBuilder.GetTx()
			},
			"",
			0,
			false,
		},
		{
			"success, dynamic fee",
			deliverTxCtx,
			MockEVMKeeper{
				EnableLondonHF: true, BaseFee: big.NewInt(10),
			},
			func() sdk.FeeTx {
				txBuilder := encodingConfig.TxConfig.NewTxBuilder()
				txBuilder.SetGasLimit(1)
				txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewCoin(evmtypes.DefaultEVMDenom, sdkmath.NewInt(10))))
				return txBuilder.GetTx()
			},
			"10abtc",
			0,
			true,
		},
		{
			"success, dynamic fee priority",
			deliverTxCtx,
			MockEVMKeeper{
				EnableLondonHF: true, BaseFee: big.NewInt(10),
			},
			func() sdk.FeeTx {
				txBuilder := encodingConfig.TxConfig.NewTxBuilder()
				txBuilder.SetGasLimit(1)
				txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewCoin(evmtypes.DefaultEVMDenom, sdkmath.NewInt(10).Mul(evmtypes.DefaultPriorityReduction).Add(sdkmath.NewInt(10)))))
				return txBuilder.GetTx()
			},
			"10000010abtc",
			10,
			true,
		},
		{
			"success, dynamic fee empty tipFeeCap",
			deliverTxCtx,
			MockEVMKeeper{
				EnableLondonHF: true, BaseFee: big.NewInt(10),
			},
			func() sdk.FeeTx {
				txBuilder := encodingConfig.TxConfig.NewTxBuilder().(authtx.ExtensionOptionsTxBuilder)
				txBuilder.SetGasLimit(1)
				txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewCoin(evmtypes.DefaultEVMDenom, sdkmath.NewInt(10).Mul(evmtypes.DefaultPriorityReduction))))

				option, err := codectypes.NewAnyWithValue(&types.ExtensionOptionDynamicFeeTx{})
				require.NoError(t, err)
				txBuilder.SetExtensionOptions(option)
				return txBuilder.GetTx()
			},
			"10abtc",
			0,
			true,
		},
		{
			"success, dynamic fee tipFeeCap",
			deliverTxCtx,
			MockEVMKeeper{
				EnableLondonHF: true, BaseFee: big.NewInt(10),
			},
			func() sdk.FeeTx {
				txBuilder := encodingConfig.TxConfig.NewTxBuilder().(authtx.ExtensionOptionsTxBuilder)
				txBuilder.SetGasLimit(1)
				txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewCoin(evmtypes.DefaultEVMDenom, sdkmath.NewInt(10).Mul(evmtypes.DefaultPriorityReduction).Add(sdkmath.NewInt(10)))))

				option, err := codectypes.NewAnyWithValue(&types.ExtensionOptionDynamicFeeTx{
					MaxPriorityPrice: sdkmath.NewInt(5).Mul(evmtypes.DefaultPriorityReduction),
				})
				require.NoError(t, err)
				txBuilder.SetExtensionOptions(option)
				return txBuilder.GetTx()
			},
			"5000010abtc",
			5,
			true,
		},
		{
			"fail, negative dynamic fee tipFeeCap",
			deliverTxCtx,
			MockEVMKeeper{
				EnableLondonHF: true, BaseFee: big.NewInt(10),
			},
			func() sdk.FeeTx {
				txBuilder := encodingConfig.TxConfig.NewTxBuilder().(authtx.ExtensionOptionsTxBuilder)
				txBuilder.SetGasLimit(1)
				txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewCoin(evmtypes.DefaultEVMDenom, sdkmath.NewInt(10).Mul(evmtypes.DefaultPriorityReduction).Add(sdkmath.NewInt(10)))))

				// set negative priority fee
				option, err := codectypes.NewAnyWithValue(&types.ExtensionOptionDynamicFeeTx{
					MaxPriorityPrice: sdkmath.NewInt(-5).Mul(evmtypes.DefaultPriorityReduction),
				})
				require.NoError(t, err)
				txBuilder.SetExtensionOptions(option)
				return txBuilder.GetTx()
			},
			"",
			0,
			false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fees, priority, err := NewDynamicFeeChecker(tc.keeper)(tc.ctx, tc.buildTx())
			if tc.expSuccess {
				require.Equal(t, tc.expFees, fees.String())
				require.Equal(t, tc.expPriority, priority)
			} else {
				require.Error(t, err)
			}
		})
	}
}
