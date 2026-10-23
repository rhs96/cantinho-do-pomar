package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/rhs96/cantinho-do-pomar/services/api/internal/app/farming"
	"github.com/rhs96/cantinho-do-pomar/services/api/internal/app/shop"
	"github.com/rhs96/cantinho-do-pomar/services/api/internal/domain/farm"
	"github.com/rhs96/cantinho-do-pomar/services/api/internal/domain/inventory"
	"github.com/rhs96/cantinho-do-pomar/services/api/internal/domain/player"
)

var errInvalidBody = errors.New("invalid request body")

func decodeJSON(r *http.Request, v any) error {
	return json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<10)).Decode(v)
}

// appErrors maps domain and use case errors to stable codes the client can
// switch on. Anything not listed is a 500.
var appErrors = []struct {
	err     error
	status  int
	code    string
	message string
}{
	{errInvalidBody, http.StatusBadRequest, "invalid_body", "the request body is not valid JSON for this endpoint"},
	{shop.ErrInvalidQuantity, http.StatusBadRequest, "invalid_quantity", "quantity must be between 1 and 99"},

	{farm.ErrNotFound, http.StatusNotFound, "farm_not_found", "open the farm before acting on it"},
	{farm.ErrOutOfBounds, http.StatusNotFound, "plot_not_found", "no plot at this position"},

	{farming.ErrUnknownCrop, http.StatusUnprocessableEntity, "unknown_crop", "this crop does not exist"},
	{shop.ErrUnknownCrop, http.StatusUnprocessableEntity, "unknown_crop", "this crop does not exist"},

	{farm.ErrAlreadyPlowed, http.StatusConflict, "already_plowed", "the plot is already plowed"},
	{farm.ErrPlotOccupied, http.StatusConflict, "plot_occupied", "something is already growing here"},
	{farm.ErrNotPlowed, http.StatusConflict, "not_plowed", "plow the plot first"},
	{farm.ErrNothingPlanted, http.StatusConflict, "nothing_planted", "nothing is planted here"},
	{farm.ErrNotRipe, http.StatusConflict, "not_ripe", "the crop is not ripe yet"},
	{farm.ErrAlreadyRipe, http.StatusConflict, "already_ripe", "the crop is ripe, harvest it"},
	{farm.ErrAlreadyWet, http.StatusConflict, "already_wet", "the soil is still wet"},
	{farming.ErrNoSeeds, http.StatusConflict, "no_seeds", "buy seeds of this crop first"},
	{player.ErrNotEnoughCoins, http.StatusConflict, "not_enough_coins", "not enough coins"},
	{shop.ErrCropLocked, http.StatusConflict, "crop_locked", "reach a higher level to buy this seed"},
	{inventory.ErrNotEnough, http.StatusConflict, "not_enough_items", "not enough items in the barn"},
}

func writeAppError(w http.ResponseWriter, r *http.Request, logger *slog.Logger, err error) {
	for _, e := range appErrors {
		if errors.Is(err, e.err) {
			writeError(w, e.status, e.code, e.message)
			return
		}
	}
	internalError(w, r, logger, err)
}
