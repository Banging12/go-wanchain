package incentive

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"math/big"
	"math/rand"
	"testing"
)

func TestAddEpochGas(t *testing.T) {

	testTimes := 100
	randNums := make([]*big.Int, 0)

	random := rand.NewSource(0)

	for index := 0; index < testTimes; index++ {
		for i := 0; i < testTimes; i++ {
			gas := big.NewInt(random.Int63())
			AddEpochGas(statedb, gas, uint64(index))
			randNums = append(randNums, gas)
		}
	}

	for index := 0; index < testTimes; index++ {
		totalInEpoch := big.NewInt(0)
		for i := 0; i < testTimes; i++ {
			totalInEpoch = totalInEpoch.Add(totalInEpoch, randNums[index*testTimes+i])
		}

		gas := getEpochGas(statedb, uint64(index))
		if gas.String() != totalInEpoch.String() {
			t.FailNow()
		}
	}
}

func TestAddEpochGasFail(t *testing.T) {
	statedb, _ = state.New(common.Hash{}, state.NewDatabase(db), nil)
	testTimes := 100
	randNums := make([]*big.Int, 0)
	random := rand.NewSource(0)

	for index := 0; index < testTimes; index++ {
		for i := 0; i < testTimes; i++ {
			gas := big.NewInt(random.Int63())
			AddEpochGas(nil, gas, uint64(index))
			randNums = append(randNums, gas)
		}
	}

	for index := 0; index < testTimes; index++ {
		gas := getEpochGas(statedb, uint64(index))
		if gas.Uint64() != 0 {
			t.FailNow()
		}
	}
}
