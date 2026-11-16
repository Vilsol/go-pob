package mcp

import (
	"encoding/json"
)

const priceBuildDesc = "Price a build at the current league's poe.ninja prices (data from poe.ninja): total chaos and divines, each gear slot's and tree jewel's price and each enabled gem's, with price_source ninja (poe.ninja listing), estimate (a documented guess: base price is poe.ninja's item level 82+ crafting base at the item's own item level, 0 with no or a lower item level or an unlisted base; a normal item is its base price, a magic item its base price + 1c; a rare is base price + T1 20c, T2 5c, T3 and lower 1c per affix (a weapon's local damage affix, 'dmg' in detail: T1 100c, T2 20c), an 'other' affix the base's table cannot place (influence, essence, veiled, delve) as T1 + bench craft costs + eldritch implicits by tier (Lesser 0 to Perfect 80c each) + 10c per anoint + a 5/6-link at the bench's Fusing cost + 50c per level of '+N to Level of all ... Skill Gems' lines, doubled when two such lines share an item; detail names each part) or unknown (never counted as free; unknown counts them). A gem takes a cheaper corrupted listing of the same or a better level and quality ('corrupted listing'); any row but its exact level and quality says 'priced as <row>'. A ninja price carries its listings count and low_confidence when fewer than 10 listings back it; the build's low_confidence counts those rows. Also names the league, the price data's age and the divine rate's source (divine_source: the in-game Currency Exchange, else the median chaos/divine ratio of poe.ninja's item lines, else a stash currency line whose buy and sell sides agree within 2x)."

const schemaPriceBuild = `{"type":"object","required":["handle"],"additionalProperties":false,"properties":{
  "handle":{"type":"string"},
  "revision":{"type":"integer","description":"default: the current revision"}}}`

type priceBuildIn struct {
	Handle   string `json:"handle"`
	Revision *int   `json:"revision"`
}

func (s *Server) priceBuild(args json.RawMessage) (any, *ToolError) {
	var in priceBuildIn
	if e := decode(args, schemaPriceBuild, &in); e != nil {
		return nil, e
	}
	if s.opts.Prices == nil {
		e := errf(CodePricesUnavailable, "This server was started with --no-prices.")
		e.Hint = "Restart go-pob mcp without --no-prices, or with --prices <snapshot file>."
		return nil, e
	}
	h, e := s.get(in.Handle)
	if e != nil {
		return nil, e
	}
	rev := h.cur
	if in.Revision != nil {
		rev = *in.Revision
	}
	b, e := h.parseRev(rev)
	if e != nil {
		return nil, e
	}
	cost := PriceBuild(s.opts.Prices, b)
	if w := s.opts.Prices.Warning(); w != "" {
		s.pendingWarnings = append(s.pendingWarnings, w)
	}
	return map[string]any{"handle": h.id, "revision": rev, "cost": cost, "source": "poe.ninja"}, nil
}
