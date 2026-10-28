package penalty

import "patra-world-social/server/internal/games/model"

// BotChoice cannot inspect either player's choice or any engine state.
func BotChoice(rng model.RNG) int { return rng.Intn(4) }
