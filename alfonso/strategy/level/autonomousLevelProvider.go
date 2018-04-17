package level

import "log"

// autonomousLevelProvider provides levels based on an exponential curve wrt. the number of assets held in the account.
// This strategy does not allow using the balance of a single asset for more strategies other than this one because
// that would require building in some trade tracking along with asset balance tracking for this strategy. The support
// for this can always be added later.
type autonomousLevelProvider struct {
	spread                        float64
	plateauThresholdPercentage    float64 // flattens price if any asset has this ratio of the total number of tokens
	useMaxQuoteInTargetAmountCalc bool    // else use maxBase
	amountSpread                  float64 // % that we take off the top of each amount order size which effectively serves as our spread when multiple levels are consumed
}

// ensure it implements Provider
var _ Provider = &autonomousLevelProvider{}

// MakeAutonomousLevelProvider is the factory method
func MakeAutonomousLevelProvider(spread float64, plateauThresholdPercentage float64, useMaxQuoteInTargetAmountCalc bool, amountSpread float64) Provider {
	if amountSpread >= 1.0 || amountSpread <= 0.0 {
		log.Fatal("amountSpread needs to be between 0 and 1 (exclusive): ", amountSpread)
	}

	return &autonomousLevelProvider{
		spread: spread,
		plateauThresholdPercentage:    plateauThresholdPercentage,
		useMaxQuoteInTargetAmountCalc: useMaxQuoteInTargetAmountCalc,
		amountSpread:                  amountSpread,
	}
}

// GetLevels impl.
func (p *autonomousLevelProvider) GetLevels(maxAssetBase float64, maxAssetQuote float64) ([]Level, error) {
	sum := maxAssetQuote + maxAssetBase
	var centerPrice float64
	if maxAssetQuote/sum >= p.plateauThresholdPercentage {
		centerPrice = p.plateauThresholdPercentage / (1 - p.plateauThresholdPercentage)
	} else if maxAssetBase/sum >= p.plateauThresholdPercentage {
		centerPrice = (1 - p.plateauThresholdPercentage) / p.plateauThresholdPercentage
	} else {
		centerPrice = maxAssetQuote / maxAssetBase
	}

	// price always adds the spread
	targetPrice := centerPrice * (1 + p.spread/2)

	targetAmount := (2 * maxAssetBase * p.spread) / (4 + p.spread)
	if p.useMaxQuoteInTargetAmountCalc {
		targetAmount = (2 * maxAssetQuote * p.spread) / (4 + p.spread)
	}
	// since targetAmount needs to be less then what we've set above based on the inequality formula, let's reduce it by 5%
	targetAmount *= (1 - p.amountSpread)
	level := Level{
		targetPrice:  targetPrice,
		targetAmount: targetAmount,
	}
	return []Level{level}, nil
}
