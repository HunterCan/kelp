package main

import (
	"fmt"
	"sort"
	"time"

	"github.com/lightyeario/kelp/api"
	"github.com/lightyeario/kelp/model"
	"github.com/lightyeario/kelp/plugins"
	"github.com/lightyeario/kelp/support/utils"
	"github.com/stellar/go/build"
	"github.com/stellar/go/clients/horizon"
	"github.com/stellar/go/support/log"
)

const maxLumenTrust float64 = 100000000000

// Bot represents a market making bot, which is composed of various parts include the strategy and various APIs.
type Bot struct {
	api                 *horizon.Client
	assetBase           horizon.Asset
	assetQuote          horizon.Asset
	tradingAccount      string
	sdex                *plugins.SDEX
	strat               api.Strategy // the instance of this bot is bound to this strategy
	tickIntervalSeconds int32
	dataKey             *model.BotKey

	// uninitialized
	maxAssetA      float64
	maxAssetB      float64
	trustAssetA    float64
	trustAssetB    float64
	buyingAOffers  []horizon.Offer // quoted A/B
	sellingAOffers []horizon.Offer // quoted B/A
}

// MakeBot is the factory method for the Bot struct
func MakeBot(
	api *horizon.Client,
	assetBase horizon.Asset,
	assetQuote horizon.Asset,
	tradingAccount string,
	sdex *plugins.SDEX,
	strat api.Strategy,
	tickIntervalSeconds int32,
	dataKey *model.BotKey,
) *Bot {
	return &Bot{
		api:                 api,
		assetBase:           assetBase,
		assetQuote:          assetQuote,
		tradingAccount:      tradingAccount,
		sdex:                sdex,
		strat:               strat,
		tickIntervalSeconds: tickIntervalSeconds,
		dataKey:             dataKey,
	}
}

// Start starts the bot with the injected strategy
func (b *Bot) Start() {
	for {
		b.update()
		log.Info(fmt.Sprintf("sleeping for %d seconds...", b.tickIntervalSeconds))
		time.Sleep(time.Duration(b.tickIntervalSeconds) * time.Second)
	}
}

// deletes all offers for the bot (not all offers on the account)
func (b *Bot) deleteAllOffers() {
	dOps := []build.TransactionMutator{}

	dOps = append(dOps, b.sdex.DeleteAllOffers(b.sellingAOffers)...)
	b.sellingAOffers = []horizon.Offer{}
	dOps = append(dOps, b.sdex.DeleteAllOffers(b.buyingAOffers)...)
	b.buyingAOffers = []horizon.Offer{}

	log.Info(fmt.Sprintf("deleting %d offers", len(dOps)))
	if len(dOps) > 0 {
		e := b.sdex.SubmitOps(dOps)
		if e != nil {
			log.Warn(e)
			return
		}
	}
}

// time to update the order book and possibly readjust the offers
func (b *Bot) update() {
	var e error
	b.load()
	b.loadExistingOffers()

	// strategy has a chance to set any state it needs
	e = b.strat.PreUpdate(b.maxAssetA, b.maxAssetB, b.trustAssetA, b.trustAssetB)
	if e != nil {
		log.Warn(e)
		b.deleteAllOffers()
		return
	}

	// delete excess offers
	var pruneOps []build.TransactionMutator
	pruneOps, b.buyingAOffers, b.sellingAOffers = b.strat.PruneExistingOffers(b.buyingAOffers, b.sellingAOffers)
	if len(pruneOps) > 0 {
		e = b.sdex.SubmitOps(pruneOps)
		if e != nil {
			log.Warn(e)
			b.deleteAllOffers()
			return
		}
	}

	// reset cached xlm exposure here so we only compute it once per update
	// TODO 2 - calculate this here and pass it in
	b.sdex.ResetCachedXlmExposure()
	ops, e := b.strat.UpdateWithOps(b.buyingAOffers, b.sellingAOffers)
	if e != nil {
		log.Warn(e)
		b.deleteAllOffers()
		return
	}

	if len(ops) > 0 {
		e = b.sdex.SubmitOps(ops)
		if e != nil {
			log.Warn(e)
			b.deleteAllOffers()
			return
		}
	}

	e = b.strat.PostUpdate()
	if e != nil {
		log.Warn(e)
		b.deleteAllOffers()
		return
	}
}

func (b *Bot) load() {
	// load the maximum amounts we can offer for each asset
	account, e := b.api.LoadAccount(b.tradingAccount)
	if e != nil {
		log.Info(e)
		return
	}

	var maxA float64
	var maxB float64
	var trustA float64
	var trustB float64
	for _, balance := range account.Balances {
		if balance.Asset == b.assetBase {
			maxA = utils.AmountStringAsFloat(balance.Balance)
			if balance.Asset.Type == utils.Native {
				trustA = maxLumenTrust
			} else {
				trustA = utils.AmountStringAsFloat(balance.Limit)
			}
			log.Infof("maxA: %.7f, trustA: %.7f\n", maxA, trustA)
		} else if balance.Asset == b.assetQuote {
			maxB = utils.AmountStringAsFloat(balance.Balance)
			if balance.Asset.Type == utils.Native {
				trustB = maxLumenTrust
			} else {
				trustB = utils.AmountStringAsFloat(balance.Limit)
			}
			log.Infof("maxB: %.7f, trustB: %.7f\n", maxB, trustB)
		}
	}
	b.maxAssetA = maxA
	b.maxAssetB = maxB
	b.trustAssetA = trustA
	b.trustAssetB = trustB
}

func (b *Bot) loadExistingOffers() {
	offers, e := utils.LoadAllOffers(b.tradingAccount, b.api)
	if e != nil {
		log.Warn(e)
		return
	}
	b.sellingAOffers, b.buyingAOffers = utils.FilterOffers(offers, b.assetBase, b.assetQuote)

	sort.Sort(utils.ByPrice(b.buyingAOffers))
	sort.Sort(utils.ByPrice(b.sellingAOffers)) // don't need to reverse since the prices are inverse
}
